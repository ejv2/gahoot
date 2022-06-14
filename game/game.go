package game

import (
	"context"
	"time"
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
	MaximumGameTime = time.Minute * 45
)

// GameState is the current state of an ongoing, running game instance. This is
// separated into a separate struct for ease of passing around combined game
// state to other objects, as well as to separate methods which act on the game
// itself and its state.
type GameState struct {
	Status  GameStatus
	Players []Player
}

// Game is a single instance of a running game
type Game struct {
	PIN GamePin

	Action  chan GameAction
	Request chan chan GameState

	reaper chan GamePin
	ctx    context.Context
	cancel context.CancelFunc
	state  GameState
}

func NewGame(pin GamePin, reaper chan GamePin, maxGameTime time.Duration) Game {
	if maxGameTime == 0 {
		maxGameTime = MaximumGameTime
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

	for {
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
