package core

import (
	"math/bits"
)

const (
	PawnValue   = 100
	KnightValue = 320
	BishopValue = 330
	RookValue   = 500
	QueenValue  = 975
	KingValue   = PosInf

	// These values are not actually infinity of course, or verfy large,
	// but functional for negascout they will be.
	PosInf = 2000000
	NegInf = -PosInf

	// Value of a draw
	DrawValue = 0

	// Indexes into the PieceSquareTables array for the king middle and endgame
	// piece square tables.
	KingPSTMiddlegameIndex = 3
	KingPSTEndgameIndex    = 4
)

// Table containg piece square tables for each piece, indexed
// by their bitboard index (see the constants in board.go)
var PieceSquareTables [8][64]int = [8][64]int{

	// Piece-square table for pawns
	{
		25, 25, 25, 25, 25, 25, 25, 25,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		-5, -5, -5, -5, -5, -5, -5, -5,
		-15, -2, 3, 15, 15, 3, -2, -15,
		-15, 2, 5, 5, 5, 5, 2, -15,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	},

	// Piece-square table for knights
	{
		-15, -15, -15, -15, -15, -15, -15, -15,
		-2, -2, -2, -2, -2, -2, -2, -2,
		-5, 0, 2, 2, 2, 2, 0, -5,
		-5, 0, 15, 25, 25, 15, 0, -5,
		-5, 0, 15, 25, 25, 15, 0, -5,
		-5, 0, 25, 25, 25, 25, 0, -5,
		-2, -2, -2, -2, -2, -2, -2, -2,
		-15, -15, -15, -15, -15, -15, -15, -15,
	},

	// Piece-square table for bishops
	{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		2, 5, 5, 0, 0, 5, 5, 2,
		2, 15, 5, 0, 0, 5, 15, 2,
		2, -5, -25, 0, 0, -25, -5, 2,
	},

	// Piece square table for kings in the middle game
	{
		-75, -75, -75, -75, -75, -75, -75, -75,
		-75, -75, -75, -75, -75, -75, -75, -75,
		-75, -75, -75, -75, -75, -75, -75, -75,
		-75, -75, -75, -75, -75, -75, -75, -75,
		-75, -75, -75, -75, -75, -75, -75, -75,
		-75, -75, -75, -75, -75, -75, -75, -75,
		25, 25, -10, -50, -50, -10, 25, 25,
		75, 50, 0, 0, 0, 0, 50, 75,
	},

	// Piece square table for kings in the endgame
	{
		-10, -10, -10, -10, -10, -10, -10, -10,
		-10, -5, -5, -5, -5, -5, -5, -10,
		-10, 2, 5, 5, 5, 5, 2, -10,
		-10, 2, 5, 25, 25, 5, 2, -10,
		-10, 2, 5, 25, 25, 5, 2, -10,
		-10, 2, 5, 5, 5, 5, 2, -10,
		-10, -5, -5, -5, -5, -5, -5, -10,
		-10, -10, -10, -10, -10, -10, -10, -10,
	},
}

// Values for various pieces that might surround and thus
// help protect a king.
var piecesAroundKingValues [6]int = [6]int{
	// Pawn value
	8,
	// Knight value
	12,
	// Bishop value
	12,
	// Rook value
	16,
	// Queen value
	88,
	// King value
	4,
}

// Evaluate a board state.
func evaluateBoard(searcher *Searcher) (score int) {
	whiteScore := evaluateSide(&searcher.Board, WhiteBB, BlackBB)
	blackScore := evaluateSide(&searcher.Board, BlackBB, WhiteBB)

	if searcher.Board.WhiteToMove {
		return whiteScore - blackScore
	}
	return blackScore - whiteScore
}

// Evaluate a board state for a side.
func evaluateSide(board *Board, usColor, enemyColor int) (score int) {
	score += evaluateMaterial(board, usColor)
	score += evaluatePosition(board, usColor)
	score += EvaluateKingSaftey(board, usColor, enemyColor)
	return score
}

// Evalute the material for a side.
func evaluateMaterial(board *Board, usColor int) (score int) {
	score += bits.OnesCount64(board.PieceBB[PawnBB]&board.PieceBB[usColor]) * PawnValue
	score += bits.OnesCount64(board.PieceBB[KnightBB]&board.PieceBB[usColor]) * KnightValue
	score += bits.OnesCount64(board.PieceBB[BishopBB]&board.PieceBB[usColor]) * BishopValue
	score += bits.OnesCount64(board.PieceBB[RookBB]&board.PieceBB[usColor]) * RookValue
	score += bits.OnesCount64(board.PieceBB[QueenBB]&board.PieceBB[usColor]) * QueenValue
	return score
}

// Evaluate the position of a side using piece square tables
func evaluatePosition(board *Board, usColor int) (score int) {
	usBB := board.PieceBB[usColor] & ^(board.PieceBB[RookBB] | board.PieceBB[QueenBB])
	kingPos := getLSBPos(board.PieceBB[usColor] & board.PieceBB[KingBB])

	if board.IsEndgame() {
		score += PieceSquareTables[KingPSTEndgameIndex][kingPos]
	} else {
		score += PieceSquareTables[KingPSTMiddlegameIndex][kingPos]
	}

	delta, perspective := 0, -1
	if usColor == WhiteBB {
		delta, perspective = 63, 1
	}
	for usBB != 0 {
		piecePos, _ := popLSB(&usBB)
		pieceType := GetPieceType(board.Pieces[piecePos])
		score += PieceSquareTables[pieceType][(delta-piecePos)*perspective]
	}
	return score
}

// Evaluate the saftey of the king. The current method for doing this
// is to figure out what kind of friendly and enemy pieces surround a king,
// and return a score that's hopefully representive of how dangerous the situation
// is for the king by getting the net score of thsi evaluation between both sides.
func EvaluateKingSaftey(board *Board, usColor, enemyColor int) (score int) {
	kingBB := board.PieceBB[KingBB] & board.PieceBB[usColor]
	squaresAroundKing := SquaresAroundKing[getLSBPos(kingBB)] & ^kingBB
	enemyPiecesAroundKing := squaresAroundKing & board.PieceBB[enemyColor]
	for enemyPiecesAroundKing != 0 {
		pos, _ := popLSB(&enemyPiecesAroundKing)
		enemyPieceType := GetPieceType(board.Pieces[pos])
		score -= piecesAroundKingValues[enemyPieceType]
	}
	/*friendlyPiecesAroundKing := squaresAroundKing & board.PieceBB[usColor]
	for friendlyPiecesAroundKing != 0 {
		pos, _ := popLSB(&friendlyPiecesAroundKing)
		friendlyPieceType := GetPieceType(board.Pieces[pos])
		score += piecesAroundKingValues[friendlyPieceType] * 2
	}*/
	return score
}

// A convinece function to get a pieces value given
// its bitboard index.
func getPieceValue(pieceType int) int {
	switch pieceType {
	case KingBB:
		return KingValue
	case QueenBB:
		return QueenValue
	case RookBB:
		return RookValue
	case BishopBB:
		return BishopValue
	case KnightBB:
		return KnightValue
	case PawnBB:
		return PawnValue
	default:
		return 0
	}
}
