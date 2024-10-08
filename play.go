package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/ejv2/gahoot/game"

	"github.com/gin-gonic/gin"
)

// handleHost is the handler for "/play/host/{game PIN}".
//
// Handles validation and filling in information before returning the hoster's
// UI.
func handleHost(c *gin.Context) {
	dat := struct {
		Title          string
		Pin            uint32
		WebsocketProto string
		SiteLink       string
	}{WebsocketProto: Config.WSProto(), SiteLink: Config.SiteLink}

	spin := c.Param("pin")
	if spin == "" {
		log.Panic("handlehost: no PIN parameter in required handler")
	}

	back := func() {
		c.Redirect(http.StatusSeeOther, "/new/")
		c.Abort()
	}

	pin, err := strconv.ParseUint(spin, 10, 32)
	if err != nil {
		back()
		return
	}
	dat.Pin = uint32(pin)

	g, ok := Coordinator.GetGame(game.Pin(dat.Pin))
	if !ok {
		c.Redirect(http.StatusSeeOther, "/create/")
		c.Abort()
		return
	}
	dat.Title = g.Title

	c.HTML(200, "host.gohtml", dat)
}

// handleGame is the handler for "/play/game/{game PIN}".
//
// Handles validation and filling in information before returning the main
// frontend. At this stage, the provided UID is not validated - the websocket
// will simply fail later if this is invalid/out of range.
func handleGame(c *gin.Context) {
	dat := struct {
		// NOTE: Must be uint32, as game.GamePin is formatted as a JS string
		Pin            uint32
		UID            int
		WebsocketProto string
	}{WebsocketProto: Config.WSProto()}

	id, pin := c.Param("pin"), game.Pin(0)
	uid, intuid := c.Query("plr"), int(0)
	if id == "" {
		log.Panic("handlegame: no PIN parameter in required handler")
	}

	back := func() {
		c.Redirect(http.StatusSeeOther, "/join?pin="+id)
		c.Abort()
	}

	i, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		back()
		return
	}
	pin = game.Pin(i)
	if !Coordinator.GameExists(pin) {
		back()
		return
	}

	if uid == "" {
		back()
		return
	}
	p, err := strconv.ParseUint(uid, 10, 32)
	intuid = int(p)
	if err != nil {
		back()
		return
	}

	dat.Pin, dat.UID = uint32(i), intuid
	log.Println("user ID", intuid, "is joining game", pin)
	c.HTML(200, "play.gohtml", dat)
}
