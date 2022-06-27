package game

import "log"

// The Host of a game is the client which receives incoming question texts and
// which handles type synchronisation for the rest of the game.
type Host struct {
	Client
}

func (h Host) Run(ev chan GameAction) {
	h.Open()
	defer func() {
		select {
		case ev <- EndGame{"host disconnect", false}:
		case <-h.Ctx.Done():
		}

		h.Cancel()
		log.Printf("%s (host) disconnected", h.conn.RemoteAddr())
	}()

readloop:
	for {
		cmd, _, err := h.ReadString()
		if err != nil {
			log.Println("host:", err)
			h.CloseReason(err.Error())
			return
		}

		// TODO: Go through commands here
		switch cmd {
		}

		select {
		case <-h.Ctx.Done():
			break readloop
		default:
		}
	}
}
