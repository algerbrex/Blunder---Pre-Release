package inter

import (
	"blunder/core"
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EngineName        = "Blunder 0.3"
	EngineAuthor      = "Christian Dean"
	BookMoveTimeDelay = 2
)

func uciCommandResponse() {
	fmt.Printf("id name %v\n", EngineName)
	fmt.Printf("id author %v\n", EngineAuthor)
	fmt.Printf("uciok\n")
}

func isreadyCommandResponse(board *core.Board) {
	board.LoadFEN(core.FENStartPosition)
	fmt.Printf("readyok\n")
}

func positionCommandResponse(searcher *core.Searcher, command string) {
	args := strings.TrimPrefix(command, "position ")
	var fenString string
	if strings.HasPrefix(args, "startpos") {
		args = strings.TrimPrefix(args, "startpos ")
		fenString = core.FENStartPosition
	} else if strings.HasPrefix(args, "fen") {
		args = strings.TrimPrefix(args, "fen ")
		remaining_args := strings.Fields(args)
		fenString = strings.Join(remaining_args[0:6], " ")
		args = strings.Join(remaining_args[6:], " ")
	}

	searcher.Board.LoadFEN(fenString)
	if strings.HasPrefix(args, "moves") {
		args = strings.TrimPrefix(args, "moves ")
		for _, moveAsString := range strings.Fields(args) {
			move := core.ConvertLongAlgebraicNotationToMove(&searcher.Board, moveAsString)
			searcher.Board.DoMove(&move, false)
		}
	}
}

func getBookMove(board *core.Board, openingBook *map[uint64]PolyglotEntry) string {
	if entry, ok := (*openingBook)[board.Hash]; ok {
		// Verify that a move from the book is legal in the current position.
		var moves []uint16
		core.GenLegalMoves(board, &moves)
		for _, move := range moves {
			if entry.Move == core.ConvertMoveToLongAlgebraicNotation(move) {
				return entry.Move
			}
		}
	}
	return ""
}

func getTimeLeftInGame(whiteToMove bool, command string) int64 {
	fields := strings.Fields(command)
	for index, field := range fields {
		if strings.HasPrefix(field, "wtime") && whiteToMove {
			timeAsStr := fields[index+1]
			time, err := strconv.Atoi(timeAsStr)
			if err != nil {
				break
			}
			return int64(time)
		} else if strings.HasPrefix(field, "btime") && !whiteToMove {
			timeAsStr := fields[index+1]
			time, err := strconv.Atoi(timeAsStr)
			if err != nil {
				break
			}
			return int64(time)
		}
	}
	// If not time is given, assume we have enough time to do a full search
	return core.TimeThreshHoldForBulletPlay + 1
}

func goCommandResponse(searcher *core.Searcher, openingBoook map[uint64]PolyglotEntry, command string) {
	command = strings.TrimPrefix(command, "go ")
	if bookMove := getBookMove(&searcher.Board, &openingBoook); bookMove != "" && searcher.BookMovesLeft > 0 {
		time.Sleep(time.Second * BookMoveTimeDelay)
		fmt.Printf("bestmove %v\n", bookMove)
		searcher.BookMovesLeft--
	} else {
		bestMove := searcher.Search(getTimeLeftInGame(searcher.Board.WhiteToMove, command))
		if bestMove == core.NullMove {
			panic("nullmove encountered")
		}
		fmt.Printf("bestmove %v\n", core.ConvertMoveToLongAlgebraicNotation(bestMove))
	}
}

func quitCommandResponse() {
	// unitialize engine memory/threads
}

func printCommandResponse() {
	// print internal engine info
}

func RunUCIProtocol() {
	reader := bufio.NewReader(os.Stdin)
	var searcher core.Searcher
	searcher.Init()

	// If we can't find our opening book, we're not in the best
	// situation, but we can play without it, so don't make the
	// program crash. Make an empty opening book and move on.
	openingBook := make(map[uint64]PolyglotEntry)
	filePath, err := os.Getwd()

	if err == nil {
		openingBook, err = LoadPolyglotFile(filepath.Join(filePath, "book.bin"))
		if err != nil {
			log.Println("Loading opening book failed")
		}
	} else {
		log.Println("Loading opening book failed")
	}

	isReadyAlreadySent := false
	for {
		command, _ := reader.ReadString('\n')
		if command == "uci\n" {
			uciCommandResponse()
		} else if command == "isready\n" {
			if !isReadyAlreadySent {
				isreadyCommandResponse(&searcher.Board)
				isReadyAlreadySent = true
			} else {
				fmt.Printf("readyok\n")
			}
		} else if strings.HasPrefix(command, "setoption") {
			// ignore
			// set internal engine options
		} else if strings.HasPrefix(command, "ucinewgame") {
			searcher.Init()
		} else if strings.HasPrefix(command, "position") {
			positionCommandResponse(&searcher, command)
		} else if strings.HasPrefix(command, "go") {
			go goCommandResponse(&searcher, openingBook, command)
		} else if strings.HasPrefix(command, "stop") {
			searcher.StopSearch = true
		} else if command == "quit\n" {
			quitCommandResponse()
			break
		} else if command == "print\n" {
			printCommandResponse()
		}
	}
}
