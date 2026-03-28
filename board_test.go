package main

import (
	"slices"
	"testing"
)

func makeBoard(tiles [][]int) Board {
	turn := 0
	flatTiles := make([]int, len(tiles)*len(tiles[0]))
	for y, row := range tiles {
		if len(row) != len(tiles) {
			panic("nonsquare tiles")
		}
		for x, t := range row {
			flatTiles[y*len(row)+x] = t
			if t > turn {
				turn = t
			}
		}
	}
	return Board{
		true,
		len(tiles),
		flatTiles,
		turn + 1,
		-1,
	}
}

func TestWinning(t *testing.T) {
	board1 := makeBoard([][]int{
		{1, 0, 0, 0, 0},
		{0, 8, 0, 0, 0},
		{1, 3, 5, 7, 9},
		{0, 0, 0, 2, 0},
		{0, 0, 0, 0, 0},
	})

	if !board1.WasWinningMove(BoardMove{0, 2}) {
		t.Errorf("should win")
	}
	if board1.WasWinningMove(BoardMove{3, 3}) {
		t.Errorf("should not win")
	}

	board2 := makeBoard([][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 8, 0, 0, 0, 0},
		{1, 3, 4, 7, 9, 0},
		{0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 12, 0},
		{0, 0, 0, 13, 14, 0},
	})

	copiedTiles := make([]int, len(board2.Tiles))
	copy(copiedTiles, board2.Tiles)
	copiedTurn := board2.Turn

	if board2.WasWinningMove(BoardMove{2, 2}) {
		t.Errorf("should not win")
	}

	if w := board2.PredictWinner(BoardMove{1, 1}); w != board2.PlayerOfTurn(2) {
		t.Errorf("wrong winner %d", w)
	}

	if !slices.Equal(board2.Tiles, copiedTiles) || board2.Turn != copiedTurn {
		t.Errorf("board was modified when it should not have been")
	}

	board3 := makeBoard([][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0},
		{2, 4, 1, 7, 8, 0},
		{0, 0, 3, 9, 0, 0},
		{0, 0, 5, 0, 11, 0},
		{0, 0, 0, 0, 0, 0},
	})

	if w := board3.PredictWinner(BoardMove{3, 2}); w != board3.PlayerOfTurn(1) {
		t.Fatalf("wrong winner %d", w)
	}

	if w := board3.PredictWinner(BoardMove{3, 3}); w != board3.PlayerOfTurn(1) {
		t.Errorf("wrong winner %d", w)
	}

	board4 := makeBoard([][]int{
		{0, 0, 16, 0, 0, 0},
		{1, 3, 5, 7, 0, 0},
		{0, 0, 0, 0, 15, 0},
		{0, 0, 0, 13, 0, 0},
		{0, 0, 11, 0, 0, 0},
		{0, 9, 0, 0, 0, 0},
	})

	if w := board4.PredictWinner(BoardMove{2, 0}); w != board4.PlayerOfTurn(1) {
		t.Errorf("wrong winner %d", w)
	}
	board4.ApplyMove(BoardMove{2, 3})
	if w := board4.PredictWinner(BoardMove{2, 3}); w != board4.PlayerOfTurn(1) {
		t.Errorf("wrong winner %d", w)
	}

	board5 := makeBoard([][]int{
		{0, 0, 0, 0, 0, 0},
		{1, 2, 4, 6, 8, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	})

	if board5.WasWinningMove(BoardMove{5, 1}) {
		t.Errorf("false winning move")
	}

	if w := board5.PredictWinner(BoardMove{5, 1}); w != -1 {
		t.Errorf("wrong winner %d", w)
	}
}
