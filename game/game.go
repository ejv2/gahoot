package game

import (
	"context"
	"time"

	"github.com/ethanv2/gahoot/game/quiz"
)

// Possible game states
const (
	// Game is waiting for the host to connect
	GameHostWaiting = iota
	// Game is waiting for sufficient players
	GameWaiting
	// Game is currently live - no new players
	GameRunning
	// Game is dead and waiting to be reaped
	GameDead
)

// GameStatus is either waiting, running or dead.
// See documentation on GameWaiting, GameRunning or GameDead for more
// details.
type GameStatus int

// Gameplay constants
const (
	MaxGameTime = time.Minute * 45
	MinPlayers  = 3
)

// stateFunc is a current state in the finite state machine of the game state.
// It returns a new stateFunc that will replace it after it returns and could
// simply be itself
type stateFunc func() stateFunc

// GameState is the current state of an ongoing, running game instance. This is
// separated into a separate struct for ease of passing around combined game
// state to other objects, as well as to separate methods which act on the game
// itself and its state.
type GameState struct {
	Status  GameStatus
	Host    Host
	Players []Player
}

// Game is a single instance of a running game
type Game struct {
	PIN GamePin
	quiz.Quiz

	Action  chan GameAction
	Request chan chan GameState

	reaper chan GamePin
	ctx    context.Context
	cancel context.CancelFunc
	state  GameState
	sf     stateFunc
}

func NewGame(pin GamePin, reaper chan GamePin, maxGameTime time.Duration) Game {
	if maxGameTime == 0 {
		maxGameTime = MaxGameTime
	}

	c, cancel := context.WithTimeout(context.Background(), maxGameTime)
	return Game{
		PIN:     pin,
		reaper:  reaper,
		ctx:     c,
		cancel:  cancel,
		Action:  make(chan GameAction),
		Request: make(chan chan GameState),
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

func (game *Game) WaitForHost() stateFunc {
	game.state.Status = GameHostWaiting
	return game.WaitForHost
}

func (game *Game) WaitForPlayers() stateFunc {
	if len(game.state.Players) >= MinPlayers {
		return game.GameStarting
	}

	game.state.Status = GameWaiting
	return game.WaitForPlayers
}

func (game *Game) GameStarting() stateFunc {
	game.state.Status = GameRunning
	return nil
}
