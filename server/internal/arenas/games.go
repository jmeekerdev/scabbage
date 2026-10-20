package arenas

import (
	"errors"
	"slices"
)

var (
	ErrNoSpawn   = errors.New("it needs a spawn")
	ErrNoGoal    = errors.New("it needs a goal")
	ErrTwoTeams  = errors.New("it needs spawns for two teams")
	ErrTeamGoals = errors.New("every team needs a goal for its flag")
	ErrBadGame   = errors.New("games are race, flags, tag, spleef, last or custom")
)

// Game is a built-in minigame whose rules an arena plays by; Custom has no special rules.
type Game uint8

const (
	Custom Game = iota
	Race        // first to the finish
	Flags       // capture-the-flag: carry another team's flag to your own goal
	Tag         // whoever is "it" tags others until one is left
	Spleef      // break the floor under others until one is left
	Last        // last one standing
)

var gameNames = []string{"custom", "race", "flags", "tag", "spleef", "last"}

func (g Game) String() string {
	if int(g) < len(gameNames) {
		return gameNames[g]
	}
	return "unknown"
}

func ParseGame(s string) (Game, bool) {
	i := slices.Index(gameNames, s)
	return Game(i), i >= 0
}

// template is how a game is won and how long its rounds last.
type template struct {
	win     Win
	points  int
	seconds int
}

var templates = map[Game]template{
	Race:   {WinGoal, 0, 300},
	Flags:  {WinPoints, 3, 600},
	Tag:    {WinLast, 0, 180},
	Spleef: {WinLast, 0, 300},
}

// Play switches the arena to a game, resetting the win and timer to the game's own.
func (a *Arena) Play(g Game) {
	t := templates[g]
	a.Game, a.Win, a.Points, a.Seconds = g, t.win, t.points, t.seconds
}

// Ready is nil when the arena has the marks its game needs, or says what is missing.
func (a Arena) Ready() error {
	switch {
	case len(a.Spawns) == 0:
		return ErrNoSpawn
	case a.Win == WinGoal && len(a.Goals) == 0:
		return ErrNoGoal
	case a.Game == Flags:
		return a.flagsReady()
	}
	return nil
}

func (a Arena) flagsReady() error {
	teams := a.Teams()
	if len(teams) < 2 {
		return ErrTwoTeams
	}
	for _, t := range teams {
		if !slices.ContainsFunc(a.Goals, func(m Mark) bool { return m.Team == t }) {
			return ErrTeamGoals
		}
	}
	return nil
}
