package main

import (
	"testing"
)

func makeBoard(tiles [][]int) Board {
	turn := 0
	for _, row := range tiles {
		for _, t := range row {
			if t > turn {
				turn = t
			}
		}
	}
	return Board{
		[]string{"a", "b"},
		true,
		tiles,
		turn + 1,
		nil,
	}
}
func TestWinning(t *testing.T) {
	board1 := makeBoard([][]int{
		{1, 0, 0, 0, 0},
		{0, 8, 0, 0, 0},
		{1, 3, 5, 7, 9},
		{0, 0, 0, 2, 0},
	})

	win1 := &FiveInARow{0, 2, 1, 0}
	lose1 := &FiveInARow{0, 0, 1, 0}

	board2 := makeBoard([][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 8, 0, 0, 0, 0},
		{1, 3, 4, 7, 9, 0},
		{0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 12, 0},
		{0, 0, 0, 13, 0, 0},
	})

	copiedTiles := make([][]int, len(board2.Tiles))
	for i := range copiedTiles {
		copiedTiles[i] = make([]int, len(board2.Tiles[i]))
		copy(copiedTiles[i], board2.Tiles[i])
	}

	win2 := &FiveInARow{0, 0, 1, 1}

	board3 := makeBoard([][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0},
		{2, 4, 1, 7, 8, 0},
		{0, 0, 3, 9, 0, 0},
		{0, 0, 5, 0, 11, 0},
		{0, 0, 0, 0, 0, 0},
	})

	board4 := makeBoard([][]int{
		{0, 8, 0, 0, 0, 0},
		{0, 3, 5, 7, 9, 0},
		{0, 0, 0, 2, 0, 0},
	})

	win4 := &FiveInARow{0, 1, 1, 0}

	if lose1.Wins(board1.Tiles) {
		t.Errorf("should not win")
	}
	if !win1.Wins(board1.Tiles) {
		t.Errorf("win1 should win board1")
	}

	if win := board1.findWinning(); win == nil || *win != *win1 {
		t.Errorf("board1 win: %x != %x", win, win1)
	}

	if win := board2.findWinning(); win != nil {
		t.Errorf("board2 won too soon %x", win)
	}

	if win := board2.findNextWinning(); win == nil || *win != *win2 {
		t.Errorf("board2 should win: %x != %x", win, win2)
	}

	for y, row := range board2.Tiles {
		for x, tile := range row {
			if tile != copiedTiles[y][x] {
				t.Errorf("board2 Tiles were messed with at (%d, %d): %x != %x", x, y, tile, copiedTiles[y][x])
			}
		}
	}
	win := board2.findNextNextWinning()
	if won, ok := win[*win2]; !ok || !won {
		t.Errorf("board2 should win: %v != %x", win, win2)
	}

	win = board3.findNextNextWinning()
	if won, ok := win[*win2]; !ok || !won {
		t.Errorf("board3 should win: %v != %x", win, win2)
	}

	win = board4.findNextNextWinning()
	if won, ok := win[*win4]; !ok || !won {
		t.Errorf("board4 win: %v != %x", win, win4)
	}
}
