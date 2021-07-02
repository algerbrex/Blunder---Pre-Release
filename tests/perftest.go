package tests

import (
	"blunder/core"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DepthLimit = 6

type PerftTest struct {
	FEN         string
	DepthValues [7]uint64
}

func loadPerftSuite() (perftTests []PerftTest) {
	home, _ := os.UserHomeDir()
	filePath := filepath.Join(home, "goprojects/blunder/tests/perftsuite.epd")

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ";")
		perftTest := PerftTest{FEN: strings.TrimSpace(fields[0])}
		perftTest.DepthValues = [7]uint64{}

		for _, nodeCountStr := range fields[1:] {
			depth, err := strconv.Atoi(string(nodeCountStr[1]))
			if err != nil {
				panic(fmt.Sprintf("Parsing error on line: %s\n", line))
			}
			nodeCountStr = strings.TrimSpace(nodeCountStr[3:])
			nodeCount, err := strconv.Atoi(nodeCountStr)
			if err != nil {
				panic(fmt.Sprintf("Parsing error on line: %s\n", line))
			}
			perftTest.DepthValues[depth-1] = uint64(nodeCount)
		}
		perftTests = append(perftTests, perftTest)
	}
	return perftTests
}

func RunPerftTests(board *core.Board, ttable *[core.TTPerftSize]core.PerftTTEntry) {
	perftTests := loadPerftSuite()
	totalTests := 0.0
	correctTests := 0.0

	for _, perftTest := range perftTests {
		fmt.Println("Running perft on position:", perftTest.FEN)
		board.LoadFEN(perftTest.FEN)
		board.PrintBoard()
		fmt.Println()

		for depth, nodeCount := range perftTest.DepthValues {
			if nodeCount == 0 {
				continue
			}

			result := core.RawPerft(board, depth+1, ttable)
			totalTests++
			if nodeCount == result {
				fmt.Println("Correct node count of", nodeCount, "at a depth of", depth+1)
				correctTests++
			} else {
				fmt.Printf("Wrong node count at depth %d. Correct is %d, but Blunder got %d\n", depth+1,
					nodeCount, result)
			}
			board.LoadFEN(perftTest.FEN)
		}
		fmt.Println()
	}
	fmt.Println("Summary of tests run:")
	fmt.Printf("Out of %f tests, %f were correct, with a percentage of %f\n",
		totalTests, correctTests, (correctTests/totalTests)*100)
}
