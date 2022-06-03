package main

import (
	"github.com/gin-gonic/gin"
)

// handleRoot is the handler for "/".
//
// Shows a page which allows the selection of starting a game or joining
// a game.
func handleRoot(c *gin.Context) {
	c.HTML(200, "index.gohtml", nil)
}

// handleJoin is the handler for "/join"
//
// Shows a page in which the game pin can be entered. This redirects
// either back to this page with an error or to the game page with
// the pin filled out.
func handleJoin(c *gin.Context) {
	c.HTML(200, "join.gohtml", nil)
}
