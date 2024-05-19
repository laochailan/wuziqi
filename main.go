package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alexandrevicenzi/go-sse"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Template struct {
	templ *template.Template
}

func (t Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templ.ExecuteTemplate(w, name, data)
}

func boardOpen(c echo.Context, board *Board, player int, sseServer *sse.Server) error {
	data := map[string]interface{}{
		"board":  board,
		"player": player,
	}
	return c.Render(http.StatusOK, "root", data)
}

func boardMove(c echo.Context, board *Board, player int, sseServer *sse.Server) error {
	x, err := strconv.Atoi(c.QueryParam("x"))
	if err != nil {
		return fmt.Errorf("invalid move: x: %w", err)
	}
	y, err := strconv.Atoi(c.QueryParam("y"))
	if err != nil {
		return fmt.Errorf("invalid move: y: %w", err)
	}

	if board.Turn%2 != player {
		c.Render(http.StatusOK, "waiting_board", map[string]interface{}{
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

	sseServer.SendMessage("/events/"+board.PlayerIds[0], sse.SimpleMessage("update"))

	return c.Render(http.StatusOK, next_board, map[string]interface{}{
		"board":  board,
		"player": player,
	})
}

func boardWait(c echo.Context, board *Board, player int, sseServer *sse.Server) error {
	target_turn, err := strconv.Atoi(c.QueryParam("turn"))
	if err != nil {
		return fmt.Errorf("turn invalid: %w", err)
	}

	for i := 1; i < 5; i++ {
		if target_turn > board.Turn+1 {
			return fmt.Errorf("target turn too far in the future")
		}

		if target_turn == board.Turn {
			next_board := "board"
			if board.Winner != nil {
				next_board = "winning_board"
			}
			return c.Render(http.StatusOK, next_board, map[string]interface{}{
				"board":  board,
				"player": player,
			})
		}

		select {
		case <-c.Request().Context().Done():
		case <-time.After(time.Second):
		}
	}

	return c.NoContent(http.StatusNoContent)

}

func findBoardAndPlayer(boards []Board, id string) (*Board, int, error) {
	for i := range boards {
		for playeridx, player := range boards[i].PlayerIds {
			if id == player {
				return &boards[i], playeridx, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("Game not found")
}

func withBoardAndPlayer(f func(echo.Context, *Board, int, *sse.Server) error, c echo.Context, boards []Board, id string, sseServer *sse.Server) error {
	board, player, err := findBoardAndPlayer(boards, id)
	if err != nil {

		return c.Redirect(http.StatusSeeOther, "/")
	}

	return f(c, board, player, sseServer)
}

type NewGameData struct {
	Size        int  `query:"size"`
	FirstPlayer bool `query:"first-player"`
	UseX        bool `query:"use-x"`
}

func main() {
	boards := &[]Board{}

	e := echo.New()
	e.Renderer = &Template{
		template.Must(template.ParseFiles("templates.html")),
	}

	e.Use(middleware.Logger())

	sseServer := sse.NewServer(nil)
	defer sseServer.Shutdown()

	sseServerEcho := echo.WrapHandler(sseServer)

	e.GET("/events/*", sseServerEcho)

	e.Static("/assets/*", "assets")
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "landing", nil)
	})
	e.GET("/start", func(c echo.Context) error {
		var data NewGameData
		if err := c.Bind(&data); err != nil {
			return c.String(http.StatusBadRequest, "bad request")
		}
		newBoard := createBoard(data.Size, data.FirstPlayer != data.UseX)

		own_id := newBoard.PlayerIds[0]
		other_id := newBoard.PlayerIds[1]
		if data.FirstPlayer {
			own_id, other_id = other_id, own_id
		}

		*boards = append(*boards, newBoard)
		share_link, err := url.JoinPath("http://", c.Request().Host, c.Request().URL.Path, "..", other_id, "/")
		if err != nil {
			return err
		}

		return c.Render(http.StatusOK, "landing-link", map[string]interface{}{
			"share_link": share_link,
			"own_link":   "/" + own_id + "/",
		})
	})

	e.GET("/:boardid/", func(c echo.Context) error {
		return withBoardAndPlayer(boardOpen, c, *boards, c.Param("boardid"), sseServer)
	})

	e.GET("/:boardid/move", func(c echo.Context) error {
		return withBoardAndPlayer(boardMove, c, *boards, c.Param("boardid"), sseServer)
	})

	e.GET("/:boardid/wait", func(c echo.Context) error {
		return withBoardAndPlayer(boardWait, c, *boards, c.Param("boardid"), sseServer)
	})

	e.Logger.Fatal(e.Start(":1323"))
}
