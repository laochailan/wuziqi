package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

//go:embed templates.html
var templateFS embed.FS

//go:embed assets/*
var staticFS embed.FS

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
	if err != nil {
		return nil, err
	}
	firstPlayer, err := strconv.ParseBool(r.FormValue("first-player"))
	if err != nil {
		return nil, err
	}
	useX, err := strconv.ParseBool(r.FormValue("use-x"))
	if err != nil {
		return nil, err
	}

	return &NewGameData{size, firstPlayer, useX}, nil
}

func writeBadRequest(w http.ResponseWriter, err error) {
	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("bad request"))
}

func getRequestURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedScheme := r.Header.Get("X-Forwarded-Proto"); forwardedScheme != "" {
		scheme = forwardedScheme
	}
	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}
	prefix := r.Header.Get("X-Forwarded-Prefix")
	return fmt.Sprintf("%s://%s/%s/", scheme, host, prefix)
}

func main() {
	listenAddr := flag.String("listen-addr", ":8080", "the listen address for the webserver")
	gameTimeout := flag.Duration("game-timeout", time.Hour, "maximum duration for one game")
	connectionTimeout := flag.Duration("connection-timeout", 5*time.Minute, "how long to wait for players to reconnect")
	rateLimit := flag.Duration("rate-limit", 250*time.Millisecond, "minimum time between player actions")
	maxGames := flag.Int("max-games", 100, "maximum number of concurrently running games")
	maxBoardSize := flag.Int("max-board-size", 43, "maximum board size")
	flag.Parse()

	gameManager := createGameManager(
		*maxGames,
		*gameTimeout,
		*connectionTimeout,
		*rateLimit,
		*maxBoardSize,
	)

	limiter := rate.NewLimiter(rate.Limit(time.Second.Nanoseconds())/rate.Limit((*rateLimit).Nanoseconds()), 5)

	templ := template.Must(template.ParseFS(templateFS, "templates.html"))

	http.Handle("GET /assets/", http.FileServer(http.FS(staticFS)))

	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		err := templ.ExecuteTemplate(w, "landing", map[string]any{
			"baseURL":      r.Header.Get("X-Forwarded-Prefix"),
			"maxBoardSize": maxBoardSize,
		})

		if err != nil {
			writeBadRequest(w, err)
		}
	})

	http.HandleFunc("GET /start", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		data, err := parseNewGameDataForm(r)
		if err != nil {
			writeBadRequest(w, err)
			return
		}

		ids, err := gameManager.startGame(data.Size, (!data.UseX) != data.FirstPlayer)
		if err != nil {
			writeBadRequest(w, err)
		}

		if data.FirstPlayer {
			ids[0], ids[1] = ids[1], ids[0]
		}

		prefix := r.Header.Get("X-Forwarded-Prefix")
		http.Redirect(w, r, fmt.Sprintf("%s/board/%s/?share=%s", prefix, ids[0], ids[1]), http.StatusSeeOther)
	})

	http.HandleFunc("GET /board/{boardid}/{$}", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		board, _ := gameManager.boardAndPlayer(r.PathValue("boardid"))
		if board == nil {
			writeBadRequest(w, fmt.Errorf("player tried to join nonexisting game"))
			return
		}

		var share_link string

		if share := r.URL.Query().Get("share"); share != "" {
			var err error
			share_link, err = url.JoinPath(getRequestURL(r), "board", share, "/")
			if err != nil {
				writeBadRequest(w, err)
				return
			}
		}
		err := templ.ExecuteTemplate(w, "root", map[string]any{
			"baseURL":     r.Header.Get("X-Forwarded-Prefix"),
			"shareLink":   share_link,
			"ownId":       r.PathValue("boardid"),
			"boardSize":   board.Size,
			"boardWinner": board.Winner})
		if err != nil {
			writeBadRequest(w, err)
			return
		}
	})

	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		numGames := gameManager.ActiveGames()

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("%f", float32(numGames)/float32(*maxGames)))
	})

	http.HandleFunc("GET /board/{boardid}/join", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		err := gameManager.joinGame(r.PathValue("boardid"), w, r)
		if err != nil {
			log.Println(err)
		}
	})

	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
