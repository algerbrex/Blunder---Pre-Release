package tests

import (
	"blunder/core"
	inter "blunder/interface"
	"fmt"
	"os"
	"path/filepath"
)

const StartingPositionHash uint64 = 0x463b96181691fc9c

// Read a file at a time from the test files in test_books, and test them to
// verify Zobrist hashing is working correctly (see below).
func RunAllZobristHashingTests(board *core.Board, verbose bool) {
	for fileNameSuffix := 1; fileNameSuffix < 12; fileNameSuffix++ {
		home, _ := os.UserHomeDir()
		fmt.Printf("=========Testing: test%v.bin========\n", fileNameSuffix)
		filePath := filepath.Join(home, fmt.Sprintf("goprojects/blunder/book/test%d.bin", fileNameSuffix))
		RunZobristHashingTest(board, verbose, filePath)
		fmt.Println()
	}
	fmt.Print("All tests of Zobrist hashing were run succesfully\n\n")
}

// To ensure zobrist hashing is working correctly, Blunder's polyglot
// reader is used to read in a polyglot file from a game played, and
// apply the moves which correspond to the current board Zobrist hash.
// If all the moves are applied successivley, then the hashing is working
// correctly. This is discovered by undoing each move and seeing if the board
// is returned to its correct beginning state, which is always the inital
// position.
func RunZobristHashingTest(board *core.Board, verbose bool, path string) {
	entries, err := inter.LoadPolyglotFile(path)
	if err != nil {
		panic(err)
	}

	// Implement the 3-repititon rule for draws
	positionRepeats := make(map[uint64]int)
	var movesMade []uint16

	for {
		if entry, ok := entries[board.Hash]; ok {
			move := board.DoMoveFromCoords(entry.Move, true, true)
			movesMade = append(movesMade, move)
			if verbose {
				fmt.Printf("applying move %v at hash 0x%x\n", entry.Move, entry.Hash)
			}
			positionRepeats[board.Hash]++
			if positionRepeats[board.Hash] == 3 {
				break
			}
		} else {
			break
		}
	}

	fmt.Print("\n=======Hashing test ran succesfully for board.DoMove=======\n")
	if verbose {
		board.PrintBoard()
	}

	for len(movesMade) != 0 {
		move := pop(&movesMade)
		if verbose {
			fmt.Printf("undoing move %v at hash 0x%x\n", core.MoveToStr(move), board.Hash)
		}
		board.UndoMove(&move)

		_, ok := entries[board.Hash]
		if !ok {
			board.PrintBoard()
			panic("invalid hash for position shown")
		}
	}

	if board.Hash != StartingPositionHash {
		panic("testing UndoMove hashing failed")
	}

	if len(movesMade) == 0 {
		fmt.Print("======Hashing test ran succesfully for board.UndoMove======\n")
	}

	if verbose {
		fmt.Println("================Final Board================")
		board.PrintBoard()
	}
}

// A helper function to pop and item from a slice
func pop(s *[]uint16) (item uint16) {
	item, *s = (*s)[len(*s)-1], (*s)[:len(*s)-1]
	return item
}
