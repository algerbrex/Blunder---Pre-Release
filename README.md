NOTICE
------

This repository is no longer being maintained, and was released as an early version of [blunder](https://github.com/algerbrex/blunder). This version has a variety of significant bugs and missing features, which doesn't make it suitable for being an actual release. The code here is open source and will remain here for the foreseeable feature, so feel free to still look around.
-------

Blunder is a chess engine that is a work in progress. Currently it implements a negamax search algorithm,
with alpha-beta pruning, a transposition table, iterative deepening, move ordering, quiesence search and
a basic opening book. A working binary compiled on a linux machine is included. To compile the code yourself,
download golang and run "go build" in blunder/blunder. The source code is quite heavily commented as I intend
for the finished project to help others learn the basics of chess programming.

A basic description of the folders included in the project are as follows:

* blunder: Contains the main.go file to be compiled and run
* book: The name here is a bit of a misnomer, as it no longer contains Blunder's opening book but polyglot files used to test the Zobrist hashing of Blunder. The opening book can be found in blunder/blunder
* core: Contains the core code of Blunder, such as the move generator, search and evaluation phase, tables, and other data and functions. Start here to begin understanding the code
* interface: Contains files which implement how Blunder interacts with its environment, such as the UCI protocol or a command line interface.
* scripts: A dirty script that was used in debugging Blunder's move generator. Essentially it compares the perft difference between Stockfish and Blunder and pinpoints for what moves the node count differs. May be useful, but the code is very clunky. 
* tests: Contains the files which tests Blunder's move generator, using it's perft function and a suite of fen strings with correct node counts at various depths, and Blunder's Zobrist hashing.
