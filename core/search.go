package core

import (
	"fmt"
	"time"
)

const (
	// Max search depth of the engine
	SearchDepth = 8

	// Max quiesence search depth of engine
	QuiesenceSearchDepth = 3

	// Represents a null best move, which should
	// never actually be returned from the search
	NullMove uint16 = 0

	// The size of the transpositon table
	TTSize = 0x100000 * 16

	// Flags to indicate what kind of value a transposition table entry has
	AlphaFlag uint8 = iota
	BetaFlag
	ExactFlag

	// A flag to represent when an entry was not found
	NoEntryFlag = -1

	// A flag representing a null Zobrist hash value
	NullHash uint64 = 0

	// Bonus given to moves that are captures. Used in move ordering
	// to ensure that even a capture by the least valuable victim and the
	// most valuable attacker is still scored above other moves.
	CaptureBonus = 1000

	// Bonuses given to the two killer moves at any ply. Used in
	// move ordering.
	FirstKillerBonus  = 150
	SecondKillerBonus = 100

	// Number of book moves the engine will use
	BookMovesDepth = 5

	// Time, in milliseconds, at and under which Blunder
	// begins limiting its search time to a specfic threshold
	// to try not to lose on time.
	TimeThreshHoldForBulletPlay = 1000 * 180

	// Time per move under bullet circumstances (~2 min of time). This can
	// be used for bullet chess, but more generally, it's used to try to
	// prevent Blunder loosing games on time. Throughout the game Blunder
	// is allowed to think as deeply as it wants, but under a minute of time
	// left we have to be more judicially, and each move gets no more than ~3
	// seconds.
	TimePerMoveBullet = 2000 // in milliseconds
)

// A transpositon table entry
type TTEntry struct {
	Hash     uint64
	Depth    int
	Value    int
	Flag     uint8
	BestMove uint16
}

// This object provides a conveient container for
// holding the state needed during a search, mainly
// the board, and transposition table.
type Searcher struct {
	Board  Board
	ttable [TTSize]TTEntry

	// Store the killer moves of a play (i.e. the moves that caused
	// a beta cutoff)
	killerMoves [SearchDepth][2]uint16

	// Store moves that caused alpha to increase irrespective of the
	// position in which they were played, and order those higher.
	searchHistory [64][64]int

	// Variables to store information useful for debugging the engine
	NodesExplored uint64
	TTHits        uint64

	// A flag set by the GUI if we're told to stop searching, according
	// to the UCI protocol
	StopSearch bool

	// Number of book moves left to use before we start searching for our
	// own moves.
	BookMovesLeft int
}

// Initalize the searcher
func (searcher *Searcher) Init() {
	searcher.ttable = [TTSize]TTEntry{}
	searcher.BookMovesLeft = BookMovesDepth
}

// Load a fen string into the searcher
func (seacher *Searcher) LoadFEN(fen string) {
	seacher.Board.LoadFEN(fen)
}

// Get the best move to play via iterative deepening
func (searcher *Searcher) Search(timeLeft int64) uint16 {
	bestMove, bestScore := NullMove, NegInf
	movesToMate := 0
	var totalSearchTime int64 = 0

	for depth := 1; depth <= SearchDepth; depth++ {
		if searcher.StopSearch {
			searcher.StopSearch = false
			break
		}

		// Record the time the search took and report it to the GUI
		start := time.Now()
		bestMove, bestScore = searcher.rootNegamax(depth)
		timeTaken := int64(time.Since(start) / time.Millisecond)
		totalSearchTime += timeTaken
		// If we're under a time crunch, break early when we've used up all of the time
		// alloted for each search.
		if timeLeft <= TimeThreshHoldForBulletPlay && totalSearchTime >= TimePerMoveBullet {
			break
		}

		if bestScore > (PosInf-SearchDepth) && bestScore <= PosInf {
			// If we're getting a huge number for the score, we're mating,
			// and shoud return mate in however many moves down we found it.
			movesToMate = (PosInf - bestScore) / 2
		} else if bestScore < (NegInf+SearchDepth) && bestScore >= NegInf {
			// Otherwise if we get a huge negative number, we're getting mated
			// soon and should report the score as negative.
			movesToMate = (NegInf - bestScore) / 2
		}

		// If the score is a mate score, let the GUI how many full moves until the mate
		if movesToMate != 0 {
			fmt.Printf("info depth %d score mate %d time %d nodes %d\n", depth, movesToMate, timeTaken, searcher.NodesExplored)
		} else {
			fmt.Printf("info depth %d score cp %d time %d nodes %d\n", depth, bestScore, timeTaken, searcher.NodesExplored)
		}
		// Reset the node counter before the next search
		searcher.NodesExplored = 0
	}
	return bestMove
}

// Get the best move for the side to move in the current board
func (searcher *Searcher) rootNegamax(depth int) (uint16, int) {
	var moves []uint16
	GenLegalMoves(&searcher.Board, &moves)
	orderMoves(searcher, &moves, depth)

	alpha, beta := NegInf, PosInf-1
	bestMove, bestScore := NullMove, NegInf

	for _, move := range moves {
		fmt.Printf("info currmove %v\n", ConvertMoveToLongAlgebraicNotation(move))
		searcher.Board.DoMove(&move, true)
		bestScore = -searcher.negamax(depth-1, -beta, -alpha)
		searcher.Board.UndoMove(&move)

		if bestScore > alpha {
			alpha = bestScore
			bestMove = move
		}
		if bestScore >= beta {
			break
		}
	}
	return bestMove, bestScore
}

// The root negamax function in the searcher calls this main
// negamax function, which only returns an integer value representing
// the score of the best move found, which is all that's needed for
// the top-level call to get a best move.
func (searcher *Searcher) negamax(depth, alpha, beta int) int {
	if score := searcher.getEntry(depth, alpha, beta); score != NoEntryFlag {
		searcher.TTHits++
		return score
	}

	if depth == 0 {
		searcher.NodesExplored++
		score := evaluateBoard(searcher)
		searcher.setEntry(depth, score, ExactFlag)
		return searcher.quiescence(QuiesenceSearchDepth, alpha, beta)
	}

	var moves []uint16
	GenLegalMoves(&searcher.Board, &moves)

	if len(moves) == 0 {
		if searcher.Board.InCheck() {
			searcher.setEntry(depth, NegInf+(SearchDepth-depth), ExactFlag)
			return NegInf + (SearchDepth - depth)
		}
		searcher.setEntry(depth, 0, ExactFlag)
		return 0
	}

	orderMoves(searcher, &moves, depth)
	entryFlag := AlphaFlag

	for _, move := range moves {
		searcher.Board.DoMove(&move, true)
		score := -searcher.negamax(depth-1, -beta, -alpha)
		searcher.Board.UndoMove(&move)
		if score >= beta {
			searcher.setEntry(depth, beta, BetaFlag)
			if getMoveType(move) != Attack && getMoveType(move) != AttackEP {
				searcher.killerMoves[depth-1][1] = searcher.killerMoves[depth-1][0]
				searcher.killerMoves[depth-1][0] = move
			}
			return beta
		}
		if score > alpha {
			entryFlag = ExactFlag
			alpha = score
			if getMoveType(move) != Attack && getMoveType(move) != AttackEP {
				searcher.searchHistory[getMoveFromSq(move)][getMoveToSq(move)] = depth * depth
			}
		}
	}
	searcher.setEntry(depth, alpha, entryFlag)
	return alpha
}

func (searcher *Searcher) quiescence(depth, alpha, beta int) int {
	stand_pat := evaluateBoard(searcher)
	if depth == 0 {
		searcher.NodesExplored++
		return stand_pat
	}
	if stand_pat >= beta {
		return beta
	}
	if alpha < stand_pat {
		alpha = stand_pat
	}

	var moves []uint16
	GenLegalMoves(&searcher.Board, &moves)
	orderMoves(searcher, &moves, depth)

	for _, move := range moves {
		_, _, moveType := GetMoveInfo(move)
		if moveType == Attack || move == AttackEP {
			searcher.Board.DoMove(&move, true)
			score := -searcher.quiescence(depth-1, -beta, -alpha)
			searcher.Board.UndoMove(&move)

			if score >= beta {
				return beta
			}
			if score > alpha {
				alpha = score
			}
		}
	}
	return alpha
}

// A helper function to probe the transpositon table
func (searcher *Searcher) getEntry(depth, alpha, beta int) int {
	entry := searcher.ttable[searcher.Board.Hash%TTSize]
	if entry.Hash == searcher.Board.Hash {
		if entry.Depth >= depth {
			if entry.Flag == ExactFlag {
				return entry.Value
			}
			if entry.Flag == AlphaFlag && entry.Value <= alpha {
				return alpha
			}
			if entry.Flag == BetaFlag && entry.Value >= beta {
				return beta
			}
		}
	}
	return NoEntryFlag
}

func (searcher *Searcher) setEntry(depth, value int, flag uint8) {
	entry := &searcher.ttable[searcher.Board.Hash%TTSize]
	entry.Hash = searcher.Board.Hash
	entry.Value = value
	entry.Flag = flag
	entry.Depth = depth
}

// Order the moves with those that are most likley to be best (e.g.
// capturing a piece with a pawn), to optimize alpha-beta pruning.
func orderMoves(searcher *Searcher, moves *[]uint16, depth int) {
	moveScores := make([]int, len(*moves))
	for moveIndex, move := range *moves {
		from, to, moveType := GetMoveInfo(move)
		movePieceType := GetPieceType(searcher.Board.Pieces[from])
		capturePieceType := GetPieceType(searcher.Board.Pieces[to])

		if moveType == Attack || moveType == AttackEP {
			moveScores[moveIndex] = getPieceValue(capturePieceType) - getPieceValue(movePieceType) + CaptureBonus
		} else if moveType == KnightPromotion {
			moveScores[moveIndex] = KnightValue + getPieceValue(capturePieceType)
		} else if moveType == BishopPromotion {
			moveScores[moveIndex] = BishopValue + getPieceValue(capturePieceType)
		} else if moveType == RookPromotion {
			moveScores[moveIndex] = RookValue + getPieceValue(capturePieceType)
		} else if moveType == QueenPromotion {
			moveScores[moveIndex] = QueenValue + getPieceValue(capturePieceType)
		} else if searcher.killerMoves[depth-1][0] == move {
			moveScores[moveIndex] = FirstKillerBonus
		} else if searcher.killerMoves[depth-1][1] == move {
			moveScores[moveIndex] = SecondKillerBonus
		} else {
			moveScores[moveIndex] = searcher.searchHistory[from][to]
		}
	}
	sortMoves(moves, &moveScores)
}

// A helper function to sort the moves given an array with a moves
// score corresponding to it's index.
func sortMoves(moves *[]uint16, moveScores *[]int) {
	for i := 0; i < len(*moves)-1; i++ {
		for j := i + 1; j > 0; j-- {
			swapIndex := j - 1
			if (*moveScores)[swapIndex] < (*moveScores)[j] {
				(*moves)[j], (*moves)[swapIndex] = (*moves)[swapIndex], (*moves)[j]
				(*moveScores)[j], (*moveScores)[swapIndex] = (*moveScores)[swapIndex], (*moveScores)[j]
			}
		}
	}
}
