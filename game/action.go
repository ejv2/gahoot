package game

import (
	"time"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// A GameAction is some action which can be sent to the game runner goroutine
// to perform an action on live game data, synchronised with the rest of the
// game. As these run on the runner thread, there is no need for any kind of
// locking.
type GameAction interface {
	Perform(game *Game)
}

// AddPlayer allocates a new slot on the server for one player to join and
// returns an ID for this player, which is the index into the players array.
// This ID will then be used by the websocket to request to join the game.
type AddPlayer struct {
	Nick string
	ID   chan int
}

func (p AddPlayer) Perform(game *Game) {
	game.state.Players = append(game.state.Players, Player{
		Nick:      p.Nick,
		Connected: false,
		Ctx:       game.ctx,
	})

	p.ID <- len(game.state.Players)
}

// TODO: This blocks the game thread until the player has joined.
//
// For now, "fixed" this by applying a freakishly short 30s authentication deadline,
// which screws over people with bad internet.
// To fix, create two methods on ConnectPlayer, one which starts a goroutine to handle
// the handshake and one to add the player on the game thread. After the one in the
// goroutine has finished, it enqueues *itself* in the game's Action queue.
type ConnectPlayer struct {
	Conn *websocket.Conn
}

func (c ConnectPlayer) Perform(game *Game) {
	bail := func(why string) {
		c.Conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, why),
			time.Now().Add(time.Second))
		c.Conn.Close()
	}

	// Enforce 30s handshake deadline to stop deadlocking of the game thread
	c.Conn.SetReadDeadline(time.Now().Add(time.Second*30))
	defer c.Conn.SetReadDeadline(*new(time.Time))

	t, b, err := c.Conn.ReadMessage()
	switch {
	case t != websocket.TextMessage:
		bail("expected text messages, got binary")
		return
	case err != nil:
		c.Conn.Close()
		return
	}

	msg := string(b)
	fields := strings.SplitN(msg, " ", 2)
	if len(fields) != 2 {
		bail("expected two fields per message")
		return
	}
	if fields[0] != "ident" {
		bail("expected first message to be IDENT")
		return
	}
	p, err := strconv.ParseUint(fields[1], 10, 32)
	id := int(p)
	if err != nil {
		bail("invalid player ID")
	}

	// Validate player object
	if id > len(game.state.Players) || id <= 0 {
		bail("invalid player identifier")
		return
	} else if game.state.Players[id].Connected {
		bail("given ID already connected")
		return
	}

	// Player valid
	// Go ahead and update player object
	game.state.Players[id-1].Connected = true
	game.state.Players[id-1].conn = c.Conn

	// Launch player runner
	go game.state.Players[id-1].Run(game.Action)
}
