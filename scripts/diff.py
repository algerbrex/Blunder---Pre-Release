from re import compile as comp
from itertools import zip_longest
from collections import namedtuple
from sys import argv

class Move(namedtuple("Move", "move nodes")):
    def __eq__(self, other):
        return self.move == other.move

def run(depth_one):

    PERFT_TOOL_OUTPUT_RE = comp('(?P<move>\D\d\D\d[\D]*): (?P<nodes>\d+)')
    BLUNDER_OUTPUT_RE = comp('(?P<move>\D\d[-|x]\D\d[\D]*): (?P<nodes>\d+)')
    DEPTH_ONE = depth_one

    perft_moves = []
    blunder_moves = []
    perft_tool_lines_done = False

    with open('output.txt', 'r') as openfile:
        for line in openfile.readlines():
            if line == '\n':
                continue
            if line.replace('\n', '').replace(' ', '') == '$':
                perft_tool_lines_done = True
                continue
            if perft_tool_lines_done:
                match = BLUNDER_OUTPUT_RE.match(line.replace('\n', ''))
                blunder_moves.append(Move(match.group("move").replace('-', '').replace('x', ''), int(match.group("nodes"))))
            else:
                match = PERFT_TOOL_OUTPUT_RE.match(line.replace('\n', ''))
                perft_moves.append(Move(match.group("move"), int(match.group("nodes"))))

    for tool_move, blunder_move in zip_longest(sorted(perft_moves), sorted(blunder_moves)):
        if tool_move is not None and blunder_move is not None:
            if tool_move.nodes != blunder_move.nodes and not DEPTH_ONE:
                print(
                    f"For move {tool_move.move}, the perft tool has {tool_move.nodes} nodes, "
                    f"blunder has {blunder_move.nodes}"
                )
            if tool_move.move != blunder_move.move:
                if len(perft_moves) > len(blunder_moves):
                    print(
                        f"The perft tool has the extra move {tool_move.move} with {tool_move.nodes} nodes"
                    )
                else:
                    print(f"Blunder has the extra move {blunder_move.move} with {blunder_move.nodes} nodes")
                break

if __name__ == '__main__':
    opt = argv[1]
    if opt == '-depth1':
        run(True)
    else:
        run(False)

