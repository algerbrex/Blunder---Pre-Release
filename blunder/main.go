package main

import (
	"blunder/core"
	inter "blunder/interface"
	"fmt"
)

var DEBUG bool = false

func main() {
	if DEBUG {
		var searcher core.Searcher
		searcher.Init()
		// r4k1r/pp3pbp/5n2/3p4/6b1/1B6/PPPP1PP1/R1B1R1K1 b - - 1 20
		searcher.LoadFEN("r1b1kbn1/pppp2pp/2n2N2/2q1p3/1rB1P1Q1/8/PPPP1PPP/RNB1K2R b KQq - 3 3")
		fmt.Println(core.EvaluateKingSaftey(&searcher.Board, core.BlackBB, core.WhiteBB))
		/*bestMove := searcher.Search(core.TimeThreshHoldForBulletPlay + 1)
		fmt.Println("Best move:", core.MoveToStr(bestMove))
		fmt.Println("Nodes explored:", searcher.NodesExplored)
		fmt.Println("Transposition table hits:", searcher.TTHits)*/
	} else {
		inter.RunUCIProtocol()
	}
}
