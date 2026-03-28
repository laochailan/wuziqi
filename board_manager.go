package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/lithammer/shortuuid/v3"
)

type Game struct {
	sync.Mutex

	board Board

	uuids     [2]string
	subscribe [2]chan *PlayerHandle
}

type GameManager struct {
	sync.RWMutex
	maxGames          int
	gameTimeout       time.Duration
	connectionTimeout time.Duration
	rateLimit         time.Duration
	maxSize           int

	uuidMap map[string]struct {
		*Game
		int
	}
}

func createGameManager(maxGames int, gameTimeout, connectionTimeout, rateLimit time.Duration, maxSize int) *GameManager {
	return &GameManager{sync.RWMutex{}, maxGames, gameTimeout, connectionTimeout, rateLimit, maxSize, make(map[string]struct {
		*Game
		int
	})}
}

func (gm *GameManager) ActiveGames() int {
	return len(gm.uuidMap) / 2
}

func (gm *GameManager) boardAndPlayer(uuid string) (*Board, int) {
	gm.RLock()
	defer gm.RUnlock()
	gameplr, ok := gm.uuidMap[uuid]
	if !ok {
		return nil, 0
	}

	gameplr.Game.Lock()
	defer gameplr.Game.Unlock()
	board := gameplr.Game.board.Clone()

	return board, gameplr.int
}

func (gm *GameManager) startGame(size int, firstUseX bool) (*[2]string, error) {
	if size > gm.maxSize {
		return nil, fmt.Errorf("requested board size above maximum: %d > %d", size, gm.maxSize)
	}

	gm.Lock()
	numGames := gm.ActiveGames()
	if numGames >= gm.maxGames {
		return nil, fmt.Errorf("too many active games")
	}

	uuids := [2]string{shortuuid.New()[1:6], shortuuid.New()[1:6]}
	game := &Game{board: *createBoard(size, firstUseX),
		uuids: uuids,
		subscribe: [2]chan *PlayerHandle{
			make(chan *PlayerHandle),
			make(chan *PlayerHandle),
		},
	}
	for i := range 2 {
		gm.uuidMap[uuids[i]] = struct {
			*Game
			int
		}{game, i}
	}
	gm.Unlock()

	go hostGame(game, gm.gameTimeout, gm.connectionTimeout, gm.rateLimit, func() {
		gm.Lock()
		defer gm.Unlock()

		for _, uuid := range uuids {
			delete(gm.uuidMap, uuid)
		}
	})
	log.Printf("started game %s: %d active/%d max", uuids[0], numGames+1, gm.maxGames)

	return &uuids, nil
}

func (gm *GameManager) joinGame(uuid string, w http.ResponseWriter, r *http.Request) error {
	gm.RLock()
	gameplr, ok := gm.uuidMap[uuid]
	gm.RUnlock()
	if !ok {
		return fmt.Errorf("player tried to join nonexisting game")
	}

	handle, err := createPlayerHandle(w, r)
	if err != nil {
		return err
	}

	go func(handle *PlayerHandle) {
		ctx, cancel := context.WithTimeoutCause(context.Background(), gm.connectionTimeout, fmt.Errorf("Join game timeout"))
		defer cancel()

		handle.Send(ctx, &HostMessage{MessageWaitForPlayerJoin, gameplr.int, &gameplr.board})

		// take back previous attempt if exists
		select {
		case <-gameplr.subscribe[gameplr.int]:
		default:
		}

		select {
		case gameplr.subscribe[gameplr.int] <- handle:
		case <-ctx.Done():
			log.Printf("player %s tried to join %s but timeout", r.RemoteAddr, uuid)
			handle.Close()
			return
		}

		log.Printf("player %s joined %s", r.RemoteAddr, uuid)

	}(handle)

	return nil
}

type PlayerHandle struct {
	conn *websocket.Conn
}

func (ph *PlayerHandle) Send(ctx context.Context, msg *HostMessage) error {
	if ph.conn == nil {
		return fmt.Errorf("attempted send on closed connection")
	}
	return wsjson.Write(ctx, ph.conn, msg)
}

func (ph *PlayerHandle) Recv(ctx context.Context, msg *PlayerMessage) error {
	if ph.conn == nil {
		return fmt.Errorf("attempted recv on closed connection")
	}
	return wsjson.Read(ctx, ph.conn, msg)
}

func createPlayerHandle(w http.ResponseWriter, r *http.Request) (*PlayerHandle, error) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &PlayerHandle{conn}, nil
}

func (p *PlayerHandle) Valid() bool {
	return p != nil && p.conn != nil
}
func (p *PlayerHandle) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.conn != nil {
		err = p.conn.CloseNow()
	}
	p.conn = nil
	return err
}

type PlayerMessageType byte

const (
	MessageMove PlayerMessageType = iota
	MessageResign
	MessageRestart
	MessageIdle
)

type PlayerMessage struct {
	Type PlayerMessageType
	Move *BoardMove
}

type HostMessageType byte

const (
	MessageStatusUpdate HostMessageType = iota
	MessageWaitForPlayerJoin
	MessageRequestTurn
	MessageGameOver
)

type HostMessage struct {
	Type   HostMessageType
	Player int
	Board  *Board
}

func waitConnect(ctx context.Context, players *[2]*PlayerHandle, subscribe [2]chan *PlayerHandle) error {
	anyInvalid := false
	for _, player := range players {
		if !player.Valid() {
			anyInvalid = true
			break
		}
	}
	if !anyInvalid {
		return nil
	}

	var wg sync.WaitGroup

	for i := range players {
		if players[i].Valid() {
			err := players[i].Send(ctx, &HostMessage{MessageWaitForPlayerJoin, i, nil})
			if err != nil {
				return err
			}
		} else {
			wg.Go(func() {
				select {
				case players[i] = <-subscribe[i]:
					log.Printf("started listening to player %d", i)
				case <-ctx.Done():
					return
				}
			})
		}
	}
	wg.Wait()
	if !players[0].Valid() || !players[1].Valid() {
		return fmt.Errorf("Could not connect to all players.")
	}
	return nil
}

func hostGame(g *Game, gameTimeout time.Duration, connectionTimeout time.Duration, rateLimit time.Duration, unregister func()) {
	gameLog := func(fmtStr string, args ...any) {
		msg := fmt.Sprintf(fmtStr, args...)
		log.Printf("[%s|%s] %s", g.uuids[0], g.uuids[1], msg)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), gameTimeout, fmt.Errorf("maximum game duration passed"))
	defer func() {
		cancel()
		gameLog("game done")
		unregister()
	}()

	var players [2]*PlayerHandle
	err := waitConnect(ctx, &players, g.subscribe)
	if err != nil {
		log.Println(err)
		return
	}
	defer players[0].Close()
	defer players[1].Close()

	msg := PlayerMessage{MessageIdle, nil}

	throttle := time.Tick(rateLimit)
	for {
		connectCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
		err := waitConnect(connectCtx, &players, g.subscribe)
		if err != nil {
			gameLog("failed to reconnect player: %v", err)
			cancel()
			return
		}
		cancel()

		g.Lock()
		switch msg.Type {
		case MessageMove:
			if msg.Move != nil && g.board.ApplyMove(*msg.Move) {
				g.board.Winner = g.board.PredictWinner(*msg.Move)
				gameLog("successful move: next turn is %d", g.board.Turn)
				if g.board.Winner != -1 {
					gameLog("player %d won!", g.board.Winner)
				}
			}
		case MessageResign:
			g.board.Resign()
			gameLog("player %d resigned!", g.board.ActivePlayer())
		case MessageRestart:
			gameLog("game restarted!")
			g.board.Reset()
		case MessageIdle:
		}
		g.Unlock()

		for i, player := range players {
			gameLog("sending state message to player %d", i)

			messageType := MessageStatusUpdate
			if g.board.Winner != -1 {
				messageType = MessageGameOver
			} else if g.board.ActivePlayer() == i {
				messageType = MessageRequestTurn
			}

			sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

			err := player.Send(sendCtx, &HostMessage{messageType, i, &g.board})
			if err != nil {
				gameLog("problem during send: %v", err)
			}
			cancel()
		}

		select {
		case <-throttle:
		case <-ctx.Done():
		}
		err = players[g.board.ActivePlayer()].Recv(ctx, &msg)
		if err != nil {
			gameLog("player %d missing", g.board.ActivePlayer())
			players[g.board.ActivePlayer()].Close()
		}
	}
}
