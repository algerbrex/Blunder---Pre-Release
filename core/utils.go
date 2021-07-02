package core

import (
	"fmt"
	"math/bits"
	"time"
)

const Int64MostSigBitSet = 0x8000000000000000

// Convert a board coordinate as a string - such as f6 or b2 -
// into a position into a 64-length array.
func CoordinateToPos(coordinate string) int {
	file := coordinate[0] - 'a'
	rank := charToDigit(coordinate[1]) - 1
	return int(rank*8 + int(file))
}

// Convert a position from a 64-length array, into a board
// coordinate as a string.
func PosToCoordinate(pos int) string {
	file := pos % 8
	rank := pos / 8
	return string(rune('a'+file)) + string(rune('0'+rank+1))
}

func charToDigit(r byte) int {
	return int(r - '0')
}

// A helper function for pretty-printing a bitstring as a 2d board
func Print2dBitboard(bitboard uint64) {
	bitstring := fmt.Sprintf("%064b\n", bitboard)
	fmt.Println()
	for rankStartPos := 56; rankStartPos >= 0; rankStartPos -= 8 {
		fmt.Printf("%v | ", (rankStartPos/8)+1)
		for index := rankStartPos; index < rankStartPos+8; index++ {
			squareChar := bitstring[index]
			if squareChar == '0' {
				squareChar = '.'
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
	fmt.Println()
}

// Set the bit of a 64-bit integer at the position given,
// where index 0 = the most significant bit
func setBit(integer *uint64, pos int) {
	*integer |= (Int64MostSigBitSet >> pos)
}

// Set a singe bit in a 64-bit integer and return the
// given value. This is contrasted to setBit, which
// does it's work in-place.
func setSingleBit(pos int) uint64 {
	return Int64MostSigBitSet >> pos
}

// Clear the bit of a 64-bit integer at the position given,
// where index 0 = the most significant bit
func clearBit(integer *uint64, pos int) {
	var mask uint64 = ^(Int64MostSigBitSet >> pos)
	*integer &= mask
}

// Determine if the bit at the given position is set in the
// 64-bit integer given.
func hasBitSet(integer uint64, pos int) bool {
	return (integer & (Int64MostSigBitSet >> pos)) > 0
}

// Get the position of the least significant bit, where
// where index 0 = the most significant bit
func getLSBPos(integer uint64) int {
	return 63 - bits.TrailingZeros64(integer)
}

// Get the position of the most significant bit, where
// where index 0 = the most significant bit
func getMSBPos(integer uint64) int {
	return bits.LeadingZeros64(integer)
}

// Get the position of the next least significant bit,
// (where index 0 = the most significant bit), and the
// bitboard for it, and finally clear the bit from the
// 64-bit integer given. This is a useful helper function
// when generating moves from a bitboard.
func popLSB(integer *uint64) (int, uint64) {
	pos := getLSBPos(*integer)
	bbWithPosSet := *integer & -*integer
	*integer &= *integer - 1
	return pos, bbWithPosSet
}

// Get the absolute value of an integer
func abs(integer int) int {
	if integer < 0 {
		return -integer
	} else {
		return integer
	}
}

// Get the maximum between two numbers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Get the minimum between two numbers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// A convience function to measure the execution time of a function
func timeit(start time.Time) {
	elapsed := time.Since(start)
	fmt.Printf("ms: %vms\n", int64(elapsed/time.Millisecond))
}

// Convert an internal move for blunder into a UCI formatted move string
func ConvertMoveToLongAlgebraicNotation(move uint16) string {
	from, to, moveType := GetMoveInfo(move)
	fromCoord := PosToCoordinate(from)
	toCoord := PosToCoordinate(to)

	if moveType == KnightPromotion {
		return fmt.Sprintf("%v%vn", fromCoord, toCoord)
	} else if moveType == BishopPromotion {
		return fmt.Sprintf("%v%vb", fromCoord, toCoord)
	} else if moveType == RookPromotion {
		return fmt.Sprintf("%v%vr", fromCoord, toCoord)
	} else if moveType == QueenPromotion {
		return fmt.Sprintf("%v%vq", fromCoord, toCoord)
	} else {
		return fmt.Sprintf("%v%v", fromCoord, toCoord)
	}
}

// Convert a move in UCI format to an interal move for Blunder
func ConvertLongAlgebraicNotationToMove(board *Board, moveAsString string) uint16 {
	fromPos := CoordinateToPos(moveAsString[0:2])
	toPos := CoordinateToPos(moveAsString[2:4])
	movePieceType := GetPieceType(board.Pieces[fromPos])
	var moveType int

	moveAsStringLen := len(moveAsString)
	if moveAsStringLen == 5 {
		if moveAsString[moveAsStringLen-1] == 'n' {
			moveType = KnightPromotion
		} else if moveAsString[moveAsStringLen-1] == 'b' {
			moveType = BishopPromotion
		} else if moveAsString[moveAsStringLen-1] == 'r' {
			moveType = RookPromotion
		} else if moveAsString[moveAsStringLen-1] == 'q' {
			moveType = QueenPromotion
		}
	} else if moveAsString == "e1g1" && movePieceType == KingBB {
		moveType = CastleWKS
	} else if moveAsString == "e1c1" && movePieceType == KingBB {
		moveType = CastleWQS
	} else if moveAsString == "e8g8" && movePieceType == KingBB {
		moveType = CastleBKS
	} else if moveAsString == "e8c8" && movePieceType == KingBB {
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
