package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket server message commands.
// Each websocket message is formatted as "<verb> <arg0>...<arg n>", where args
// are variadic and separated by whitespace. The "<verb>" can be any of these
// constants.
const (
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

// Client mechanism constants.
const (
	// MaxMessagesize is the maximum size of a client message in bytes.
	MaxMessagesize = 4096
	// PongInterval is the time between keepalive pings.
	PongInterval = time.Second * 15
	// PongTimeout is the maximum time allowed waiting for a keepalive pong.
	PongTimeout = time.Second * 10
)

// Client errors.
var (
	ErrorConnectionClosed = fmt.Errorf("client: connection closed")
	ErrorBadMessageType   = fmt.Errorf("client: binary message received")
	ErrorMalformedMessage = fmt.Errorf("client: invalid message syntax")
)

// A Client represents a connection from a generic client which is capable of
// both receiving and sending messages, handling pings and graceful
// disconnections.
type Client struct {
	Connected bool
	Ctx       context.Context
	Cancel    context.CancelFunc

	Send chan string

	conn     *websocket.Conn
	lastPong time.Time
}

func (c Client) writer(interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case msg := <-c.Send:
			err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				c.Cancel()
				return
			}
		case <-tick.C:
			log.Println("sending ping message to", c.conn.RemoteAddr())
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(interval))
			if err != nil {
				c.Cancel()
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

// Open sets up the websocket connection for reading as a client. Among other
// things, it sets up the read deadline and PING subsystem handlers.
//
// It is NOT necessary to call Open to use the client, and is often undesirable
// if you are not planning on taking ownership of this connection yet.
func (c Client) Open() {
	c.lastPong = time.Now()
	c.conn.SetReadDeadline(c.lastPong.Add(PongInterval).Add(PongTimeout))
	c.conn.SetPongHandler(func(appData string) error {
		latency := time.Now().Add(-PongInterval).Sub(c.lastPong)
		if latency < 0 {
			latency = 0 - latency
		}

		log.Println("got pong response with latency", latency, "from", c.conn.RemoteAddr())
		c.lastPong = time.Now()
		c.conn.SetReadDeadline(c.lastPong.Add(PongInterval).Add(PongTimeout))
		return nil
	})

	go c.writer(PongInterval)
}

// CloseReason gracefully tears down the connection with the specified teardown
// message for the client.
func (c Client) CloseReason(why string) {
	c.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, why),
		time.Now().Add(time.Second*10))
	c.conn.Close()
}

// Close gracefully tears down the connection with a generic teardown message
// for the client.
func (c Client) Close() {
	c.CloseReason("game over")
}

// Read reads the next client message into buf, handling any formatting errors
// and errors. Errors returned are fatal to the application and cannot be
// recovered. The application read loop must terminate after the first error.
func (c Client) Read(buf []byte) (int, error) {
	t, msg, err := c.conn.ReadMessage()
	switch {
	case err != nil:
		c.conn.Close()
		return 0, fmt.Errorf("client: %w", err)
	case t == websocket.PingMessage || t == websocket.PongMessage:
		return 0, nil
	case t == websocket.CloseMessage:
		return 0, ErrorConnectionClosed
	case t == websocket.BinaryMessage:
		return 0, ErrorBadMessageType
	}

	max := len(buf)
	if max > len(msg) {
		max = len(msg)
	}
	copy(buf[:max], msg)

	return len(msg), nil
}

// Write sends a message to the client, always formatted as a string. This
// method may ONLY be called after Open() and always returns successfully. An
// unsuccessful call will close the websocket connection.
func (c Client) Write(msg []byte) (int, error) {
	c.Send <- string(msg)
	return len(msg), nil
}

// ReadString reads a single message from the client and returns the verb, data
// and any errors encountered. Errors returned are usually fatal to the client.
func (c Client) ReadString() (string, string, error) {
	msg := make([]byte, MaxMessagesize)
	n, err := c.Read(msg)
	if err != nil {
		return "", "", err
	}
	msg = msg[:n]

	return StringMessage(string(msg))
}

// ReadMessage reads a single message from the client and parses according to
// the message passing scheme. Data arguments are unmarshalled into data. The
// command verb and any possible errors are returned.
func (c Client) ReadMessage(data interface{}) (string, error) {
	msg := make([]byte, MaxMessagesize)
	n, err := c.Read(msg)
	if err != nil {
		return "", err
	}
	msg = msg[:n]

	return ParseMessage(string(msg), data)
}

// StringMessage parses a given message into a verb and a data string,
// returning any errors encountered while parsing.
func StringMessage(msg string) (verb string, data string, err error) {
	fields := strings.SplitN(msg, " ", 2)
	if len(fields) != 2 {
		err = ErrorMalformedMessage
		return
	}

	verb, data, err = fields[0], fields[1], nil
	return
}

// ParseMessage parses a formatted websocket message, exctracting the attached
// JSON data and the command verb, returning the verb and unmarshalling the
// data into "data". Any encontered errors are returned and are usually fatal
// to the client.
func ParseMessage(msg string, data interface{}) (string, error) {
	verb, dat, err := StringMessage(msg)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(dat), data)
	if err != nil {
		return "", fmt.Errorf("client: data syntax: %w", err)
	}

	return verb, nil
}

// FormatMessage returns the client-readable form of the message consisting of
// the verb "command" and arguments from data in JSON form.
func FormatMessage(command string, data interface{}) string {
	var payload []byte
	if data == nil {
		payload = []byte{}
	} else {
		payload, _ = json.Marshal(data)
	}

	return command + " " + string(payload)
}
