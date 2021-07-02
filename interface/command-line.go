package inter

import (
	"blunder/core"
	"bufio"
	"fmt"
	"os"
	"strings"
)

// This file contains a basic program to play blunder
// from the command line. Moves are entered in basic
// coordinate notation

func RunCommandLineProtocol() {
	reader := bufio.NewReader(os.Stdin)
	var searcher core.Searcher
	searcher.Init()

	fmt.Println("Enter a fen string for the starting position (or startpos for the start position): ")
	input, _ := reader.ReadString('\n')

	if input == "startpos\n" {
		searcher.LoadFEN(core.FENStartPosition)
	} else {
		searcher.LoadFEN(strings.TrimSuffix(input, "\n"))
	}

	var playerToMove bool
	fmt.Println("Are you white or black? ")
	input, _ = reader.ReadString('\n')

	if input == "white\n" && searcher.Board.WhiteToMove {
		playerToMove = true
	} else if input == "black\n" && !searcher.Board.WhiteToMove {
		playerToMove = true
	}

	for {
		searcher.Board.PrintBoard()

		if playerToMove {
			fmt.Print("Enter your move (in uci protocol formation)> ")
			input, _ = reader.ReadString('\n')
			if input == "quit\n" {
				break
			}
			input = strings.TrimSuffix(input, "\n")
			searcher.Board.DoMoveFromCoords(input, false, false)
			playerToMove = false
		} else {
			// No time restriction, so always pass in something above a 1:30 of time
			// so Blunder won't think it has to rush.
			bestMove := searcher.Search(core.TimeThreshHoldForBulletPlay + 1)
			searcher.Board.DoMove(&bestMove, false)
			playerToMove = true
		}
	}
}
