package rounds

import (
	"slices"
	"strings"
	"testing"
	"time"

	"scabbage/internal/arenas"
)

var (
	t0    = time.Unix(1000, 0)
	lobby = arenas.Mark{X: 50, Y: 1, Z: 50}
	goal  = arenas.Mark{X: 5, Y: 1, Z: 9}
)

func arena(g arenas.Game, spawns ...arenas.Mark) arenas.Arena {
	a := arenas.Arena{Zone: "Field", Spawns: spawns, Goals: []arenas.Mark{goal}}
	a.Play(g)
	return a
}

// flags has a red and a blue spawn, each with its team's goal.
func flags() arenas.Arena {
	a := arena(arenas.Flags, spawn(arenas.Red, 1), spawn(arenas.Blue, 2))
	a.Goals = []arenas.Mark{{Team: arenas.Red, X: 1, Y: 1, Z: 5}, {Team: arenas.Blue, X: 2, Y: 1, Z: 5}}
	return a
}

func spawn(team arenas.Team, x int32) arenas.Mark { return arenas.Mark{Team: team, X: x, Y: 1} }

// said is everything the round told a player.
func said(effects []Effect, id uint32) string {
	var out []string
	for _, e := range effects {
		if e.Player == id && e.Say != "" {
			out = append(out, e.Say)
		}
	}
	return strings.Join(out, " ")
}

// sent is where the round last sent a player, if anywhere.
func sent(effects []Effect, id uint32) *arenas.Mark {
	var at *arenas.Mark
	for _, e := range effects {
		if e.Player == id && e.Move != nil {
			at = e.Move
		}
	}
	return at
}

// started joins players 1..n and runs the countdown out.
func started(t *testing.T, a arenas.Arena, n int) (*Round, []Effect) {
	t.Helper()
	r := New(a)
	for id := range uint32(n) {
		r.Join(id+1, string(rune('A'+id)), lobby, t0)
	}
	effects := r.Step(t0.Add(Countdown))
	if r.Board(t0.Add(Countdown)).Status != "Playing" {
		t.Fatalf("not playing: %+v", r.Board(t0))
	}
	return r, effects
}

func TestARaceCountsDownThenSendsRacersToTheirSpawns(t *testing.T) {
	r := New(arena(arenas.Race, spawn("", 1), spawn("", 2)))
	effects := r.Join(1, "Ann", lobby, t0)
	if !strings.Contains(said(effects, 1), "10 seconds") || r.Board(t0).Seconds != 10 {
		t.Fatalf("said %q, board %+v", said(effects, 1), r.Board(t0))
	}
	r.Join(2, "Bo", lobby, t0.Add(time.Second))
	if effects := r.Step(t0.Add(Countdown - time.Millisecond)); len(effects) != 0 {
		t.Fatalf("started early: %v", effects)
	}
	effects = r.Step(t0.Add(Countdown))
	if sent(effects, 1).X != 1 || sent(effects, 2).X != 2 || said(effects, 2) == "" {
		t.Fatalf("effects %v", effects)
	}
	if b := r.Board(t0.Add(Countdown)); b.Seconds != 300 || len(b.Scores) != 2 {
		t.Fatalf("board %+v", b)
	}
}

func TestFightingGamesWaitForTwo(t *testing.T) {
	r := New(arena(arenas.Last, spawn("", 1)))
	r.Join(1, "Ann", lobby, t0)
	if b := r.Board(t0); b.Status != "Waiting for 1 more" || b.Seconds != 0 {
		t.Fatalf("board %+v", b)
	}
	if effects := r.Step(t0.Add(time.Hour)); len(effects) != 0 {
		t.Fatalf("started alone: %v", effects)
	}
}

func TestTeamsAreDealtInTurn(t *testing.T) {
	r, effects := started(t, flags(), 3)
	teams := []arenas.Team{}
	for _, s := range r.Board(t0).Scores {
		teams = append(teams, s.Team)
	}
	if !slices.Equal(teams, []arenas.Team{arenas.Red, arenas.Blue, arenas.Red}) || sent(effects, 2).X != 2 {
		t.Fatalf("teams %v, effects %v", teams, effects)
	}
}

func TestJoiningDuringARoundIsRefused(t *testing.T) {
	r, _ := started(t, arena(arenas.Race, spawn("", 1)), 1)
	effects := r.Join(9, "Late", lobby, t0)
	if r.Has(9) || !strings.Contains(said(effects, 9), "round is on") {
		t.Fatalf("effects %v", effects)
	}
}

func TestArenasMissingMarksRefuseJoins(t *testing.T) {
	a := arena(arenas.Race, spawn("", 1))
	a.Goals = nil
	r := New(a)
	if effects := r.Join(1, "Ann", lobby, t0); r.Has(1) || !strings.Contains(said(effects, 1), "needs a goal") {
		t.Fatalf("effects %v", effects)
	}
}

func TestTheFirstRacerToTheGoalWins(t *testing.T) {
	r, _ := started(t, arena(arenas.Race, spawn("", 1)), 2)
	r.Stand(2, 1, 1, 0, true)
	effects := r.Stand(2, goal.X, goal.Y, goal.Z, true)
	if !strings.Contains(said(effects, 1), "B wins") || *sent(effects, 1) != lobby || !r.Empty() {
		t.Fatalf("effects %v", effects)
	}
}

func TestGoalsScorePointsAndSendScorersBack(t *testing.T) {
	a := arena(arenas.Custom, spawn("", 1))
	must(t, a.SetWin(arenas.WinPoints, 2))
	r, _ := started(t, a, 1)
	effects := r.Stand(1, goal.X, goal.Y, goal.Z, true)
	if sent(effects, 1).X != 1 || r.Board(t0).Scores[0].Points != 1 {
		t.Fatalf("effects %v", effects)
	}
	effects = r.Stand(1, goal.X, goal.Y, goal.Z, true)
	if !strings.Contains(said(effects, 1), "A wins") || !r.Empty() {
		t.Fatalf("effects %v", effects)
	}
}

func TestFlagsAreCarriedFromTheOtherGoalToYourOwn(t *testing.T) {
	r, _ := started(t, flags(), 2)
	if effects := r.Stand(1, 1, 1, 5, true); len(effects) != 0 {
		t.Fatalf("scored without a flag: %v", effects)
	}
	effects := r.Stand(1, 2, 1, 5, true)
	if !strings.Contains(said(effects, 2), "A has the blue flag") || r.Board(t0).Scores[0].Flag != arenas.Blue {
		t.Fatalf("effects %v", effects)
	}
	effects = r.Stand(1, 1, 1, 5, true)
	if !strings.Contains(said(effects, 2), "A scored for red") || r.Board(t0).Scores[0].Points != 1 {
		t.Fatalf("effects %v", effects)
	}
}

func TestDroppingOutDropsTheFlag(t *testing.T) {
	r, _ := started(t, flags(), 2)
	r.Stand(1, 2, 1, 5, true)
	effects, ok := r.Fall(1)
	if !ok || sent(effects, 1).X != 1 || r.Board(t0).Scores[0].Flag != "" {
		t.Fatalf("effects %v", effects)
	}
}

func TestTheLastOneStandingWins(t *testing.T) {
	r, _ := started(t, arena(arenas.Last, spawn("", 1)), 3)
	effects, _ := r.Fall(1)
	if *sent(effects, 1) != lobby || !strings.Contains(said(effects, 2), "A is out") || r.Empty() {
		t.Fatalf("effects %v", effects)
	}
	effects, _ = r.Fall(2)
	if !strings.Contains(said(effects, 1), "C wins") || *sent(effects, 3) != lobby || sent(effects, 1) != nil {
		t.Fatalf("effects %v", effects)
	}
}

func TestOutPlayersDoNotFallAgain(t *testing.T) {
	r, _ := started(t, arena(arenas.Last, spawn("", 1)), 3)
	r.Fall(1)
	if _, ok := r.Fall(1); ok {
		t.Fatal("out players respawn as usual")
	}
}

func TestFallingInARaceStartsYouOver(t *testing.T) {
	r, _ := started(t, arena(arenas.Race, spawn("", 1)), 1)
	if effects, ok := r.Fall(1); !ok || sent(effects, 1).X != 1 {
		t.Fatalf("effects %v", effects)
	}
}

func TestLeavingTheZoneCountsOnceYouAreIn(t *testing.T) {
	r, _ := started(t, arena(arenas.Spleef, spawn("", 1)), 2)
	if effects := r.Stand(1, 50, 1, 50, false); len(effects) != 0 {
		t.Fatalf("out before arriving: %v", effects)
	}
	r.Stand(1, 1, 1, 0, true)
	if effects := r.Stand(1, 50, 1, 50, false); !strings.Contains(said(effects, 1), "B wins") {
		t.Fatalf("effects %v", effects)
	}
}

func TestTimeUp(t *testing.T) {
	end := t0.Add(Countdown + 300*time.Second)
	race, _ := started(t, arena(arenas.Race, spawn("", 1)), 2)
	if effects := race.Step(end); !strings.Contains(said(effects, 1), "No one won") || !race.Empty() {
		t.Fatalf("race: %v", effects)
	}
	r, _ := started(t, flags(), 2)
	r.Stand(2, 1, 1, 5, true)
	r.Stand(2, 2, 1, 5, true)
	if effects := r.Step(t0.Add(Countdown + 600*time.Second)); !strings.Contains(said(effects, 1), "Blue team wins") {
		t.Fatalf("flags: %v", effects)
	}
	tag, _ := started(t, arena(arenas.Tag, spawn("", 1)), 2)
	if effects := tag.Step(t0.Add(Countdown + 180*time.Second)); !strings.Contains(said(effects, 1), "tie between A and B") {
		t.Fatalf("tag: %v", effects)
	}
}

func TestLeavingDuringTheCountdownStopsIt(t *testing.T) {
	r := New(arena(arenas.Last, spawn("", 1)))
	r.Join(1, "Ann", lobby, t0)
	r.Join(2, "Bo", lobby, t0)
	effects := r.Leave(2)
	if *sent(effects, 2) != lobby || r.Board(t0).Status != "Waiting for 1 more" {
		t.Fatalf("effects %v, board %+v", effects, r.Board(t0))
	}
	if effects := r.Step(t0.Add(Countdown)); len(effects) != 0 {
		t.Fatalf("started anyway: %v", effects)
	}
}

func TestLeavingAFightCanEndIt(t *testing.T) {
	r, _ := started(t, arena(arenas.Last, spawn("", 1)), 2)
	if effects := r.Leave(1); !strings.Contains(said(effects, 2), "B wins") || !r.Empty() {
		t.Fatalf("effects %v", effects)
	}
}

func TestTaggingOnlyWhileTagIsOn(t *testing.T) {
	r := New(arena(arenas.Tag, spawn("", 1)))
	if r.Tagging() {
		t.Fatal("no one is playing yet")
	}
	r, _ = started(t, arena(arenas.Tag, spawn("", 1)), 2)
	if !r.Tagging() {
		t.Fatal("tag is on")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
