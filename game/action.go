package game

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// A Action is some action which can be sent to the game runner goroutine
// to perform an action on live game data, synchronised with the rest of the
// game. As these run on the runner thread, there is no need for any kind of
// locking. The only condition is that these actions MUST NEVER set game.sf to
// nil. Use game.GameEnding instead.
type Action interface {
	Perform(game *Game)
}

type ConnectHost struct {
	Conn *websocket.Conn
}

func (c ConnectHost) Perform(game *Game) {
	// NOTE: There is no need, for now, to enforce a deadline, as players
	// cannot connect before their host.
	cl := Client{
		Connected: true,
		conn:      c.Conn,
		send:      nil,
		Ctx:       context.Background(),
	}
	verb, _, err := cl.ReadString()
	switch {
	case err != nil:
		cl.CloseReason(err.Error())
		return
	case verb != "host":
		cl.CloseReason("expected first message to be HOST")
		return
	case game.state.Host != nil:
		cl.CloseReason("game already has a host")
		return
	}

	deadline, ok := game.ctx.Deadline()
	if !ok {
		panic("connecthost: found game with no deadline")
	}

	ctx, cancel := context.WithDeadline(game.ctx, deadline)
	game.state.Host = &Host{
		Client: Client{
			Connected: true,
			conn:      c.Conn,
			Ctx:       ctx,
			Cancel:    cancel,
			send:      make(chan string),
		},
	}

	log.Println("Host successfully joined", game.PIN.String())
	go game.state.Host.Run(game.Action)
}

// AddPlayer allocates a new slot on the server for one player to join and
// returns an ID for this player, which is the index into the players array.
// This ID will then be used by the websocket to request to join the game.
type AddPlayer struct {
	Nick string
	ID   chan int
}

func (p AddPlayer) Perform(game *Game) {
	// NOTE: Deliberately does not start the player context.
	// Runner has not yet started and the context must be re-created on
	// re-connection
	game.state.Players = append(game.state.Players, Player{
		ID:   len(game.state.Players) + 1,
		Nick: p.Nick,
		Client: Client{
			Connected: false,
			send:      make(chan string),
		},
	})

	p.ID <- len(game.state.Players)
}

// ConnectPlayer initializes a player's connection, performs the startup
// handshake asynchronously. When this is complete, submits a new action to the
// game runner to update the player's state.
type ConnectPlayer struct {
	Conn *websocket.Conn
	cl   Client
	id   uint32
	fin  bool
}

func (c ConnectPlayer) handleConnection(game Game) {
	// Enforce 30s handshake deadline to stop deadlocking of the game thread
	c.Conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	defer c.Conn.SetReadDeadline(*new(time.Time))

	// Temporary client object.
	//
	// We set Ctx to background (and implicitly Cancel to nil) here, as we
	// are not planning on calling Open, so the PING system doesn't need to
	// start.
	//
	// DO NOT call cl.Init() and DO NOT call cl.Close{,Reason} unless there
	// was a fatal error.
	c.cl = Client{
		Connected: true,
		conn:      c.Conn,
		send:      nil,
		Ctx:       context.Background(),
	}
	verb, err := c.cl.ReadMessage(&c.id)
	if err != nil {
		c.cl.CloseReason(err.Error())
		return
	}

	if verb != "ident" {
		c.cl.CloseReason("expected first message to be IDENT")
		return
	}

	c.fin = true
	game.Action <- c
}

func (c ConnectPlayer) handleInsertion(game *Game) {
	// Validate player object
	if c.id > uint32(len(game.state.Players)) {
		c.cl.CloseReason("invalid player identifier")
		return
	} else if game.state.Players[c.id-1].Connected {
		c.cl.CloseReason("given ID already connected")
		return
	}

	// Player valid
	// Go ahead and update player object
	if game.state.Players[c.id-1].Banned {
		log.Println("banned ID", c.id, "attempted rejoin: rejected")
		c.cl.CloseReason("ID banned")
		return
	} else if game.state.Host == nil {
		log.Println("ID", c.id, "attempted to join before host")
		c.cl.CloseReason("host not connected")
		return
	}

	game.state.Players[c.id-1].Connected = true
	game.state.Players[c.id-1].conn = c.Conn

	// Add context for player
	end, ok := game.ctx.Deadline()
	if !ok {
		panic("addplayer: found game with no deadline")
	}
	game.state.Players[c.id-1].Ctx,
		game.state.Players[c.id-1].Cancel = context.WithDeadline(game.ctx, end)

	log.Printf("%s (ID: %d) successfully joined %d", game.state.Players[c.id-1].Nick, c.id, game.PIN)

	// Launch player runner
	go game.state.Players[c.id-1].Run(game.Action)

	// Inform host
	inf := struct {
		ID   int    `json:"id"`
		Nick string `json:"name"`
	}{game.state.Players[c.id-1].ID, game.state.Players[c.id-1].Nick}

	game.state.Host.SendMessage(CommandNewPlayer, inf)
}

func (c ConnectPlayer) Perform(game *Game) {
	if !c.fin {
		go c.handleConnection(*game)
		return
	}

	c.handleInsertion(game)
}

// ConnectionUpdate submits a new connection state to the game loop.
//
// Used to inform the game loop of a disconnection or re-connection, if
// appropriate. This does not remove the player from the player roster, but
// does make it possible for the player to re-connect and resume.
type ConnectionUpdate struct {
	PlayerID  int
	Connected bool
}

func (c ConnectionUpdate) Perform(game *Game) {
	// PlayerID is the human-readable ID, so subtract one
	game.state.Players[c.PlayerID-1].Connected = c.Connected

	plr := game.state.Players[c.PlayerID-1]
	game.state.Host.SendMessage(CommandDisconPlayer, plr.Info())
}

// KickPlayer disconnects and bans a player ID from this game.
// This means the player will be disconnected and will be prevented from
// rejoining.
type KickPlayer struct {
	ID int
}

func (k KickPlayer) Perform(game *Game) {
	game.state.Players[k.ID-1].Connected = false
	game.state.Players[k.ID-1].Banned = true
	game.state.Players[k.ID-1].Cancel()

	plr := game.state.Players[k.ID-1]
	game.state.Host.SendMessage(CommandRemovePlayer, plr.Info())
}

// StartGame either begins a game or game countdown.
// If Count is > 0, the countdown state is activated. Else the game is
// immediately started.
type StartGame struct {
	Count int
}

func (s StartGame) Perform(game *Game) {
	if s.Count <= 0 {
		if len(game.state.Players) < MinPlayers {
			log.Println(game.PIN, "attempted to start with", len(game.state.Players), "(too few; rejected)")
		}

		game.state.Host.SendMessage(CommandStartAck, struct{}{})
		game.sf = game.Question
		game.state.Status = GameRunning

		log.Println(game.PIN, "now commencing")
		return
	}

	game.sf = game.Sustain
	for _, plr := range game.state.Players {
		go func(plr Player) {
			plr.SendMessage(CommandGameCount, struct {
				Count int    `json:"count"`
				Title string `json:"title"`
			}{s.Count, game.Title})
		}(plr)
	}

	game.sf = game.Sustain
	log.Println(game.PIN, "countdown started")
}

type NextQuestion struct{}

func (n NextQuestion) Perform(game *Game) {
	game.sf = game.Question
	game.state.CurrentQuestion++

	game.state.countdownDone = false
	game.state.acceptingAnswers = false
	game.state.questionSkipped = false
	for i := range game.state.Players {
		game.state.Players[i].canAnswer = false
		game.state.Players[i].answer = 0
	}
}

type StartAnswer struct{}

func (s StartAnswer) Perform(game *Game) {
	game.state.countdownDone = true
	game.state.answersAt = time.Now()
	go game.state.Host.SendMessage(CommandQuestionAck, struct{}{})
	for _, plr := range game.state.Players {
		plr.SendMessage(CommandNewQuestion, game.Questions[game.state.CurrentQuestion])
	}
}

type EndAnswer struct{}

func (s EndAnswer) Perform(game *Game) {
	game.state.questionSkipped = true
}

type Answer struct {
	PlayerID, Number int
}

func (a Answer) Perform(game *Game) {
	if a.Number < 1 {
		panic("answer: invalid answer: less than 1")
	}
	if !game.state.acceptingAnswers {
		log.Printf("%d attempted to answer out of answer time [%s]", a.PlayerID, game.PIN)
		return
	}
	if a.PlayerID <= 0 || a.PlayerID > len(game.state.Players) {
		log.Printf("invalid player attempted to answer (ID: %d) [%s]", a.PlayerID, game.PIN)
		return
	}
	if !game.state.Players[a.PlayerID-1].canAnswer {
		log.Printf("%d attempted to steal an answer slot [%s]", a.PlayerID, game.PIN)
		return
	}
	if game.state.Players[a.PlayerID-1].answer > 0 {
		log.Printf("%d attempted multiple answer [%s]", a.PlayerID, game.PIN)
		return
	}

	log.Println(a.PlayerID, "answered", game.state.CurrentQuestion, "with", a.Number, "in", game.PIN)

	game.state.Players[a.PlayerID-1].answer = a.Number
	game.state.Players[a.PlayerID-1].answeredAt = time.Now()
}

// EndGame shuts down the game runner, thereby terminating the current
// game.
//
// If the shutdown is clean, the state is merely shifted to the GameEnding
// state, which allows for the final leaderboard to be shown.
// If the shutdown was NOT clean, the state is immediately set to nil and
// the game runner shuts down on the spot. This is usually used when the
// host disconnects.
type EndGame struct {
	Reason string
	Clean  bool
}

func (e EndGame) Perform(game *Game) {
	if !e.Clean {
		game.cancel()
		game.sf = game.GameTerminate
		return
	}

	game.sf = game.GameEnding
}
