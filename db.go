package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func NewDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:/db/test.db")

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return db, nil
}

func InsertBoard(db *sql.DB, board *Board) error {
	query := `
		INSERT INTO boards (player1id, player2id, data) VALUES ($1, $2, $3)
	`

	b, err := json.Marshal(board)
	if err != nil {
		return err
	}

	_, err = db.Exec(query, board.PlayerIds[0], board.PlayerIds[1], b)
	return err
}

func UpdateBoard(db *sql.DB, board *Board) error {
	query := `
		UPDATE boards
		SET player1id = $1, player2id = $2, data = $3
		WHERE player1id = $1
	`
	b, err := json.Marshal(board)
	if err != nil {
		return err
	}

	res, err := db.Exec(query, board.PlayerIds[0], board.PlayerIds[1], b)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("Update affected unexpected number of rows: %d", n)
	}
	return err
}

func ReadBoard(db *sql.DB, id string) (*Board, int, error) {
	query := `
		SELECT *
		FROM boards
		WHERE player1id = $1 OR player2id = $1
	`

	playerIds := []string{"", ""}
	var data []byte

	err := db.QueryRow(query, id).Scan(&playerIds[0], &playerIds[1], &data)

	if err != nil {
		return nil, 0, err
	}

	var board Board
	err = json.Unmarshal(data, &board)

	board.Winner = board.findNextNextWinning()

	player := 0
	if playerIds[0] != id {
		player = 1
	}

	return &board, player, nil
}
