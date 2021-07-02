package core

import (
	"fmt"
	"math/bits"
	"strings"
	"unicode"
)

const (
	// Constant indexes into a list of bitboards
	// in a Board instance which contains a bitboard
	// for each type of pice and its color.
	PawnBB = iota
	KnightBB
	BishopBB
	RookBB
	QueenBB
	KingBB
	WhiteBB
	BlackBB

	// Piece constants, structured in such a way that
	// we can easily determine the type and color of a
	// piece on our mailbox board.
	Pawn   uint8 = 0x0
	Knight uint8 = 0x20
	Bishop uint8 = 0x40
	Rook   uint8 = 0x60
	Queen  uint8 = 0x80
	King   uint8 = 0xA0

	White   uint8 = 0x18
	Black   uint8 = 0x1C
	NoPiece uint8 = 0x0

	PieceMask uint8 = 0xE0
	ColorMask uint8 = 0x1C

	// 8-bit values representing the four castling
	// rights. A different bit is set for each right.
	WhiteKingside  uint8 = 0x80
	WhiteQueenside uint8 = 0x40
	BlackKingside  uint8 = 0x20
	BlackQueenside uint8 = 0x10

	// Constant representing no en passant square
	NoEPSquare = -1

	// Starting FEN position
	FENStartPosition = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	// A FEN position called kiwipete. It's a particular
	// tricky position for chess move generators and serves
	// as a good test for the move generator.
	FENKiwiPete = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

	// A constant representing how many pieces are left before the engine
	// switches into an endgame mode.
	EndgameThreshold = 12
)

// A structure for holding data concering the current position
// that needs to be saved.
type UndoInfo struct {
	EPSquare        int
	CastlingRights  uint8
	HalfMoveClock   int
	HalfMoveCounter int
	FullMoveCounter int
	CaptureSq       uint8
	FromSq          uint8
}

// The primary internal representation of the board
type Board struct {
	// List of bitboards for each type of piece and color
	PieceBB [8]uint64

	// Mailbox representation of the board. Using a hybrid approach
	// of bitboards and mailbox representations allow for cleaner,
	// and more efficent code. The bitboard internal representation
	// is used for move generation, and the mailbox representation is
	// used when making or unmaking a move, as it allows for quick and
	// easy lookup of type and color of a piece on a particular square.
	Pieces [64]uint8

	// Boolean representing the current side to move
	WhiteToMove bool

	// The current en passant target square
	EPSquare int

	// The half move *counter* (not clock). Used to update the full
	// move counter correctly. The counter begins at 0, and each time
	// a half move is made, it's incremented. When it reaches an even
	// number, a full move has been made, and so the full move counter
	// is updated,
	HalfMoveCounter int
	FullMoveCounter int

	// The half move clock which is incremnted every half move
	// and serves to implement the fifty-move rule. Reset every
	// capture or pawn push.
	HalfMoveClock int

	// A different bit set in this variable
	// represents the four castling rights:
	//
	// Castling White Kingside = 1st bit
	// Castling White Queenside = 2nd bit
	// Castling Black Kingside = 3rd bit
	// Castling Black Queenside  = 4th bit
	CastlingRights uint8

	// The Zobrist hashing representing the current
	// board state. This is initalized from a loaded
	// fen string and updated incrementally as moves
	// are made and unmade from the board.
	Hash uint64

	// An array that holds UndoInfo structures (see above)
	// concering positions at different game plys. The current
	// ply is kept track of by gamePly
	undoInfoList [200]UndoInfo
	gamePly      int
}

// Push an undoInfo object to the stack
func (board *Board) saveState(undoInfo UndoInfo) {
	board.gamePly++
	board.undoInfoList[board.gamePly] = undoInfo
}

// Pop an undoInfo object from the stack
func (board *Board) popState() UndoInfo {
	undoInfo := board.undoInfoList[board.gamePly]
	board.gamePly--
	return undoInfo
}

// Do a move to the internal boards. Save the state of the
// board if requested by the caller before modyifying the
// board.
func (board *Board) DoMove(move *uint16, saveState bool) {
	from, to, moveType := GetMoveInfo(*move)
	usColor := BlackBB
	if board.WhiteToMove {
		usColor = WhiteBB
	}

	undoInfo := UndoInfo{
		CastlingRights:  board.CastlingRights,
		EPSquare:        board.EPSquare,
		HalfMoveClock:   board.HalfMoveClock,
		HalfMoveCounter: board.HalfMoveCounter,
		FullMoveCounter: board.FullMoveCounter,
		CaptureSq:       board.Pieces[to],
		FromSq:          board.Pieces[from],
	}

	// Remove the current en passant square if any
	if board.EPSquare != NoEPSquare && isValidZobristEPSq(board, board.EPSquare) {
		board.Hash ^= getEPFileHash(board.EPSquare)
	}
	board.EPSquare = NoEPSquare

	switch moveType {
	case CastleWKS:
		board.movePiece(E1, G1)
		board.movePiece(H1, F1)
	case CastleWQS:
		board.movePiece(E1, C1)
		board.movePiece(A1, D1)
	case CastleBKS:
		board.movePiece(E8, G8)
		board.movePiece(H8, F8)
	case CastleBQS:
		board.movePiece(E8, C8)
		board.movePiece(A8, D8)
	case KnightPromotion:
		board.removePiece(from)
		if undoInfo.CaptureSq != NoPiece {
			board.removePiece(to)
		}
		board.putPiece(KnightBB, usColor, to)
	case BishopPromotion:
		board.removePiece(from)
		if undoInfo.CaptureSq != NoPiece {
			board.removePiece(to)
		}
		board.putPiece(BishopBB, usColor, to)
	case RookPromotion:
		board.removePiece(from)
		if undoInfo.CaptureSq != NoPiece {
			board.removePiece(to)
		}
		board.putPiece(RookBB, usColor, to)
	case QueenPromotion:
		board.removePiece(from)
		if undoInfo.CaptureSq != NoPiece {
			board.removePiece(to)
		}
		board.putPiece(QueenBB, usColor, to)
	case AttackEP:
		capturePos := to + 8
		if usColor == WhiteBB {
			capturePos = to - 8
		}
		undoInfo.CaptureSq = board.Pieces[capturePos]
		board.removePiece(capturePos)
		board.movePiece(from, to)
	case Attack:
		if GetPieceType(board.Pieces[to]) == KingBB {
			board.PrintBoard()
			Print2dBitboard(board.PieceBB[WhiteBB])
			Print2dBitboard(board.PieceBB[BlackBB])
			panic("illegal king capture")
		}
		board.removePiece(to)
		board.movePiece(from, to)
	case Quiet:
		board.movePiece(from, to)
	}

	// Update the half move clock and counter
	board.HalfMoveClock++
	board.HalfMoveCounter++

	// Reset the half move clock
	if GetPieceType(undoInfo.FromSq) == PawnBB || moveType == Attack || moveType == AttackEP {
		board.HalfMoveClock = 0
	}

	// If the half move counter is even, then a
	// full move has been reached and we need to
	// increment the counter by one
	if board.HalfMoveCounter%2 == 0 {
		board.FullMoveCounter++
	}

	// Set the new en passant target square if any
	if GetPieceType(undoInfo.FromSq) == PawnBB && abs(from-to) == 16 {
		board.EPSquare = to + 8
		if usColor == WhiteBB {
			board.EPSquare = to - 8
		}
	}

	// Update the castling rights
	if GetPieceType(board.Pieces[E1]) != KingBB {
		board.CastlingRights &= ^WhiteKingside
		board.CastlingRights &= ^WhiteQueenside
	}
	if GetPieceType(board.Pieces[H1]) != RookBB {
		board.CastlingRights &= ^WhiteKingside
	}
	if GetPieceType(board.Pieces[A1]) != RookBB {
		board.CastlingRights &= ^WhiteQueenside
	}
	if GetPieceType(board.Pieces[E8]) != KingBB {
		board.CastlingRights &= ^BlackKingside
		board.CastlingRights &= ^BlackQueenside
	}
	if GetPieceType(board.Pieces[H8]) != RookBB {
		board.CastlingRights &= ^BlackKingside
	}
	if GetPieceType(board.Pieces[A8]) != RookBB {
		board.CastlingRights &= ^BlackQueenside
	}

	// Only update the hash castling rights if they were
	// changed!
	if board.CastlingRights != undoInfo.CastlingRights {
		if undoInfo.CastlingRights&WhiteKingside != 0 &&
			board.CastlingRights&WhiteKingside == 0 {

			board.Hash ^= Random64[CastleWKSHash]
		}
		if undoInfo.CastlingRights&WhiteQueenside != 0 &&
			board.CastlingRights&WhiteQueenside == 0 {
			board.Hash ^= Random64[CastleWQSHash]
		}
		if undoInfo.CastlingRights&BlackKingside != 0 &&
			board.CastlingRights&BlackKingside == 0 {
			board.Hash ^= Random64[CastleBKSHash]
		}
		if undoInfo.CastlingRights&BlackQueenside != 0 &&
			board.CastlingRights&BlackQueenside == 0 {
			board.Hash ^= Random64[CastleBQSHash]
		}
	}

	board.WhiteToMove = !board.WhiteToMove
	board.Hash ^= Random64[SideToMove]

	// Add the new en passant square if any
	if board.EPSquare != NoEPSquare && isValidZobristEPSq(board, board.EPSquare) {
		board.Hash ^= getEPFileHash(board.EPSquare)
	}

	if saveState {
		board.saveState(undoInfo)
	}
}

// Undo a move to the interal boards
func (board *Board) UndoMove(move *uint16) {
	undoInfo := board.popState()
	board.HalfMoveClock = undoInfo.HalfMoveClock
	board.HalfMoveCounter = undoInfo.HalfMoveCounter
	board.FullMoveCounter = undoInfo.FullMoveCounter
	board.WhiteToMove = !board.WhiteToMove
	board.Hash ^= Random64[SideToMove]

	from, to, moveType := GetMoveInfo(*move)
	usColor := BlackBB
	if board.WhiteToMove {
		usColor = WhiteBB
	}

	// Only update the hash castling rights if they were
	// changed!
	if board.CastlingRights != undoInfo.CastlingRights {
		if undoInfo.CastlingRights&WhiteKingside != 0 &&
			board.CastlingRights&WhiteKingside == 0 {
			board.Hash ^= Random64[CastleWKSHash]
		}
		if undoInfo.CastlingRights&WhiteQueenside != 0 &&
			board.CastlingRights&WhiteQueenside == 0 {
			board.Hash ^= Random64[CastleWQSHash]
		}
		if undoInfo.CastlingRights&BlackKingside != 0 &&
			board.CastlingRights&BlackKingside == 0 {
			board.Hash ^= Random64[CastleBKSHash]
		}
		if undoInfo.CastlingRights&BlackQueenside != 0 &&
			board.CastlingRights&BlackQueenside == 0 {
			board.Hash ^= Random64[CastleBQSHash]
		}
	}
	board.CastlingRights = undoInfo.CastlingRights

	// Remove the current en passant square if any
	if board.EPSquare != NoEPSquare && isValidZobristEPSq(board, board.EPSquare) {
		board.Hash ^= getEPFileHash(board.EPSquare)
	}
	board.EPSquare = undoInfo.EPSquare

	switch moveType {
	case CastleWKS:
		board.movePiece(G1, E1)
		board.movePiece(F1, H1)
	case CastleWQS:
		board.movePiece(C1, E1)
		board.movePiece(D1, A1)
	case CastleBKS:
		board.movePiece(G8, E8)
		board.movePiece(F8, H8)
	case CastleBQS:
		board.movePiece(C8, E8)
		board.movePiece(D8, A8)
	case KnightPromotion:
		fallthrough
	case BishopPromotion:
		fallthrough
	case RookPromotion:
		fallthrough
	case QueenPromotion:
		board.removePiece(to)
		if undoInfo.CaptureSq != NoPiece {
			board.putPiece(GetPieceType(undoInfo.CaptureSq), getPieceColor(undoInfo.CaptureSq), to)
		}
		board.putPiece(PawnBB, usColor, from)
	case AttackEP:
		capturePos := to + 8
		if usColor == WhiteBB {
			capturePos = to - 8
		}
		board.movePiece(to, from)
		board.putPiece(PawnBB, getPieceColor(undoInfo.CaptureSq), capturePos)
	case Attack:
		board.removePiece(to)
		board.putPiece(GetPieceType(undoInfo.CaptureSq), getPieceColor(undoInfo.CaptureSq), to)
		board.putPiece(GetPieceType(undoInfo.FromSq), usColor, from)
	case Quiet:
		board.movePiece(to, from)
	}

	// Add back the old en passant square if any
	if undoInfo.EPSquare != NoEPSquare && isValidZobristEPSq(board, undoInfo.EPSquare) {
		board.Hash ^= getEPFileHash(board.EPSquare)
	}
}

// Put a piece from the given square to the given square.
// For this function, the move is gureenteed to be quiet.
func (board *Board) movePiece(from, to int) {
	piece := board.Pieces[from]
	pieceType, pieceColor := GetPieceType(piece), getPieceColor(piece)

	clearBit(&board.PieceBB[pieceType], from)
	clearBit(&board.PieceBB[pieceColor], from)
	board.Hash ^= getPieceHash(piece, from)

	setBit(&board.PieceBB[pieceType], to)
	setBit(&board.PieceBB[pieceColor], to)
	board.Hash ^= getPieceHash(piece, to)

	board.Pieces[from] = NoPiece
	board.Pieces[to] = uint8((pieceType << 5) | (pieceColor << 2))
}

// Put the piece given on the given square
func (board *Board) putPiece(pieceType, pieceColor int, to int) {
	setBit(&board.PieceBB[pieceType], to)
	setBit(&board.PieceBB[pieceColor], to)
	board.Pieces[to] = uint8((pieceType << 5) | (pieceColor << 2))
	board.Hash ^= getPieceHash(board.Pieces[to], to)
}

// Remove the piece given on the given square.
func (board *Board) removePiece(from int) {
	piece := board.Pieces[from]
	pieceType, pieceColor := GetPieceType(piece), getPieceColor(piece)
	clearBit(&board.PieceBB[pieceType], from)
	clearBit(&board.PieceBB[pieceColor], from)
	board.Hash ^= getPieceHash(piece, from)
	board.Pieces[from] = NoPiece
}

func (board *Board) LoadFEN(fen string) {
	board.PieceBB = [8]uint64{}
	board.Pieces = [64]uint8{}
	board.WhiteToMove = true
	board.CastlingRights = 0
	board.EPSquare = NoEPSquare
	board.undoInfoList = [200]UndoInfo{}
	board.gamePly = -1

	fenFields := strings.Fields(fen)
	if len(fenFields) > 6 {
		panic("Invalid FEN position")
	}

	pieces := fenFields[0]
	turn := fenFields[1]
	castling := fenFields[2]
	epSq := fenFields[3]
	halfMove := fenFields[4]
	fullMove := fenFields[5]

	board.HalfMoveClock = int(halfMove[0] - '0')
	board.FullMoveCounter = int(fullMove[0] - '0')

	for index, square := 0, 56; index < len(pieces); index++ {
		char := pieces[index]
		switch char {
		case 'p':
			setBit(&board.PieceBB[PawnBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = Pawn | Black
			square++
		case 'n':
			setBit(&board.PieceBB[KnightBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = Knight | Black
			square++
		case 'b':
			setBit(&board.PieceBB[BishopBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = Bishop | Black
			square++
		case 'r':
			setBit(&board.PieceBB[RookBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = Rook | Black
			square++
		case 'q':
			setBit(&board.PieceBB[QueenBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = Queen | Black
			square++
		case 'k':
			setBit(&board.PieceBB[KingBB], square)
			setBit(&board.PieceBB[BlackBB], square)
			board.Pieces[square] = King | Black
			square++
		case 'P':
			setBit(&board.PieceBB[PawnBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = Pawn | White
			square++
		case 'N':
			setBit(&board.PieceBB[KnightBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = Knight | White
			square++
		case 'B':
			setBit(&board.PieceBB[BishopBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = Bishop | White
			square++
		case 'R':
			setBit(&board.PieceBB[RookBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = Rook | White
			square++
		case 'Q':
			setBit(&board.PieceBB[QueenBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = Queen | White
			square++
		case 'K':
			setBit(&board.PieceBB[KingBB], square)
			setBit(&board.PieceBB[WhiteBB], square)
			board.Pieces[square] = King | White
			square++
		case '/':
			square -= 16
		case '1', '2', '3', '4', '5', '6', '7', '8':
			square += charToDigit(char)
		}
	}

	if turn == "b" {
		board.WhiteToMove = false
	}

	if epSq != "-" {
		board.EPSquare = CoordinateToPos(epSq)
	}

	if castling != "-" {
		if strings.Contains(castling, "K") {
			board.CastlingRights |= WhiteKingside
		}
		if strings.Contains(castling, "Q") {
			board.CastlingRights |= WhiteQueenside
		}
		if strings.Contains(castling, "k") {
			board.CastlingRights |= BlackKingside
		}
		if strings.Contains(castling, "q") {
			board.CastlingRights |= BlackQueenside
		}
	}
	board.Hash = initZobristHash(board)
}

// Determine when the endgame has been reached
func (board *Board) IsEndgame() bool {
	return bits.OnesCount64(board.PieceBB[WhiteBB]|board.PieceBB[BlackBB]) >= EndgameThreshold
}

// Determine whether the current color to move is in check
func (board *Board) InCheck() bool {
	if board.WhiteToMove {
		kingBB := board.PieceBB[KingBB] & board.PieceBB[WhiteBB]
		return squareIsAttacked(board, BlackBB, kingBB, board.PieceBB[WhiteBB])
	} else {
		kingBB := board.PieceBB[KingBB] & board.PieceBB[BlackBB]
		return squareIsAttacked(board, WhiteBB, kingBB, board.PieceBB[BlackBB])
	}
}

// A convinece function used to make a move on the board
// using coordinate notation. This function is useful for
// debugging and loading moves from the uci interface. It
// returns the move to be used in UndoMove if needed.
func (board *Board) DoMoveFromCoords(move string, saveState bool, useChess960Castling bool) uint16 {
	moveInt := makeMoveFromCoords(board, move, useChess960Castling)
	board.DoMove(&moveInt, saveState)
	return moveInt
}

// Create a move from it's coordinate representation.
func makeMoveFromCoords(board *Board, move string, useChess960Castling bool) uint16 {
	fromPos := CoordinateToPos(move[0:2])
	toPos := CoordinateToPos(move[2:4])
	movePieceType := GetPieceType(board.Pieces[fromPos])
	var moveType int

	moveLen := len(move)
	if moveLen == 5 {
		if move[moveLen-1] == 'n' {
			moveType = KnightPromotion
		} else if move[moveLen-1] == 'b' {
			moveType = BishopPromotion
		} else if move[moveLen-1] == 'r' {
			moveType = RookPromotion
		} else if move[moveLen-1] == 'q' {
			moveType = QueenPromotion
		}
	} else if move == "e1g1" && movePieceType == KingBB && !useChess960Castling {
		moveType = CastleWKS
	} else if move == "e1c1" && movePieceType == KingBB && !useChess960Castling {
		moveType = CastleWQS
	} else if move == "e8g8" && movePieceType == KingBB && !useChess960Castling {
		moveType = CastleBKS
	} else if move == "e8c8" && movePieceType == KingBB && !useChess960Castling {
		moveType = CastleBQS
	} else if move == "e1h1" && movePieceType == KingBB && useChess960Castling {
		moveType = CastleWKS
	} else if move == "e1a1" && movePieceType == KingBB && useChess960Castling {
		moveType = CastleWQS
	} else if move == "e8h8" && movePieceType == KingBB && useChess960Castling {
		moveType = CastleBKS
	} else if move == "e8a8" && movePieceType == KingBB && useChess960Castling {
		moveType = CastleBQS
	} else if toPos == board.EPSquare {
		moveType = AttackEP
	} else {
		capturePiece := board.Pieces[toPos]
		if capturePiece == NoPiece {
			moveType = Quiet
		} else {
			moveType = Attack
		}
	}
	return MakeMove(fromPos, toPos, moveType)
}

// Pretty-print a representation of the internal board.
func (board *Board) PrintBoard() {
	fmt.Println()
	for rankStartPos := 56; rankStartPos >= 0; rankStartPos -= 8 {
		fmt.Printf("%v | ", (rankStartPos/8)+1)
		for index := rankStartPos; index < rankStartPos+8; index++ {
			square := board.Pieces[index]
			piece := GetPieceType(square)
			color := getPieceColor(square)
			var squareChar rune
			if square != NoPiece {
				if piece == PawnBB {
					squareChar = 'p'
				} else if piece == KnightBB {
					squareChar = 'n'
				} else if piece == BishopBB {
					squareChar = 'b'
				} else if piece == RookBB {
					squareChar = 'r'
				} else if piece == QueenBB {
					squareChar = 'q'
				} else if piece == KingBB {
					squareChar = 'k'
				}
			} else {
				squareChar = '.'
			}
			if color == WhiteBB {
				squareChar = unicode.ToUpper(squareChar)
			}
			fmt.Printf("%c ", squareChar)
		}
		fmt.Println()
	}

	fmt.Print("   ")
	for fileNo := 0; fileNo < 8; fileNo++ {
		fmt.Print("--")
	}

	fmt.Print("\n    ")
	for _, file := range "abcdefgh" {
		fmt.Printf("%c ", file)
	}
	fmt.Print("\n\n")

	if board.WhiteToMove {
		fmt.Println("Whose move: White")
	} else {
		fmt.Println("Whose move: Black")
	}

	fmt.Print("Castling rights: ")
	if board.CastlingRights&WhiteKingside != 0 {
		fmt.Print("K")
	}
	if board.CastlingRights&WhiteQueenside != 0 {
		fmt.Print("Q")
	}
	if board.CastlingRights&BlackKingside != 0 {
		fmt.Print("k")
	}
	if board.CastlingRights&BlackQueenside != 0 {
		fmt.Print("q")
	}

	fmt.Print("\nEn passant square: ")
	if board.EPSquare == NoEPSquare {
		fmt.Print("None")
	} else {
		fmt.Printf(PosToCoordinate(board.EPSquare))
	}

	fmt.Printf("\nHalf-move clock: %d\n", board.HalfMoveClock)
	fmt.Printf("Full-move counter: %d\n", board.FullMoveCounter)
	fmt.Printf("Zobrist hash: 0x%x\n\n", board.Hash)
}

// Helper functions to apply the correct masks
// to a piece-square from Board.Pieces,and get
// the correct index into Board.PieceBB where the
// corresponding color and piece bitboards are
// stored.

// Get a pieces type
func GetPieceType(square uint8) int {
	return int((square & PieceMask) >> 5)
}

// Get a pieces color
func getPieceColor(square uint8) int {
	return int((square & ColorMask) >> 2)
}
