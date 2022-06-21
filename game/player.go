package game

import (
	"context"
	"log"
	"time"
	"encoding/json"

	"github.com/gorilla/websocket"
)

// WebSocket server message commands.
// Each websocket message is formatted as "<verb> <arg0>...<arg n>", where args
// are variadic and separated by whitespace. The "<verb>" can be any of these
// constants.
const (
	CommandGameOver     = "end"
	CommandQuestionOver = "qend"
	CommandNewQuestion  = "ques"
	CommandNewOptions   = "opts"
	CommandSeeResults   = "res"
)

// WebSocket client message commands.
// Client equivalent of server message commands. See documentation for server
// message commands for more details on format.
const (
	MessageAcknowledge = "ack"
	MessageIdenfity    = "ident"
	MessageAnswer      = "ans"
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
	ID      int
	Nick    string
	Score   int64
	Correct int

	Connected bool
	Ctx       context.Context
	Cancel    context.CancelFunc

	Send chan string
	update chan<- GameAction
	conn *websocket.Conn
}

func (p Player) writer(interval time.Duration) {
	for {
		select {
		case msg := <-p.Send:
			err := p.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				p.Cancel()
				return
			}
		case <-time.After(interval):
			log.Println("sending ping message to", p.conn.RemoteAddr())
			err := p.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(interval))
			if err != nil {
				p.Cancel()
				return
			}
		case <-p.Ctx.Done():
			return

		}
	}
}

func (p Player) updateConnected(connected bool) {
	select {
	case p.update <- ConnectionUpdate{p.ID, connected}:
	case <-p.Ctx.Done():
	}
}

// Run is the game runner thread. It continually receives from the "conn"
// websocket connection until notified to stop by "ctx"
func (p Player) Run(ev chan GameAction) {
	bail := func(why string) {
		log.Println("websocket: closing due to error:", why)
		p.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, why),
			time.Now().Add(time.Second*10))
		p.conn.Close()
	}

	lastPong := time.Now()
	pongInterval, pongTimeout := time.Second*15, time.Second*10
	p.conn.SetReadDeadline(lastPong.Add(pongInterval).Add(pongTimeout))
	p.conn.SetPongHandler(func(appData string) error {
		latency := time.Now().Add(-pongInterval).Sub(lastPong)
		if latency < 0 {
			latency = 0 - latency
		}

		log.Println("got pong response with latency", latency, "from", p.conn.RemoteAddr())
		lastPong = time.Now()
		p.conn.SetReadDeadline(lastPong.Add(pongInterval).Add(pongTimeout))
		return nil
	})

	go p.writer(pongInterval)
	defer func() {
		p.Cancel()
		p.updateConnected(false)
		log.Printf("%s (nick: %q) disconnected", p.conn.RemoteAddr(), p.Nick)
	}()

readloop:
	for {
		t, msg, err := p.conn.ReadMessage()
		switch {
		case err != nil:
			p.conn.Close()
			return
		case t == websocket.PingMessage || t == websocket.PongMessage:
			continue readloop
		case t != websocket.TextMessage:
			bail("expected text messages, got binary")
			return
		}
		log.Println("got message from", p.conn.RemoteAddr().String(), ":", msg)

		select {
		case <-p.Ctx.Done():
			break readloop
		default:
		}
	}

	// If we got here, the game is done and the player instance is exiting
	// Send the final goodbyes
	p.Send <- CommandGameOver
	p.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "game over"),
		time.Now().Add(time.Second*10))
	p.conn.Close()
}

func FormatMessage(command string, data interface{}) string {
	var payload []byte
	if data == nil {
		payload = []byte{}
	} else {
		payload, _ = json.Marshal(data)
	}

	return command + " " + string(payload)
}
