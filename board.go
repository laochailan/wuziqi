package main

import (
	"encoding/json"

	"github.com/lithammer/shortuuid/v3"
)

type Board struct {
	PlayerIds []string
	FirstUseX bool
	Tiles     [][]int
	Turn      int
	Winner    WinnerSet
}

func createBoard(size int, firstUseX bool) Board {
	tiles := make([][]int, size)
	for i := range tiles {
		tiles[i] = make([]int, size)
	}
	return Board{[]string{shortuuid.New()[1:6], shortuuid.New()[1:6]}, firstUseX, tiles, 1, nil}
}

func (b *Board) ActiveTurn(player int) bool {
	return player == b.Turn%2
}

func (b *Board) NextTurn() int {
	return b.Turn + 1
}

func (b *Board) FormatTile(turn int) string {
	if turn == 0 {
		return "empty_tile"
	}
	if b.FirstUseX {
		turn += 1
	}

	if turn%2 == 1 {
		return "x_tile"
	}
	return "o_tile"
}

type FiveInARow struct {
	x, y       int
	dirx, diry int
}

func (b *Board) WinningTile(x, y int) bool {
	return b.Tiles[y][x] != 0 && b.Winner.WinningTile(x, y)
}

func (f FiveInARow) WinningTile(x, y int) bool {
	for i := 0; i < 5; i++ {
		dx := i * f.dirx
		dy := i * f.diry

		if f.y+dy == y && f.x+dx == x {
			return true
		}
	}
	return false
}
func (f FiveInARow) Wins(tiles [][]int) bool {
	player := tiles[f.y][f.x] % 2
	for i := 0; i < 5; i++ {
		dx := i * f.dirx
		dy := i * f.diry
		if f.y+dy < 0 || f.x+dx < 0 || f.y+dy >= len(tiles) ||
			f.x+dx >= len(tiles[f.y+dy]) {
			return false
		}

		if t := tiles[f.y+dy][f.x+dx]; t == 0 || t%2 != player {
			return false
		}
	}
	return true
}

func (f FiveInARow) WinningPlayer(board *Board) int {
	for i := 0; i < 5; i++ {
		dx := i * f.dirx
		dy := i * f.diry

		t := board.Tiles[f.y+dy][f.x+dx]
		if t != 0 {
			return t % 2
		}
	}

	panic("Row was not actually won!")
}

type WinnerSet map[FiveInARow]bool

func (fm *WinnerSet) UnmarshalJSON(b []byte) error {
	var winners []FiveInARow
	if err := json.Unmarshal(b, &winners); err != nil {
		return err
	}

	*fm = make(map[FiveInARow]bool)

	for _, w := range winners {
		(*fm)[w] = true
	}

	return nil
}

func (fm *WinnerSet) MarshalJSON() ([]byte, error) {
	var winners []FiveInARow
	for w, _ := range *fm {
		winners = append(winners, w)
	}

	return json.Marshal(winners)
}

func (fm WinnerSet) WinningTile(x, y int) bool {
	for f := range fm {
		if f.WinningTile(x, y) {
			return true
		}
	}
	return false
}
func (fm WinnerSet) WinningPlayer(board *Board) int {
	for f := range fm {
		return f.WinningPlayer(board)
	}
	return -1
}

func (b *Board) findWinning() *FiveInARow {
	for y := range b.Tiles {
		for x, t := range b.Tiles[y] {
			if t == 0 {
				continue
			}

			for _, row := range []FiveInARow{
				{x, y, 1, 0},
				{x, y, 1, 1},
				{x, y, 0, 1},
				{x, y, 1, -1},
			} {
				if row.Wins(b.Tiles) {
					return &row
				}
			}
		}
	}

	return nil
}

func (b *Board) findNextWinning() *FiveInARow {
	for y := range b.Tiles {
		for x, t := range b.Tiles[y] {
			if t == 0 {
				b.Tiles[y][x] = b.Turn
				b.Turn += 1
				winner := b.findWinning()
				b.Turn -= 1
				b.Tiles[y][x] = 0

				if winner != nil {
					return winner
				}
			}
		}
	}
	return nil
}

func (b *Board) findNextNextWinning() map[FiveInARow]bool {
	winners := make(WinnerSet)
	all_winners := true
	for y := range b.Tiles {
		for x, t := range b.Tiles[y] {
			if t == 0 {
				b.Tiles[y][x] = b.Turn
				b.Turn += 1
				winner_next := b.findWinning()
				var winner *FiveInARow
				if winner_next == nil {
					winner = b.findNextWinning()
				}
				b.Turn -= 1
				b.Tiles[y][x] = 0

				if winner_next != nil {
					return map[FiveInARow]bool{*winner_next: true}
				}

				if winner == nil {
					all_winners = false
				} else {
					winners[*winner] = true
				}
			}
		}
	}
	if !all_winners {
		return nil
	}
	return winners
}
