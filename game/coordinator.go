package game

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ejv2/gahoot/game/quiz"
)

// Gameplay constants with no need for configuration.
const (
	MinGamePin = 1111111111
	MaxGamePin = 4294967295
)

// Pin is the 10-digit PIN which can be used to join a game.
// It is constrained at an int32, as that is the smallest int type which
// provides enough space without allowing any higher inputs.
type Pin uint32

// String properly formats the game PIN such that any leading digits are
// displayed properly as zeroes.
func (p Pin) String() string {
	return fmt.Sprintf("%010d", p)
}

// Validate returns true if a game pin is within the required limits.
func (p Pin) Validate() bool {
	return p < MaxGamePin && p < MinGamePin
}

// generatePin generates a pseudorandom game PIN between MinGamePin and
// MaxGamePin, ensuring to eliminate possible overflows.
func generatePin() Pin {
	return Pin(rand.Int63n(MaxGamePin-MinGamePin) + MinGamePin)
}

// Coordinator is responsible for managing all ongoing games in order to
// receive and delegate incoming events.
type Coordinator struct {
	mut        *sync.RWMutex // protects games
	games      map[Pin]Game
	reapNotify chan Pin

	maxTime time.Duration
}

// NewCoordinator allocates and returns a new game coordinator with a blank
// initial game map.
func NewCoordinator(maxGameTime time.Duration) Coordinator {
	c := Coordinator{
		mut:        new(sync.RWMutex),
		games:      make(map[Pin]Game),
		reapNotify: make(chan Pin),
		maxTime:    maxGameTime,
	}
	go c.reaper()

	return c
}

// Reaper recieves termination events from finished game instances and removes
// them from the ongoing games map. Reaper will block the calling goroutine
// until the GameCoordinator's reapNotify channel is closed (i.e the server is
// closing).
func (c *Coordinator) reaper() {
	for pin := range c.reapNotify {
		c.mut.Lock()
		delete(c.games, pin)
		c.mut.Unlock()

		log.Println("Reaper: game died:", pin)
	}
}

// CreateGame creates a new game blank game with no players waiting for a host
// connection, generating a random PIN by continually regenerating a random PIN
// until a free one is found. If the maximum concurrent games are running,
// blocks until one is available (which hopefully should occur *very* rarely).
func (c *Coordinator) CreateGame(q quiz.Quiz) Game {
	p := generatePin()
	for c.GameExists(p) {
		p = generatePin()
	}

	g := NewGame(p, q, c.reapNotify, c.maxTime)
	c.mut.Lock()
	c.games[g.PIN] = g
	c.mut.Unlock()

	// NOTE: Must return copy from map here, or subtle data race caused
	go g.Run()
	return c.games[g.PIN]
}

// GetGame does a thread safe lookup in the game map for the specified PIN.
// Arguments returned are in the Game, ok form as in default maps.
func (c Coordinator) GetGame(pin Pin) (Game, bool) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	g, ok := c.games[pin]
	return g, ok
}

// GameExists checks if a game with the specified PIN is present in the game
// map.
func (c Coordinator) GameExists(pin Pin) bool {
	_, ok := c.GetGame(pin)
	return ok
}
