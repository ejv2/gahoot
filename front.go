package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ethanv2/gahoot/game"
	"github.com/ethanv2/gahoot/game/quiz"
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
		Pin        game.Pin
		PinValid   bool
		PinPresent bool
	}{}
	// Aliases for landing pages.
	joinPin := func() {
		c.HTML(200, "join.gohtml", dat)
	}
	joinNick := func() {
		c.HTML(200, "join_nick.gohtml", dat)
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
		dat.Pin = game.Pin(i)
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

			c.Redirect(http.StatusSeeOther,
				"/play/game/"+p+"?plr="+strconv.FormatInt(id, 10))
			return
		}

		joinNick()
		return
	}

	joinPin()
}

func handleCreate(c *gin.Context) {
	c.HTML(200, "create.gohtml", nil)
}

func handleFind(c *gin.Context) {
	dat := struct {
		Quizzes    []quiz.Quiz
		Categories []string
	}{
		Quizzes:    QuizManager.GetAll(),
		Categories: QuizManager.GetCategories(),
	}

	c.HTML(200, "create_find.gohtml", dat)
}

func handleUpload(c *gin.Context) {
	dat := struct {
		// Maximum file size in MB
		FileSize int64
	}{
		FileSize: quiz.MaxQuizSize / (1024 * 1024),
	}

	c.HTML(200, "create_upload.gohtml", dat)
}

func handleEditor(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func handleCreateGame(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		log.Panic("handleCreateGame: no hash parameter in required handler")
	}

	c.String(200, "text/plain", "Coming soon: will create game "+hash)
}

func handleBlankCreateGame(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/create/")
}
