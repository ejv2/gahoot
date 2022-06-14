package game

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

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
	Nick    string
	Score   int64
	Correct int

	Connected bool
	Ctx       context.Context

	lastPing time.Time
	conn     *websocket.Conn
}

func pingHandler() {

}

// Run is the game runner thread. It continually receives from the "conn"
// websocket connection until notified to stop by "ctx"
func (p Player) Run(ev chan GameAction) {
	bail := func(why string) {
		p.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, why),
			time.Now().Add(time.Second))
		p.conn.Close()
	}
	end, _ := p.Ctx.Deadline()
	p.conn.SetReadDeadline(end)

readloop:
	for {
		t, r, err := p.conn.NextReader()
		switch {
		case err != nil:
			p.conn.Close()
			break readloop
		case t != websocket.TextMessage:
			bail("expected text messages, got binary")
			break readloop
		}
		msg := new(strings.Builder)
		io.Copy(msg, r)
		log.Println("got message from", p.conn.RemoteAddr().String(), ":", msg)

		select {
		case <-p.Ctx.Done():
			break
		default:
		}
	}
}
