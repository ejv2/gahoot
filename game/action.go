package game

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// A Action is some action which can be sent to the game runner goroutine
// to perform an action on live game data, synchronised with the rest of the
// game. As these run on the runner thread, there is no need for any kind of
// locking.
type Action interface {
	Perform(game *Game)
}

type ConnectHost struct {
	Conn *websocket.Conn
}

func (c ConnectHost) Perform(game *Game) {
	// NOTE: There is no need, for now, to enforce a deadline, as players
	// cannot connect before their host.
	cl := Client{
		Connected: true,
		conn:      c.Conn,
		Send:      nil,
		Ctx:       context.Background(),
	}
	verb, _, err := cl.ReadString()
	switch {
	case err != nil:
		cl.CloseReason(err.Error())
		return
	case verb != "host":
		cl.CloseReason("expected first message to be HOST")
		return
	case game.state.Host != nil:
		cl.CloseReason("game already has a host")
		return
	}

	deadline, ok := game.ctx.Deadline()
	if !ok {
		panic("connecthost: found game with no deadline")
	}

	ctx, cancel := context.WithDeadline(game.ctx, deadline)
	game.state.Host = &Host{
		Client: Client{
			Connected: true,
			conn:      c.Conn,
			Ctx:       ctx,
			Cancel:    cancel,
			Send:      make(chan string),
		},
	}

	log.Println("Host successfully joined", game.PIN.String())
	go game.state.Host.Run(game.Action)
}

// AddPlayer allocates a new slot on the server for one player to join and
// returns an ID for this player, which is the index into the players array.
// This ID will then be used by the websocket to request to join the game.
type AddPlayer struct {
	Nick string
	ID   chan int
}

func (p AddPlayer) Perform(game *Game) {
	// NOTE: Deliberately does not start the player context.
	// Runner has not yet started and the context must be re-created on
	// re-connection
	game.state.Players = append(game.state.Players, Player{
		ID:   len(game.state.Players) + 1,
		Nick: p.Nick,
		Client: Client{
			Connected: false,
			Send:      make(chan string),
		},
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
	// Enforce 30s handshake deadline to stop deadlocking of the game thread
	c.Conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	defer c.Conn.SetReadDeadline(*new(time.Time))

	// Temporary client object.
	//
	// We set Ctx to background (and implicitly Cancel to nil) here, as we
	// are not planning on calling Open, so the PING system doesn't need to
	// start.
	//
	// DO NOT call cl.Init() and DO NOT call cl.Close{,Reason} unless there
	// was a fatal error.
	cl := Client{
		Connected: true,
		conn:      c.Conn,
		Send:      nil,
		Ctx:       context.Background(),
	}
	var id int32
	verb, err := cl.ReadMessage(&id)
	if err != nil {
		cl.CloseReason(err.Error())
		return
	}

	if verb != "ident" {
		cl.CloseReason("expected first message to be IDENT")
		return
	}

	// Validate player object
	if id > int32(len(game.state.Players)) || id <= 0 {
		cl.CloseReason("invalid player identifier")
		return
	} else if game.state.Players[id-1].Connected {
		cl.CloseReason("given ID already connected")
		return
	}

	// Player valid
	// Go ahead and update player object
	game.state.Players[id-1].Connected = true
	game.state.Players[id-1].conn = c.Conn

	// Add context for player
	end, ok := game.ctx.Deadline()
	if !ok {
		panic("addplayer: found game with no deadline")
	}
	game.state.Players[id-1].Ctx,
		game.state.Players[id-1].Cancel = context.WithDeadline(game.ctx, end)

	log.Printf("%s (ID: %d) successfully joined %d", game.state.Players[id-1].Nick, id, game.PIN)

	// Launch player runner
	go game.state.Players[id-1].Run(game.Action)
}

// ConnectionUpdate submits a new connection state to the game loop.
//
// Used to inform the game loop of a disconnection or re-connection, if
// appropriate. This does not remove the player from the player roster, but
// does make it possible for the player to re-connect and resume.
type ConnectionUpdate struct {
	PlayerID  int
	Connected bool
}

func (c ConnectionUpdate) Perform(game *Game) {
	// PlayerID is the human-readable ID, so subtract one
	game.state.Players[c.PlayerID-1].Connected = c.Connected
}

// EndGame shuts down the game runner, thereby terminating the current
// game.
//
// If the shutdown is clean, the state is merely shifted to the GameEnding
// state, which allows for the final leaderboard to be shown.
// If the shutdown was NOT clean, the state is immediately set to nil and
// the game runner shuts down on the spot. This is usually used when the
// host disconnects.
type EndGame struct {
	Reason string
	Clean  bool
}

func (e EndGame) Perform(game *Game) {
	if e.Clean {
		game.sf = game.GameEnding
		return
	}

	game.sf = nil
	game.cancel()
}
