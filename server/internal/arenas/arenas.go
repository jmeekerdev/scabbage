// Package arenas keeps minigame setups: a zone with spawns, goals, a timer and a way to win.
package arenas

import (
	"errors"
	"slices"

	"scabbage/internal/zones"
)

const (
	MaxMarks   = 16
	MaxPoints  = 100
	MaxSeconds = 3600
)

var (
	ErrNoArena  = errors.New("no such arena")
	ErrOutside  = errors.New("marks go inside the arena's zone")
	ErrTooMany  = errors.New("too many marks")
	ErrBadTeam  = errors.New("teams are red, blue, green or yellow")
	ErrBadWin   = errors.New("arenas are won by last, goal or points 1 to 100")
	ErrBadTimer = errors.New("timers run 0 to 3600 seconds")
)

// Team is a side players are on; NoTeam is everyone for themselves.
type Team string

const (
	NoTeam Team = ""
	Red    Team = "red"
	Blue   Team = "blue"
	Green  Team = "green"
	Yellow Team = "yellow"
)

func (t Team) Valid() bool { return slices.Contains([]Team{NoTeam, Red, Blue, Green, Yellow}, t) }

// Win is how a round is won.
type Win uint8

const (
	WinLast   Win = iota // the last player or team still in
	WinGoal              // the first to reach a goal
	WinPoints            // the first to Points, or the most when time runs out
)

var winNames = []string{"last", "goal", "points"}

func (w Win) String() string {
	if int(w) < len(winNames) {
		return winNames[w]
	}
	return "unknown"
}

func ParseWin(s string) (Win, bool) {
	i := slices.Index(winNames, s)
	return Win(i), i >= 0
}

// Mark is a block in an arena that belongs to a team, or to everyone.
type Mark struct {
	Team    Team
	X, Y, Z int32
}

// Kind picks which marks to add to.
type Kind uint8

const (
	Spawns Kind = iota
	Goals
)

// Arena is a zone's minigame setup.
type Arena struct {
	Zone    string `json:"-"`
	Game    Game
	Win     Win
	Points  int  // the score that wins, with WinPoints
	Seconds int  // how long a round lasts; 0 for no limit
	PVP     bool // survival players inside may hit each other
	Spawns  []Mark
	Goals   []Mark
}

// Mark adds a spawn or goal, which must lie inside the zone.
func (a *Arena) Mark(k Kind, z zones.Zone, m Mark) error {
	marks := &a.Spawns
	if k == Goals {
		marks = &a.Goals
	}
	switch {
	case !m.Team.Valid():
		return ErrBadTeam
	case !z.Contains(m.X, m.Z):
		return ErrOutside
	case len(*marks) >= MaxMarks:
		return ErrTooMany
	}
	*marks = append(*marks, m)
	return nil
}

// SetWin picks how the arena is won; points is the winning score, and only for WinPoints.
func (a *Arena) SetWin(w Win, points int) error {
	ok := w == WinPoints && points >= 1 && points <= MaxPoints || (w == WinLast || w == WinGoal) && points == 0
	if !ok {
		return ErrBadWin
	}
	a.Win, a.Points = w, points
	return nil
}

func (a *Arena) SetTimer(seconds int) error {
	if seconds < 0 || seconds > MaxSeconds {
		return ErrBadTimer
	}
	a.Seconds = seconds
	return nil
}

// Teams lists the teams that have spawns, in the order they were first marked.
func (a Arena) Teams() []Team {
	var out []Team
	for _, m := range a.Spawns {
		if m.Team != NoTeam && !slices.Contains(out, m.Team) {
			out = append(out, m.Team)
		}
	}
	return out
}
