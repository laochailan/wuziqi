package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alexandrevicenzi/go-sse"
	_ "github.com/ncruces/go-sqlite3/driver"
)

type Context struct {
	w         http.ResponseWriter
	r         *http.Request
	templ     *template.Template
	sseServer *sse.Server
	db        *sql.DB
}

func (c *Context) Render(name string, data interface{}) error {
	return c.templ.ExecuteTemplate(c.w, name, data)
}

func boardOpen(c Context, board *Board, player int) error {
	data := map[string]interface{}{
		"board":  board,
		"player": player,
	}
	return c.Render("root", data)
}

func boardMove(c Context, board *Board, player int) error {
	query := c.r.URL.Query()
	x, err := strconv.Atoi(query.Get("x"))
	if err != nil {
		return fmt.Errorf("invalid move: x: %w", err)
	}
	y, err := strconv.Atoi(query.Get("y"))
	if err != nil {
		return fmt.Errorf("invalid move: y: %w", err)
	}

	if board.Turn%2 != player {
		c.Render("waiting_board", map[string]interface{}{
			"board":  board,
			"player": player,
		})
	}

	if board.Tiles[y][x] == 0 {
		board.Tiles[y][x] = board.Turn
		board.Turn += 1
	} else {
		return fmt.Errorf("invalid move")
	}

	board.Winner = board.findNextNextWinning()

	next_board := "waiting_board"
	if board.Winner != nil {
		next_board = "winning_board"
	}

	err = UpdateBoard(c.db, board)
	if err != nil {
		log.Println(err)
	}
	c.sseServer.SendMessage("/events/"+board.PlayerIds[0], sse.SimpleMessage("update"))

	return c.Render(next_board, map[string]interface{}{
		"board":  board,
		"player": player,
	})
}

func boardWait(c Context, board *Board, player int) error {
	query := c.r.URL.Query()
	target_turn, err := strconv.Atoi(query.Get("turn"))
	if err != nil {
		return fmt.Errorf("turn invalid: %w", err)
	}

	for i := 1; i < 5; i++ {
		board, player, err = ReadBoard(c.db, board.PlayerIds[player])
		if err != nil {
			return fmt.Errorf("could not find board: %w", err)
		}
		if target_turn > board.Turn+1 {
			return fmt.Errorf("target turn too far in the future")
		}

		if target_turn == board.Turn {
			next_board := "board"
			if board.Winner != nil {
				next_board = "winning_board"
			}
			return c.Render(next_board, map[string]interface{}{
				"board":  board,
				"player": player,
			})
		}

		select {
		case <-c.r.Context().Done():
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf("could not find new turn")
}

func withBoardAndPlayer(f func(Context, *Board, int) error, c Context) error {
	id := c.r.PathValue("boardid")
	board, player, err := ReadBoard(c.db, id)
	if err != nil {
		log.Println(err)
		http.Redirect(c.w, c.r, "/", http.StatusSeeOther)
	}

	err = f(c, board, player)
	if err != nil {
		log.Println(err)
		return err
	}
	return err
}

type NewGameData struct {
	Size        int
	FirstPlayer bool
	UseX        bool
}

func parseNewGameDataForm(r *http.Request) (*NewGameData, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	size, err := strconv.Atoi(r.FormValue("size"))
	firstPlayer, err := strconv.ParseBool(r.FormValue("first-player"))
	useX, err := strconv.ParseBool(r.FormValue("use-x"))

	return &NewGameData{size, firstPlayer, useX}, nil
}

func writeBadRequest(w http.ResponseWriter, err error) {
	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("bad request"))
}

func main() {
	db, err := NewDB()
	if err != nil {
		log.Fatal(err)
	}
	db.Ping()

	templ := template.Must(template.ParseFiles("templates.html"))

	sseServer := sse.NewServer(nil)
	defer sseServer.Shutdown()

	http.Handle("GET /events/", sseServer)
	http.Handle("GET /assets/", http.FileServer(http.Dir(".")))

	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		err := templ.ExecuteTemplate(w, "landing", nil)

		if err != nil {
			writeBadRequest(w, err)
		}
	})

	http.HandleFunc("GET /start", func(w http.ResponseWriter, r *http.Request) {
		data, err := parseNewGameDataForm(r)
		if err != nil {
			writeBadRequest(w, err)
			return
		}

		newBoard := createBoard(data.Size, data.FirstPlayer != data.UseX)

		own_id := newBoard.PlayerIds[0]
		other_id := newBoard.PlayerIds[1]
		if data.FirstPlayer {
			own_id, other_id = other_id, own_id
		}

		err = InsertBoard(db, &newBoard)
		if err != nil {
			writeBadRequest(w, err)
		}

		share_link, err := url.JoinPath("http://", r.Host, r.URL.Path, "../board/", other_id, "/")
		if err != nil {
			writeBadRequest(w, err)
			return
		}

		err = templ.ExecuteTemplate(w, "landing-link", map[string]interface{}{
			"share_link": share_link,
			"own_link":   "/board/" + own_id + "/",
		})

		if err != nil {
			writeBadRequest(w, err)
		}
	})

	http.HandleFunc("GET /board/{boardid}/{$}", func(w http.ResponseWriter, r *http.Request) {
		withBoardAndPlayer(boardOpen, Context{w, r, templ, sseServer, db})
	})

	http.HandleFunc("GET /board/{boardid}/move", func(w http.ResponseWriter, r *http.Request) {
		withBoardAndPlayer(boardMove, Context{w, r, templ, sseServer, db})
	})
	http.HandleFunc("GET /board/{boardid}/wait", func(w http.ResponseWriter, r *http.Request) {
		withBoardAndPlayer(boardWait, Context{w, r, templ, sseServer, db})
	})

	log.Fatal(http.ListenAndServe(":1323", nil))
}
