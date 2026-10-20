package arenas

import (
	"errors"
	"testing"
)

func TestGamesSetTheirWinAndTimer(t *testing.T) {
	cases := map[Game]Arena{
		Race:   {Game: Race, Win: WinGoal, Seconds: 300},
		Flags:  {Game: Flags, Win: WinPoints, Points: 3, Seconds: 600},
		Tag:    {Game: Tag, Win: WinLast, Seconds: 180},
		Spleef: {Game: Spleef, Win: WinLast, Seconds: 300},
		Last:   {Game: Last, Win: WinLast},
	}
	for g, want := range cases {
		a := Arena{Win: WinPoints, Points: 9, Seconds: 5}
		a.Play(g)
		if a.Game != want.Game || a.Win != want.Win || a.Points != want.Points || a.Seconds != want.Seconds {
			t.Errorf("%v: got %+v", g, a)
		}
	}
}

func TestGamesKeepTheirMarks(t *testing.T) {
	var a Arena
	must(t, a.Mark(Spawns, field, Mark{Team: Red}))
	a.Play(Race)
	if len(a.Spawns) != 1 {
		t.Fatalf("arena %+v", a)
	}
}

func TestGameNames(t *testing.T) {
	for _, g := range []Game{Custom, Race, Flags, Tag, Spleef, Last} {
		if got, ok := ParseGame(g.String()); !ok || got != g {
			t.Errorf("%v round-trips to %v", g, got)
		}
	}
	if _, ok := ParseGame("chess"); ok {
		t.Error("chess is not a game")
	}
}

func TestReadyExplainsWhatIsMissing(t *testing.T) {
	spawn := func(team Team) Mark { return Mark{Team: team, X: 1, Z: 1} }
	cases := []struct {
		name   string
		game   Game
		spawns []Mark
		goals  []Mark
		want   error
	}{
		{"nothing", Last, nil, nil, ErrNoSpawn},
		{"last", Last, []Mark{spawn(NoTeam)}, nil, nil},
		{"race without a finish", Race, []Mark{spawn(NoTeam)}, nil, ErrNoGoal},
		{"race", Race, []Mark{spawn(NoTeam)}, []Mark{{}}, nil},
		{"flags alone", Flags, []Mark{spawn(Red)}, []Mark{{Team: Red}}, ErrTwoTeams},
		{"flags missing one", Flags, []Mark{spawn(Red), spawn(Blue)}, []Mark{{Team: Red}}, ErrTeamGoals},
		{"flags", Flags, []Mark{spawn(Red), spawn(Blue)}, []Mark{{Team: Red}, {Team: Blue}}, nil},
		{"tag", Tag, []Mark{spawn(NoTeam)}, nil, nil},
		{"spleef", Spleef, []Mark{spawn(NoTeam)}, nil, nil},
	}
	for _, c := range cases {
		a := Arena{Spawns: c.spawns, Goals: c.goals}
		a.Play(c.game)
		if err := a.Ready(); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}
