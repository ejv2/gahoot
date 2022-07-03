package game

import (
	"context"
	"time"

	"github.com/ethanv2/gahoot/game/quiz"
)

// Possible game states.
const (
	// Game is waiting for the host to connect.
	GameHostWaiting = iota
	// Game is waiting for sufficient players.
	GameWaiting
	// Game is currently live - no new players.
	GameRunning
	// Game is dead and waiting to be reaped.
	GameDead
)

// Status is either waiting, running or dead.
// See documentation on GameWaiting, GameRunning or GameDead for more
// details.
type Status int

// Gameplay constants.
const (
	MaxGameTime = time.Minute * 45
	MinPlayers  = 3
)

// StateFunc is a current state in the finite state machine of the game state.
// It returns a new StateFunc that will replace it after it returns and could
// simply be itself.
type StateFunc func() StateFunc

// State is the current state of an ongoing, running game instance. This is
// separated into a separate struct for ease of passing around combined game
// state to other objects, as well as to separate methods which act on the game
// itself and its state.
type State struct {
	Status  Status
	Host    *Host
	Players []Player
}

// Game is a single instance of a running game.
type Game struct {
	PIN Pin
	quiz.Quiz

	Action  chan Action
	Request chan chan State

	reaper chan Pin
	ctx    context.Context
	cancel context.CancelFunc
	state  State
	sf     StateFunc
}

func NewGame(pin Pin, quiz quiz.Quiz, reaper chan Pin, maxGameTime time.Duration) Game {
	if maxGameTime == 0 {
		maxGameTime = MaxGameTime
	}

	c, cancel := context.WithTimeout(context.Background(), maxGameTime)
	return Game{
		PIN:     pin,
		Quiz:    quiz,
		reaper:  reaper,
		ctx:     c,
		cancel:  cancel,
		Action:  make(chan Action),
		Request: make(chan chan State),
	}
}

// Run enters into the main game loop for this game instance, listening for events
// and mutating internal game state. Run blocks the calling goroutine until the game
// has terminated, at which point it will inform the GameCoordinator through the
// reaper channel it was initialised with.
func (game *Game) Run() {
	defer func() {
		game.state.Status = GameDead
		game.reaper <- game.PIN
		game.cancel()
	}()

	// Loop until our statefunc tells us that we are dead (nil state).
	//
	// We begin in the WaitForHost state, as the host will still be waiting
	// on our connection, after which we wait for players.
	for game.sf = game.WaitForHost; game.sf != nil; game.sf = game.sf() {
		select {
		case <-game.ctx.Done():
			return
		case act := <-game.Action:
			act.Perform(game)
		case req := <-game.Request:
			req <- game.state
		}
	}
}

// WaitForHost is the state while the host is still in the process of
// connecting.
func (game *Game) WaitForHost() StateFunc {
	game.state.Status = GameHostWaiting

	if game.state.Host != nil {
		return game.WaitForPlayers
	}
	return game.WaitForHost
}

// WaitForPlayers is the state while we are waiting on the host to start
// the game.
func (game *Game) WaitForPlayers() StateFunc {
	game.state.Status = GameWaiting
	return game.WaitForPlayers
}

// GameStarting is the state while we are showing the game start screen
// and title countdown.
func (game *Game) GameStarting() StateFunc {
	game.state.Status = GameRunning
	return nil
}

// GameEnding is the state while we are showing the game end screen and
// results summary, after which the game runner can shut down.
func (game *Game) GameEnding() StateFunc {
	return nil
}
