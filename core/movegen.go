package core

import (
	"fmt"
	"math/bits"
	"time"
)

// Each constant represents a type of
// move.

const (
	Quiet = iota
	Attack
	AttackEP
	CastleWKS
	CastleWQS
	CastleBKS
	CastleBQS
	KnightPromotion
	BishopPromotion
	RookPromotion
	QueenPromotion
)

const (
	// Three masks to help get the from square,
	// to square, and move type from the 16-bit
	// representation of a mopve.
	FromSquareMask uint16 = 0xFC00
	ToSquareMask   uint16 = 0x3F0
	MoveTypeMask   uint16 = 0xF

	// These masks help determine whether or not the squares between
	// the king and it's rooks are clear for castling
	F1_G1, B1_C1_D1 = 0x600000000000000, 0x7000000000000000
	F8_G8, B8_C8_D8 = 0x6, 0x70

	// Constants representing the squares involved in castling
	A1, C1, D1, E1, F1, G1, H1 = 0, 2, 3, 4, 5, 6, 7
	A8, C8, D8, E8, F8, G8, H8 = 56, 58, 59, 60, 61, 62, 63

	// Size of the transposition table used in perft
	TTPerftSize = 0x100000 * 2
)

// Struct that holds perft entries
type PerftTTEntry struct {
	Hash  uint64
	Depth int
	Nodes uint64
}

// A helper function to create moves. Each move generated
// by our move generator is encoded in 16-bits, where the
// first six bits are the from square, the second 6, are the
// to square, and the last four are the move type (see above).
func MakeMove(from, to, moveType int) uint16 {
	return uint16(from<<10 | to<<4 | moveType)
}

// A helper function to get the from, to, and move type
// from the 16-bit representation of a move.
func GetMoveInfo(move uint16) (int, int, uint16) {
	from := (move & FromSquareMask) >> 10
	to := (move & ToSquareMask) >> 4
	moveType := move & MoveTypeMask
	return int(from), int(to), moveType
}

// Convince functions to only get certian parts of a move
func getMoveFromSq(move uint16) int {
	return int((move & FromSquareMask) >> 10)
}

func getMoveToSq(move uint16) int {
	return int((move & ToSquareMask) >> 4)
}

func getMoveType(move uint16) uint16 {
	return move & MoveTypeMask
}

// A helper function to extract the info from a move represented
// as 16-bits, and display it.
func MoveToStr(move uint16) string {
	from, to, moveType := GetMoveInfo(move)
	promotionType, seperator := "", "-"
	switch moveType {
	case Attack:
		fallthrough
	case AttackEP:
		seperator = "x"
	case KnightPromotion:
		promotionType = "n"
	case BishopPromotion:
		promotionType = "b"
	case RookPromotion:
		promotionType = "r"
	case QueenPromotion:
		promotionType = "q"
	}
	return fmt.Sprintf("%v%v%v%v", PosToCoordinate(int(from)), seperator, PosToCoordinate(int(to)), promotionType)
}

// Compute all legal moves for the given side in the current position
func GenLegalMoves(board *Board, moves *[]uint16) {
	usColor := BlackBB
	enemyColor := WhiteBB
	if board.WhiteToMove {
		usColor = WhiteBB
		enemyColor = BlackBB
	}

	enemyBB := board.PieceBB[enemyColor]
	usBB := board.PieceBB[usColor]
	pawnsBB := board.PieceBB[PawnBB] & usBB
	knightsBB := board.PieceBB[KnightBB] & usBB
	bishopsBB := board.PieceBB[BishopBB] & usBB
	rooksBB := board.PieceBB[RookBB] & usBB
	queensBB := board.PieceBB[QueenBB] & usBB
	kingBB := board.PieceBB[KingBB] & usBB

	checkersBB := attackersOfSquare(board, enemyColor, kingBB, usBB)
	notPinnedMask := ^genPinnedPiecesMoves(board, enemyColor, usColor, kingBB, moves)
	if bits.OnesCount64(checkersBB) == 0 {
		genPawnMoves(board, pawnsBB&notPinnedMask, enemyBB, usBB, moves)
		genKnightMoves(knightsBB&notPinnedMask, enemyBB, usBB, moves)
		genBishopMoves(bishopsBB&notPinnedMask, enemyBB, usBB, moves)
		genRookMoves(rooksBB&notPinnedMask, enemyBB, usBB, moves)
		genQueenMoves(queensBB&notPinnedMask, enemyBB, usBB, moves)
		genKingMoves(board, enemyColor, kingBB, usBB, moves)
		genCastlingMoves(board, enemyColor, usBB, moves)
	} else {
		*moves = (*moves)[:0]
		genCheckEvasionMoves(board, enemyColor, usColor, kingBB, checkersBB, notPinnedMask, moves)
	}
}

// Generate pawn moves for the current side to move.
func genPawnMoves(board *Board, pawnsBB, enemyBB, usBB uint64, moves *[]uint16) {
	if board.WhiteToMove {
		genWhitePawnMoves(board, pawnsBB, enemyBB, usBB, board.EPSquare, moves)
	} else {
		genBlackPawnMoves(board, pawnsBB, enemyBB, usBB, board.EPSquare, moves)
	}
}

// Generate white pawn moves
func genWhitePawnMoves(board *Board, pawnsBB, enemyBB, usBB uint64, epSq int, moves *[]uint16) {
	ourKing := board.PieceBB[KingBB] & board.PieceBB[WhiteBB]
	for pawnsBB != 0 {
		from, _ := popLSB(&pawnsBB)
		pawnOnePush := WhitePawnPushes[from] & ^(usBB | enemyBB)
		pawnPush := pawnOnePush | ((pawnOnePush&MaskRank[Rank3])>>8) & ^(usBB|enemyBB)
		pawnAttacks := WhitePawnAttacks[from]
		for pawnPush != 0 {
			to, _ := popLSB(&pawnPush)
			if to >= 56 && to <= 63 {
				makePromotionMoves(from, to, moves)
				continue
			}
			*moves = append(*moves, MakeMove(from, to, Quiet))
		}
		for pawnAttacks != 0 {
			to, toBB := popLSB(&pawnAttacks)
			if to == epSq {
				// Check to make sure we aren't allowing the edge case
				// of playing an illegal en passant discovered check on
				// ourselves
				capturePos := to - 8
				board.movePiece(from, to)
				board.removePiece(capturePos)
				if !squareIsAttacked(board, BlackBB, ourKing, board.PieceBB[WhiteBB]) {
					*moves = append(*moves, MakeMove(from, to, AttackEP))
				}
				board.movePiece(to, from)
				board.putPiece(PawnBB, BlackBB, capturePos)
			} else if toBB&enemyBB != 0 {
				if to >= 56 && to <= 63 {
					makePromotionMoves(from, to, moves)
					continue
				}
				*moves = append(*moves, MakeMove(from, to, Attack))
			}
		}
	}
}

// Generate black pawn moves
func genBlackPawnMoves(board *Board, pawnsBB, enemyBB, usBB uint64, epSq int, moves *[]uint16) {
	ourKing := board.PieceBB[KingBB] & board.PieceBB[BlackBB]
	for pawnsBB != 0 {
		from, _ := popLSB(&pawnsBB)
		pawnOnePush := BlackPawnPushes[from] & ^(usBB | enemyBB)
		pawnPush := pawnOnePush | ((pawnOnePush&MaskRank[Rank6])<<8) & ^(usBB|enemyBB)
		pawnAttacks := BlackPawnAttacks[from]
		for pawnPush != 0 {
			to, _ := popLSB(&pawnPush)
			if to >= 0 && to <= 7 {
				makePromotionMoves(from, to, moves)
				continue
			}
			*moves = append(*moves, MakeMove(from, to, Quiet))
		}
		for pawnAttacks != 0 {
			to, toBB := popLSB(&pawnAttacks)
			if to == epSq {
				// Check to make sure we aren't allowing the edge case
				// of playing an illegal en passant discovered check on
				// ourselves
				capturePos := to + 8
				board.movePiece(from, to)
				board.removePiece(capturePos)
				if !squareIsAttacked(board, WhiteBB, ourKing, board.PieceBB[BlackBB]) {
					*moves = append(*moves, MakeMove(from, to, AttackEP))
				}
				board.movePiece(to, from)
				board.putPiece(PawnBB, WhiteBB, capturePos)
			} else if toBB&enemyBB != 0 {
				if to >= 0 && to <= 7 {
					makePromotionMoves(from, to, moves)
					continue
				}
				*moves = append(*moves, MakeMove(from, to, Attack))
			}
		}
	}
}

func makePromotionMoves(from, to int, moves *[]uint16) {
	*moves = append(*moves, MakeMove(from, to, KnightPromotion))
	*moves = append(*moves, MakeMove(from, to, BishopPromotion))
	*moves = append(*moves, MakeMove(from, to, RookPromotion))
	*moves = append(*moves, MakeMove(from, to, QueenPromotion))
}

// Generate knight moves
func genKnightMoves(knightsBB, enemyBB, usBB uint64, moves *[]uint16) {
	for knightsBB != 0 {
		from, _ := popLSB(&knightsBB)
		knightMoves := KnightMoves[from] & ^usBB
		for knightMoves != 0 {
			to, toBB := popLSB(&knightMoves)
			moveType := Quiet
			if toBB&enemyBB != 0 {
				moveType = Attack
			}
			*moves = append(*moves, MakeMove(from, to, moveType))
		}
	}
}

// Generate bishop moves
func genBishopMoves(bishopsBB, enemyBB, usBB uint64, moves *[]uint16) {
	for bishopsBB != 0 {
		from, fromBB := popLSB(&bishopsBB)
		bishopMoves := genIntercardianlMovesBB(fromBB, enemyBB|usBB) & ^usBB
		for bishopMoves != 0 {
			to, toBB := popLSB(&bishopMoves)
			moveType := Quiet
			if toBB&enemyBB != 0 {
				moveType = Attack
			}
			*moves = append(*moves, MakeMove(from, to, moveType))
		}
	}
}

// Generate rook moves
func genRookMoves(rooksBB, enemyBB, usBB uint64, moves *[]uint16) {
	for rooksBB != 0 {
		from, fromBB := popLSB(&rooksBB)
		bishopMoves := genCardianlMovesBB(fromBB, enemyBB|usBB) & ^usBB
		for bishopMoves != 0 {
			to, toBB := popLSB(&bishopMoves)
			moveType := Quiet
			if toBB&enemyBB != 0 {
				moveType = Attack
			}
			*moves = append(*moves, MakeMove(from, to, moveType))
		}
	}
}

// Generate queen moves
func genQueenMoves(queensBB, enemyBB, usBB uint64, moves *[]uint16) {
	genBishopMoves(queensBB, enemyBB, usBB, moves)
	genRookMoves(queensBB, enemyBB, usBB, moves)
}

// Generate king moves
func genKingMoves(board *Board, enemyColor int, kingBB, usBB uint64, moves *[]uint16) {
	from := getLSBPos(kingBB)
	enemyBB := board.PieceBB[enemyColor]
	kingMoves := KingMoves[from] & ^usBB
	for kingMoves != 0 {
		to, toBB := popLSB(&kingMoves)
		if squareIsAttacked(board, enemyColor, toBB, usBB) {
			continue
		}
		moveType := Quiet
		if toBB&enemyBB != 0 {
			moveType = Attack
		}
		*moves = append(*moves, MakeMove(from, to, moveType))
	}
}

// Generate castling moves
func genCastlingMoves(board *Board, enemyColor int, usBB uint64, moves *[]uint16) {
	allPieces := board.PieceBB[enemyColor] | usBB
	if board.WhiteToMove {
		if board.CastlingRights&WhiteKingside != 0 && allPieces&F1_G1 == 0 &&
			(!squareIsAttacked(board, enemyColor, setSingleBit(5), usBB) &&
				!squareIsAttacked(board, enemyColor, setSingleBit(6), usBB)) {
			*moves = append(*moves, MakeMove(4, 6, CastleWKS))
		}
		if board.CastlingRights&WhiteQueenside != 0 && allPieces&B1_C1_D1 == 0 &&
			(!squareIsAttacked(board, enemyColor, setSingleBit(2), usBB) &&
				!squareIsAttacked(board, enemyColor, setSingleBit(3), usBB)) {
			*moves = append(*moves, MakeMove(4, 2, CastleWQS))
		}
	} else {
		if board.CastlingRights&BlackKingside != 0 && allPieces&F8_G8 == 0 &&
			(!squareIsAttacked(board, enemyColor, setSingleBit(61), usBB) &&
				!squareIsAttacked(board, enemyColor, setSingleBit(62), usBB)) {
			*moves = append(*moves, MakeMove(60, 62, CastleBKS))
		}
		if board.CastlingRights&BlackQueenside != 0 && allPieces&B8_C8_D8 == 0 &&
			(!squareIsAttacked(board, enemyColor, setSingleBit(58), usBB) &&
				!squareIsAttacked(board, enemyColor, setSingleBit(59), usBB)) {
			*moves = append(*moves, MakeMove(60, 58, CastleBQS))
		}
	}
}

// If the king is in check, then this special check evasion function is called that calculates
// the few moves the color to move has. The basic algorithm is first to check if the king is in
// double or single check. If double check, the king has to move. If single check and the checker
// is a knight, then the only choices are to move the king or capture the knight. Otherwise, then
// the options are to block, capture, or move the king from the slider piece giving check.
func genCheckEvasionMoves(board *Board, enemyColor, usColor int, kingBB, checkersBB, notPinnedMask uint64, moves *[]uint16) {
	kingPos := getLSBPos(kingBB)
	usBB := board.PieceBB[usColor]
	enemyBB := board.PieceBB[enemyColor]

	ourPawns := board.PieceBB[PawnBB] & usBB & notPinnedMask
	var pseduolegalPawnMoves []uint16
	genPawnMoves(board, ourPawns, enemyBB, usBB, &pseduolegalPawnMoves)

	// We need to remove the king from our board when calculating check evasion moves,
	// so that enemy sliders can "xray" the king and show that they attack the squares
	// *behind* the king as well, so the king doesn't just slide back still in check.
	genKingMoves(board, enemyColor, kingBB, usBB & ^kingBB, moves)

	if bits.OnesCount64(checkersBB) > 1 {
		return
	}
	checkerPos := getLSBPos(checkersBB)
	checkerType := GetPieceType(board.Pieces[checkerPos])

	if checkerType == KnightBB {
		for _, move := range pseduolegalPawnMoves {
			_, to, moveType := GetMoveInfo(move)
			if moveType == Attack && to == checkerPos {
				*moves = append(*moves, move)
			}
		}

		sqProtectorsBB := attackersOfSquare(board, usColor, checkersBB, enemyBB) & notPinnedMask

		for sqProtectorsBB != 0 {
			protectorPos, _ := popLSB(&sqProtectorsBB)
			protectorType := GetPieceType(board.Pieces[protectorPos])
			if protectorType != KingBB && protectorType != PawnBB {
				*moves = append(*moves, MakeMove(protectorPos, checkerPos, Attack))
			}
		}
	} else {
		betweenBB := LinesBewteen[kingPos][checkerPos]
		for _, move := range pseduolegalPawnMoves {
			_, to, moveType := GetMoveInfo(move)
			if setSingleBit(to)&betweenBB != 0 {
				*moves = append(*moves, move)
			} else if moveType == AttackEP {
				capturePos := to + 8
				if usColor == WhiteBB {
					capturePos = to - 8
				}
				if capturePos == checkerPos {
					*moves = append(*moves, move)
				}
			} else if moveType == Attack && to == checkerPos {
				*moves = append(*moves, move)
			}
		}

		for betweenBB != 0 {
			sqPos, sqBB := popLSB(&betweenBB)
			ourSqProtectors := attackersOfSquare(board, usColor, sqBB, enemyBB)
			ourSqProtectors &= notPinnedMask

			for ourSqProtectors != 0 {
				protectorPos, _ := popLSB(&ourSqProtectors)
				protectorType := GetPieceType(board.Pieces[protectorPos])
				if protectorType != KingBB && protectorType != PawnBB {
					moveType := Quiet
					if sqPos == checkerPos {
						moveType = Attack
					}
					*moves = append(*moves, MakeMove(protectorPos, sqPos, moveType))
				}
			}
		}
	}
}

// Find which pieces are pinned in the current board state, and generate any possible
// moves they have. Return a bitboard containing the pinned pieces so that they can
// be removed from the bitboards passed into generating normal moves, since they're
// moves have already been considered.
func genPinnedPiecesMoves(board *Board, enemyColor, usColor int, kingBB uint64, moves *[]uint16) (pinnedBB uint64) {
	enemyBB := board.PieceBB[enemyColor]
	enemyBishops := enemyBB & board.PieceBB[BishopBB]
	enemyRooks := enemyBB & board.PieceBB[RookBB]
	enemyQueens := enemyBB & board.PieceBB[QueenBB]
	usBB := board.PieceBB[usColor]
	pinnersBB := (genIntercardianlMovesBB(kingBB, enemyBB)&(enemyBishops|enemyQueens) |
		genCardianlMovesBB(kingBB, enemyBB)&(enemyRooks|enemyQueens))
	kingPos := getLSBPos(kingBB)
	for pinnersBB != 0 {
		pinnerPos, pinnerBB := popLSB(&pinnersBB)
		possiblyPinnedBB := LinesBewteen[kingPos][pinnerPos] & usBB
		if bits.OnesCount64(possiblyPinnedBB) == 1 {
			pinnedBB |= possiblyPinnedBB
			pinnedPos := getLSBPos(possiblyPinnedBB)

			pinnerType := GetPieceType(board.Pieces[pinnerPos])
			pinnedType := GetPieceType(board.Pieces[pinnedPos])
			pinnerRayDirection := LinesBetweenDirections[kingPos][pinnerPos]
			rayBetween := LinesBewteen[kingPos][pinnerPos] & ^possiblyPinnedBB

			if pinnerType == BishopBB && pinnedType == BishopBB {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if pinnerType == RookBB && pinnedType == RookBB {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if pinnerType == QueenBB && pinnedType == QueenBB {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if (pinnerType == BishopBB || pinnerType == RookBB) && pinnedType == QueenBB {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if pinnerType == QueenBB && pinnedType == BishopBB && directionIsIntercardinal(pinnerRayDirection) {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if pinnerType == QueenBB && pinnedType == RookBB && directionIsCardinal(pinnerRayDirection) {
				genMovesFromBB(pinnedPos, rayBetween, enemyBB, moves)
			} else if (pinnerType == RookBB || pinnerType == QueenBB) && pinnedType == PawnBB &&
				directionIsNorthOrSouth(pinnerRayDirection) {
				pawnPush := BlackPawnPushes[pinnedPos] & ^(enemyBB | usBB)
				pawnPush |= ((pawnPush & MaskRank[Rank6]) << 8) & ^(enemyBB | usBB)
				if usColor == WhiteBB {
					pawnPush = WhitePawnPushes[pinnedPos] & ^(enemyBB | usBB)
					pawnPush |= ((pawnPush & MaskRank[Rank3]) >> 8) & ^(enemyBB | usBB)
				}
				genMovesFromBB(pinnedPos, pawnPush, 0, moves)
			} else if (pinnerType == BishopBB || pinnerType == QueenBB) && pinnedType == PawnBB &&
				directionIsIntercardinal(pinnerRayDirection) {
				pawnAttacks := BlackPawnAttacks[pinnedPos]
				if usColor == WhiteBB {
					pawnAttacks = WhitePawnAttacks[pinnedPos]
				}
				if pawnAttacks&pinnerBB != 0 {
					if usColor == WhiteBB && pinnerPos >= 56 && pinnerPos <= 63 {
						makePromotionMoves(pinnedPos, pinnerPos, moves)
					} else if usColor == BlackBB && pinnerPos >= 0 && pinnerPos <= 7 {
						makePromotionMoves(pinnedPos, pinnerPos, moves)
					} else {
						*moves = append(*moves, MakeMove(pinnedPos, pinnerPos, Attack))
					}
				}

			}
		}
	}
	return pinnedBB
}

// A helper function used in genPinnedPiecesMoves to generate
// moves from a bitboard with multiple bits set.
func genMovesFromBB(from int, movesBB, enemyBB uint64, moves *[]uint16) {
	for movesBB != 0 {
		to, toBB := popLSB(&movesBB)
		moveType := Quiet
		if toBB&enemyBB != 0 {
			moveType = Attack
		}
		*moves = append(*moves, MakeMove(from, to, moveType))
	}
}

// Helper function for testing whether or not a Direction is cardinal
func directionIsCardinal(direction Direction) bool {
	return direction == North || direction == South || direction == East || direction == West
}

// Helper function for testing whether or not a Direction is intercardinal
func directionIsIntercardinal(direction Direction) bool {
	return direction == NorthEast || direction == SouthEast || direction == NorthWest || direction == SouthWest
}

// Helper function for testing whether or not a Direction is north or south
func directionIsNorthOrSouth(direction Direction) bool {
	return direction == North || direction == South
}

// Compute a bitboard representing the enemy attackers of a particular
// square. The algorithm used to find attacking pieces is to sit a super-
// piece on the square of interest, and generating cardinal, intercardinal,
// and knight rays from the square. If any of these rays interesect with the
// enemyBB, then that intersection is an attacker.s
func attackersOfSquare(board *Board, enemyColor int, squareBB, usBB uint64) (attackers uint64) {
	enemyBB := board.PieceBB[enemyColor]
	enemyBishop := enemyBB & board.PieceBB[BishopBB]
	enemyRook := enemyBB & board.PieceBB[RookBB]
	enemyQueen := enemyBB & board.PieceBB[QueenBB]
	enemyKing := board.PieceBB[enemyColor] & board.PieceBB[KingBB]
	enemyKnights := enemyBB & board.PieceBB[KnightBB]
	enemyPawns := enemyBB & board.PieceBB[PawnBB]

	squarePos := getLSBPos(squareBB)
	intercardinalRays := genIntercardianlMovesBB(squareBB, enemyBB|usBB)
	cardinalRaysRays := genCardianlMovesBB(squareBB, enemyBB|usBB)

	attackers |= intercardinalRays & (enemyBishop | enemyQueen)
	attackers |= cardinalRaysRays & (enemyRook | enemyQueen)
	attackers |= KnightMoves[squarePos] & enemyKnights
	attackers |= KingMoves[squarePos] & enemyKing
	if enemyColor == WhiteBB {
		attackers |= BlackPawnAttacks[squarePos] & enemyPawns
	} else {
		attackers |= WhitePawnAttacks[squarePos] & enemyPawns
	}
	return attackers
}

// Similar to attackersOfSquare except instead of returning a bitboard of
// attackers of a certian square, this function only returns whether or
// not the function is being attacked. Thus, this function is more efficent
// when we only care about whether or not a square is attacked since it can
// return early.
func squareIsAttacked(board *Board, enemyColor int, squareBB, usBB uint64) bool {
	enemyBB := board.PieceBB[enemyColor]
	enemyBishop := board.PieceBB[enemyColor] & board.PieceBB[BishopBB]
	enemyRook := board.PieceBB[enemyColor] & board.PieceBB[RookBB]
	enemyQueen := board.PieceBB[enemyColor] & board.PieceBB[QueenBB]
	enemyKnights := board.PieceBB[enemyColor] & board.PieceBB[KnightBB]
	enemyKing := board.PieceBB[enemyColor] & board.PieceBB[KingBB]
	enemyPawns := board.PieceBB[enemyColor] & board.PieceBB[PawnBB]

	squarePos := getLSBPos(squareBB)
	intercardinalRays := genIntercardianlMovesBB(squareBB, enemyBB|usBB)
	cardinalRaysRays := genCardianlMovesBB(squareBB, enemyBB|usBB)

	if intercardinalRays&(enemyBishop|enemyQueen) != 0 {
		return true
	}
	if cardinalRaysRays&(enemyRook|enemyQueen) != 0 {
		return true
	}
	if KnightMoves[squarePos]&enemyKnights != 0 {
		return true
	}
	if KingMoves[squarePos]&enemyKing != 0 {
		return true
	}
	if enemyColor == WhiteBB && BlackPawnAttacks[squarePos]&enemyPawns != 0 {
		return true
	} else if enemyColor == BlackBB && WhitePawnAttacks[squarePos]&enemyPawns != 0 {
		return true
	}
	return false
}

// Given a bitboard with a single bit set at a slider's location,
// and an occupancy bitboard containing the bits of all white and
// black pieces, this function calculates all possible moves for the
// slider in the intercardinal directions. The algorithm used is
// "Hyperbola Quintessence", created by Gerd Isenberg, and the specfic
// formula used here was given by Johnathan, from Logic Crazy Chess.
func genIntercardianlMovesBB(sliderBB, occupiedBB uint64) uint64 {
	sliderPos := getLSBPos(sliderBB)
	diagonalMask := MaskDiagonal[(sliderPos%8)-(sliderPos/8)+7]
	antidiagonalMask := MaskAntidiagonal[14-((sliderPos/8)+(sliderPos%8))]

	rhs := bits.Reverse64(bits.Reverse64((occupiedBB & diagonalMask)) - (2 * bits.Reverse64(sliderBB)))
	lhs := (occupiedBB & diagonalMask) - 2*sliderBB
	diagonalMoves := (rhs ^ lhs) & diagonalMask

	rhs = bits.Reverse64(bits.Reverse64((occupiedBB & antidiagonalMask)) - (2 * bits.Reverse64(sliderBB)))
	lhs = (occupiedBB & antidiagonalMask) - 2*sliderBB
	antidiagonalMoves := (rhs ^ lhs) & antidiagonalMask

	return diagonalMoves | antidiagonalMoves
}

// Given a bitboard with a single bit set at a slider's location,
// and an occupancy bitboard containing the bits of all white and
// black pieces, this function calculates all possible moves for the
// slider in the cardinal directions. The algorithm used is
// "Hyperbola Quintessence", created by Gerd Isenberg, and the specfic
// formula used here was given by Johnathan, from Logic Crazy Chess.
func genCardianlMovesBB(sliderBB, occupiedBB uint64) uint64 {
	sliderPos := getLSBPos(sliderBB)
	fileMask := MaskFile[sliderPos%8]
	rankMask := MaskRank[sliderPos/8]

	rhs := bits.Reverse64(bits.Reverse64((occupiedBB & rankMask)) - (2 * bits.Reverse64(sliderBB)))
	lhs := (occupiedBB & rankMask) - 2*sliderBB
	eastWestMoves := (rhs ^ lhs) & rankMask

	rhs = bits.Reverse64(bits.Reverse64((occupiedBB & fileMask)) - (2 * bits.Reverse64(sliderBB)))
	lhs = (occupiedBB & fileMask) - 2*sliderBB
	northSouthMoves := (rhs ^ lhs) & fileMask

	return northSouthMoves | eastWestMoves
}

// A global variable used for debugging and measuring the effiency
// of perft and divideperft by recording the number of transposition
// table hits.
var TTHits int

// Explore the move tree up to depth, and return the total
// number of nodes explored.  This function is used to
// debug move generation and ensure it is working by comparing
// the results to the known results of other engines.
func perft(board *Board, depth int, ttable *[TTPerftSize]PerftTTEntry) uint64 {
	moves := make([]uint16, 0, 220)
	GenLegalMoves(board, &moves)
	if depth == 1 {
		return uint64(len(moves))
	}

	entry := ttable[board.Hash%TTPerftSize]
	if entry.Hash == board.Hash && entry.Depth == depth {
		TTHits++
		return entry.Nodes
	}

	var nodes uint64
	for _, move := range moves {
		board.DoMove(&move, true)
		nodes += perft(board, depth-1, ttable)
		board.UndoMove(&move)
	}
	ttable[board.Hash%TTPerftSize] = PerftTTEntry{Hash: board.Hash, Depth: depth, Nodes: nodes}
	return nodes
}

// Similar to perft, but prints the nodes for each move at depth-1.
// This makes it more practical to use for actually debugging, since
// one can see where their program deviates from the correct value
// and recursivley explore a position using smaller depth values until
// a bug is found.
func dividePerft(board *Board, depth, divdeAt int, ttable *[TTPerftSize]PerftTTEntry) uint64 {
	if depth == 0 {
		return 1
	}

	entry := ttable[board.Hash%TTPerftSize]
	if entry.Hash == board.Hash && entry.Depth == depth {
		TTHits++
		return entry.Nodes
	}

	moves := make([]uint16, 0, 220)
	var nodes uint64
	GenLegalMoves(board, &moves)
	for _, move := range moves {
		board.DoMove(&move, true)
		moveNodes := dividePerft(board, depth-1, divdeAt, ttable)
		if depth == divdeAt {
			fmt.Printf("%v: %v\n", MoveToStr(move), moveNodes)
		}
		nodes += moveNodes
		board.UndoMove(&move)
	}
	ttable[board.Hash%TTPerftSize] = PerftTTEntry{Hash: board.Hash, Depth: depth, Nodes: nodes}
	return nodes
}

// A wrapper for a perft function with no extra frills,
// just returns the total node count. Used in testing
// in the tests package.
func RawPerft(board *Board, depth int, ttable *[TTPerftSize]PerftTTEntry) uint64 {
	return perft(board, depth, ttable)
}

// A convient wrapper around perft
func Perft(board *Board, depth int, ttable *[TTPerftSize]PerftTTEntry) {
	defer timeit(time.Now())
	totalNodes := perft(board, depth, ttable)
	fmt.Println("total nodes:", totalNodes)
	fmt.Println("Transpositon table hits:", TTHits)
}

// A convient wrapper around divdePerft
func DividePerft(board *Board, depth int, ttable *[TTPerftSize]PerftTTEntry) {
	defer timeit(time.Now())
	totalNodes := dividePerft(board, depth, depth, ttable)
	fmt.Println("total nodes:", totalNodes)
	fmt.Println("Transpositon table hits:", TTHits)
}
