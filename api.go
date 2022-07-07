package main

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/ethanv2/gahoot/game"
)

var (
	upgrader websocket.Upgrader = websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: false,
	}
)

// handlePlayAPI is the handler for "/api/play/{PIN}"
//
// Accepts an incoming request to play a game for a specific game PIN and
// fabricates a new user from the provided data before handing off control over
// the websocket connection to the ongoing game runner.
//
// NOTE: API does not directly validate anything - simply hands off to the game
// runner. HOWEVER, this action will fail if:
//	- The game does not exist
//	- The game hit the player cap
//	- The game has ended
//
// Additionally, this call will not block if still waiting on the host. The client
// will be connected and initialised, but will be instructed to simply do nothing
// until the host is available.
func handlePlayAPI(c *gin.Context) {
	param := c.Param("pin")
	if param == "" {
		log.Panic("handlePlayApi: no PIN parameter in required handler")
	}
	pin, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		c.AbortWithStatus(400)
		log.Println("API error:", err)
		return
	}

	g, ok := Coordinator.GetGame(game.Pin(pin))
	if !ok {
		c.AbortWithStatus(404)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("play api failure:", err)
		c.Abort()
		return
	}
	log.Println("Got websocket play request from", conn.RemoteAddr(), "for", pin)

	// Hand off to the game runner, when its ready
	g.Action <- game.ConnectPlayer{Conn: conn}
}

// handleHostApi is the handler for "/api/host/{PIN}"
//
// Accepts an incoming request to host a game for a specific game PIN and
// fabricates a new host from the provided data and simply hands off to the
// game runner.
//
// NOTE: At this stage, no validation is performed. HOWEVER, this action will fail
// if:
//	- The game does not exist
//
// This websocket lasts the lifetime of a game. If the client disconnects for any
// reason (missed hearbeats or manual disconnect), the game will be immediately
// cancelled.
func handleHostAPI(c *gin.Context) {
	param := c.Param("pin")
	pin, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		c.AbortWithStatus(400)
		log.Println("API error:", err)
		return
	}

	g, ok := Coordinator.GetGame(game.Pin(pin))
	if !ok {
		c.AbortWithStatus(404)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("host api failure:", err)
		c.Abort()
		return
	}

	g.Action <- game.ConnectHost{Conn: conn}
}
