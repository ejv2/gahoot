package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ethanv2/gahoot/game"
)

// handleRoot is the handler for "/".
//
// Shows a page which allows the selection of starting a game or joining a
// game.
func handleRoot(c *gin.Context) {
	c.HTML(200, "index.gohtml", nil)
}

// handleJoin is the handler for "/join"
//
// Shows a page in which the game pin can be entered. This redirects either
// back to this page with an error or to the game page with the pin filled out.
func handleJoin(c *gin.Context) {
	dat := struct {
		Pin        game.GamePin
		PinValid   bool
		PinPresent bool
	}{}

	// Aliases for landing pages.
	joinPin := func() {
		c.HTML(200, "join.gohtml", dat)
	}
	joinNick := func() {
		c.HTML(200, "joinNick.gohtml", dat)
	}

	// If PIN provided, second stage is needed - unless there is an error,
	// which puts us back on stage one with an error
	if p := c.Query("pin"); p != "" {
		dat.PinPresent = true
		i, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			dat.PinValid = false
			joinPin()
			return
		}
		dat.Pin = game.GamePin(i)
		if dat.Pin < 1 {
			dat.PinValid = false
			joinPin()
			return
		}

		g, ok := Coordinator.GetGame(dat.Pin)
		if !ok {
			dat.PinValid = false
			joinPin()
			return
		}

		// If nick provided, second stage completed.
		// Validate nickname with game rules and then request the game
		// runner add a new player.
		if n := c.Query("nick"); n != "" {
			// Notify running game instance
			act := game.AddPlayer{Nick: n, ID: make(chan int, 1)}
			g.Action <- act
			id := int64(<-act.ID)

			c.Redirect(http.StatusTemporaryRedirect,
				"/play/game/"+p+"?plr="+strconv.FormatInt(id, 10))
			return
		}

		joinNick()
		return
	}

	joinPin()
}
