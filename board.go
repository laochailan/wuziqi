package main

import "slices"

type Board struct {
	FirstUseX bool
	Size      int
	Tiles     []int
	Turn      int

	Winner int
}

func createBoard(size int, firstUseX bool) *Board {
	tiles := make([]int, size*size)
	return &Board{firstUseX, size, tiles, 1, -1}
}

func (b *Board) Reset() {
	for i := range b.Tiles {
		b.Tiles[i] = 0
	}
	b.Turn = 1
	b.Winner = -1
}

func (b *Board) Clone() *Board {
	copied := &*b
	copied.Tiles = slices.Clone(copied.Tiles)
	return copied
}

func (b *Board) ActivePlayer() int {
	return b.PlayerOfTurn(b.Turn)
}

func (b *Board) PlayerOfTurn(turn int) int {
	return turn % 2
}

func (b *Board) Resign() {
	b.Winner = b.PlayerOfTurn(b.Turn - 1)
}

type BoardMove struct {
	X int
	Y int
}

func (b *Board) ApplyMove(move BoardMove) bool {
	idx := b.Size*move.Y + move.X
	if move.X < 0 || move.X >= b.Size || move.Y < 0 || move.Y >= b.Size || b.Tiles[idx] != 0 {
		return false
	}

	b.Tiles[idx] = b.Turn
	b.Turn++
	return true
}

func (b *Board) UnapplyMove(move BoardMove) {
	b.Tiles[b.Size*move.Y+move.X] = 0
	b.Turn--
}

// WasWinningMove returns true if the  tile marked at x, y triggered a win for that player
func (b *Board) WasWinningMove(move BoardMove) bool {
	directions := [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	player := b.PlayerOfTurn(b.Tiles[move.Y*b.Size+move.X])
	for _, d := range directions {
		numHits := 0
		for i := -4; i <= 4; i++ {
			ix := move.X + d[0]*i
			iy := move.Y + d[1]*i

			if ix < 0 || iy < 0 || ix >= b.Size || iy >= b.Size {
				continue
			}
			tile := b.Tiles[iy*b.Size+ix]
			if tile != 0 && b.PlayerOfTurn(tile) == player {
				numHits++
			} else {
				numHits = 0
			}
			if numHits == 5 {
				return true
			}
		}
	}
	return false
}

// If, after move was taken, the game can be won by one of the players within their next turn,
// PredictWinner will return that player. Otherwise, it will return -1.
// The result is only valid if move was the last move on the board.
func (b *Board) PredictWinner(move BoardMove) int {
	player := b.PlayerOfTurn(b.Tiles[move.Y*b.Size+move.X])
	if b.WasWinningMove(move) {
		return player
	}

	if player == b.ActivePlayer() {
		// move argument was not the last move
		return -1
	}

	futureWin := true
	for otherY := 0; otherY < b.Size; otherY++ {
		for otherX := 0; otherX < b.Size; otherX++ {
			otherMove := BoardMove{otherX, otherY}
			success := b.ApplyMove(otherMove)
			if !success {
				continue
			}

			if b.WasWinningMove(otherMove) {
				winner := b.PlayerOfTurn(b.Tiles[otherMove.Y*b.Size+otherMove.X])
				b.UnapplyMove(otherMove)
				return winner
			}

			if futureWin {
				possibleFutureWin := false
			futureWinFound:
				for ownY := 0; ownY < b.Size; ownY++ {
					for ownX := 0; ownX < b.Size; ownX++ {
						ownMove := BoardMove{ownX, ownY}
						success := b.ApplyMove(ownMove)
						if !success {
							continue
						}
						if b.WasWinningMove(ownMove) {
							possibleFutureWin = true
							b.UnapplyMove(ownMove)
							break futureWinFound
						}
						b.UnapplyMove(ownMove)
					}
				}
				if !possibleFutureWin {
					futureWin = false
				}
			}
			b.UnapplyMove(otherMove)
		}
	}
	if futureWin {
		return player
	}
	return -1
}
