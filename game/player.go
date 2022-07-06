package game

import (
	"log"
)

// PlayerInfo is a message object, mirroring the PlayerData interface on
// the client.
type PlayerInfo struct {
	ID int `json:"id"`
	Nick string `json:"name"`
	Score int64 `json:"score"`
	Correct int `json:correct"`
}

// A Player is one registered player as part of a running game. Each player is
// a wrapper around a single websocket connection combined with a runner
// thread, which handles user-focused events, such as receiving answers or
// returning feedback.
//
// All exported fields may be accessed concurrently, but all unexported -
// especially "conn" - are exclusively held by the player runner, which
// translates messages from the client into actions for the game runner and
// handles ping timeouts. These must never be used, except by the player runner
// thread.
type Player struct {
	Client

	ID      int
	Nick    string
	Score   int64
	Correct int

	Banned bool
}

// Run is the game runner thread. It continually receives from the "conn"
// websocket connection until notified to stop by the connection-scoped
// context. It parses each message and takes appropriate action on each based
// on the verb passed.
func (p Player) Run(ev chan Action) {
	p.Open()
	defer func() {
		select {
		case ev <- ConnectionUpdate{p.ID, false}:
		case <-p.Ctx.Done():
		}

		p.Cancel()
		log.Printf("%s (nick: %q) disconnected", p.conn.RemoteAddr(), p.Nick)
	}()

readloop:
	for {
		cmd, _, err := p.ReadString()
		if err != nil {
			log.Println(err)
			p.CloseReason(err.Error())
			return
		}

		// TODO: Go through commands here
		switch cmd {
		}

		select {
		case <-p.Ctx.Done():
			break readloop
		default:
		}
	}

	// If we reach here, we are gracefully ending the player session
	// error-free
	p.Close()
}
