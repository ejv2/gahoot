package game

import (
	"context"
	"math"
	"time"

	"github.com/ejv2/gahoot/game/quiz"
)

// Possible game states.
const (
	// Game is waiting for the host to connect.
	GameHostWaiting = iota
	// Game is waiting for sufficient players.
	GameWaiting
	// Game is currently live.
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
	MaxGameTime    = time.Minute * 45
	MinPlayers     = 3
	BasePoints     = 1000
	StreakBonus    = 100
	MaxStreakBonus = 500
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
	Status          Status
	Host            *Host
	Players         []Player
	CurrentQuestion int

	// Caches the used names in the current game.
	namecache map[string]struct{}

	// Has the host completed the countdown?
	countdownDone bool
	// Curently in answer time?
	acceptingAnswers bool
	// Is this the last player?
	// Used to prevent client race between ansack and qend messages.
	lastPlayer bool
	// Has the host skipped the question?
	questionSkipped bool
	// Time at which answers begin being accepted.
	// Used to calculate points bonus from time taken.
	answersAt time.Time
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

// Score returns the number of points that should be awarded for an answer.
// The rules which govern this are as follows:
//  1. No points are awarded for an incorrect answer
//  2. As the time taken to answer tends toward zero, points tend toward
//     2*base.
//  3. 100 additional points are awarded for each correct answer in a row
//     (streak), up to a maximum of 500 bonus streak points
//  4. At half time, no bonus points are awarded. After half time, bonus points
//     are negative up to -base (no points awarded at all, not counting
//     bonuses)
//
// This function does all calculations as floating points, but truncates the
// result in the end.
// If taken > allowed or taken < 0, Score panics (these must never be allowed
// to happen).
func Score(correct bool, base, streak int, taken, allowed time.Duration) int64 {
	if !correct || allowed == 0 {
		return 0
	}
	if taken > allowed || taken < 0 {
		panic("score: invalid time taken while scoring (took " + allowed.String() + "?)")
	}

	start := float64(base)
	start += float64(base) * (1.0 - (float64(taken) / (float64(allowed) / 2)))
	start += math.Min(MaxStreakBonus, StreakBonus*float64(streak))
	return int64(start)
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
		game.state.Status = GameWaiting
		return game.Sustain
	}
	return game.WaitForHost
}

// Question is active when the game is showing a question but BEFORE we
// are accepting answers. At the point where the countdown is finished, a
// snapshot of the current player state is taken such that new players do
// not disrupt the existing players' game.
func (game *Game) Question() StateFunc {
	if game.state.countdownDone {
		game.state.acceptingAnswers = true
		return game.AcceptAnswers
	}

	q := struct {
		quiz.Question
		Index int `json:"index"`
		Total int `json:"total"`
	}{
		game.Questions[game.state.CurrentQuestion],
		game.state.CurrentQuestion + 1,
		len(game.Questions),
	}
	go game.state.Host.SendMessage(CommandNewQuestion, q)

	for i, plr := range game.state.Players {
		if plr.Connected {
			game.state.Players[i].canAnswer = true
			go plr.SendMessage(CommandQuestionCount, struct {
				Count int `json:"count"`
			}{5})
		}
	}

	return game.sf
}

// AcceptAnswers is active when the game is idle accepting answers until
//  1. Every player has answered
//  2. The time runs out (host will notify us)
//  3. The host manually skips the question (host will notify us)
func (game *Game) AcceptAnswers() StateFunc {
	type feedback struct {
		Info    PlayerInfo `json:"leaderboard"`
		Correct bool       `json:"correct"`
		Points  int64      `json:"points"`
	}

	pending, count := false, 0
	for _, plr := range game.state.Players {
		if plr.Connected && plr.canAnswer && plr.answer <= 0 {
			pending = true
			count++
		}
	}
	// Only waiting on one player, so set lastPlayer flag
	if count == 1 {
		game.state.lastPlayer = true
	}

	game.state.acceptingAnswers = true
	if !pending || game.state.questionSkipped {
		dats := make([]feedback, len(game.state.Players))

		game.state.acceptingAnswers = false
		game.state.countdownDone = false
		game.state.lastPlayer = false

		clip := 6
		if clip > len(game.state.Players) {
			clip = len(game.state.Players)
		}

		game.state.Host.SendMessage(CommandQuestionOver, struct{}{})

		for i, plr := range game.state.Players {
			correct := false
			dur := 0

			if plr.answer > 0 {
				correct = game.Questions[game.state.CurrentQuestion].Answers[plr.answer-1].Correct
				dur = game.Questions[game.state.CurrentQuestion].Duration
			}

			if correct {
				game.state.Players[i].Correct++
				game.state.Players[i].Streak++
			} else {
				game.state.Players[i].Streak = 0
			}

			score := Score(correct, BasePoints, game.state.Players[i].Streak, plr.answeredAt.Sub(game.state.answersAt), time.Duration(dur)*time.Second)
			game.state.Players[i].Score += score
			dats[i] = feedback{
				Info:    game.state.Players[i].Info(),
				Correct: correct,
				Points:  score,
			}
			plr.SendMessage(CommandQuestionOver, dats[i])
		}

		board := NewLeaderboard(game.state.Players)
		game.state.Host.SendMessage(CommandSeeResults, board[:clip])
		return game.Sustain
	}

	return game.AcceptAnswers
}

// GameEnding is the state while we are showing the game end screen and
// results summary, after which the game runner can shut down. It accepts
// one more message, which is the host communicating that it is finished.
func (game *Game) GameEnding() StateFunc {
	return nil
}

// GameTerminate terminates the game loop on next iteration.
func (game *Game) GameTerminate() StateFunc {
	return nil
}

// Sustain keeps the game in the current state.
func (game *Game) Sustain() StateFunc {
	return game.sf
}
