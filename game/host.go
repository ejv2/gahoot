package game

import (
	"log"
	"strconv"
)

// The Host of a game is the client which receives incoming question texts and
// which handles time synchronisation for the rest of the game.
type Host struct {
	Client
}

func (h Host) Run(ev chan Action) {
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
		cmd, data, err := h.ReadString()
		if err != nil {
			log.Println("host:", err)
			h.CloseReason(err.Error())
			return
		}

		switch cmd {
		case MessageCountdown:
			time, err := strconv.ParseInt(data, 10, 32)
			if err != nil {
				log.Println("invalid countdown timer:", data)
				break
			}
			ev <- StartGame{int(time)}
		case MessageStartGame:
			ev <- StartGame{}
		case MessageKick:
			id, err := strconv.ParseInt(data, 10, 32)
			if err != nil {
				break
			}
			ev <- KickPlayer{int(id)}
		}

		select {
		case <-h.Ctx.Done():
			break readloop
		default:
		}
	}
}
