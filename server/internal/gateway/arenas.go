package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"scabbage/internal/arenas"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/zones"
)

const arenaHelp = "Arenas: /arena spawn ZONE [TEAM], /arena goal ZONE [TEAM], " +
	"/arena game ZONE race|flags|tag|spleef|last|custom, /arena timer ZONE SECONDS, " +
	"/arena win ZONE last|goal|points N, /arena clear ZONE, /arena list"

// arenaCommand runs an admin's /arena command, which sets up a minigame in a zone.
func (s *session) arenaCommand(ctx context.Context, arg string) error {
	f := strings.Fields(arg)
	if len(f) == 0 || f[0] == "list" {
		return s.listArenas(ctx)
	}
	if len(f) < 2 {
		return s.notice(ctx, arenaHelp)
	}
	zone, ok := s.gw.cfg.Zones.Named(f[1])
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", f[1]))
	}
	if f[0] == "clear" {
		return s.clearArena(ctx, zone.Name)
	}
	change, ok := s.arenaChange(f[0], zone, f[2:])
	if !ok {
		return s.notice(ctx, arenaHelp)
	}
	return s.changeArena(ctx, zone.Name, change)
}

// arenaChange is the edit an /arena verb makes, if the verb and its arguments make sense.
func (s *session) arenaChange(verb string, zone zones.Zone, args []string) (func(*arenas.Arena) error, bool) {
	switch {
	case (verb == "spawn" || verb == "goal") && len(args) <= 1:
		kind, team := arenas.Spawns, arenas.NoTeam
		if verb == "goal" {
			kind = arenas.Goals
		}
		if len(args) == 1 {
			team = arenas.Team(args[0])
		}
		m := s.feetMark(team)
		return func(a *arenas.Arena) error { return a.Mark(kind, zone, m) }, true
	case verb == "game" && len(args) == 1:
		g, known := arenas.ParseGame(args[0])
		return func(a *arenas.Arena) error {
			if !known {
				return arenas.ErrBadGame
			}
			a.Play(g)
			return nil
		}, true
	case verb == "timer" && len(args) == 1:
		n, err := strconv.Atoi(args[0])
		if err != nil {
			n = -1
		}
		return func(a *arenas.Arena) error { return a.SetTimer(n) }, true
	case verb == "win" && len(args) >= 1 && len(args) <= 2:
		w, known := arenas.ParseWin(args[0])
		n := 0
		if len(args) == 2 {
			n, _ = strconv.Atoi(args[1])
		}
		return func(a *arenas.Arena) error {
			if !known {
				return arenas.ErrBadWin
			}
			return a.SetWin(w, n)
		}, true
	}
	return nil, false
}

// feetMark is a mark for the block the admin stands in.
func (s *session) feetMark(team arenas.Team) arenas.Mark {
	at := s.body.Pos()
	return arenas.Mark{Team: team, X: int32(math.Floor(at.X)), Y: int32(math.Floor(at.Y)), Z: int32(math.Floor(at.Z))}
}

// changeArena edits a zone's arena, making one if needed, and tells the admin how it now looks.
func (s *session) changeArena(ctx context.Context, zone string, change func(*arenas.Arena) error) error {
	a, _, err := s.gw.cfg.Arenas.Get(zone)
	if err != nil {
		return err
	}
	a.Zone = zone
	if err := change(&a); err != nil {
		return s.notice(ctx, sentence(err))
	}
	err = s.gw.cfg.Arenas.Put(zone, a)
	if errors.Is(err, zones.ErrNoZone) {
		return s.notice(ctx, fmt.Sprintf("Save %s first with /zone add %s SIZE.", zone, zone))
	}
	if err != nil {
		return err
	}
	s.log.Info("arena changed", "zone", zone, "by", s.player.Name)
	s.gw.zonesChanged()
	return s.notice(ctx, describeArena(a))
}

func (s *session) clearArena(ctx context.Context, zone string) error {
	err := s.gw.cfg.Arenas.Remove(zone)
	if errors.Is(err, arenas.ErrNoArena) {
		return s.notice(ctx, zone+" is not an arena.")
	}
	if err != nil {
		return err
	}
	s.log.Info("arena cleared", "zone", zone, "by", s.player.Name)
	s.gw.zonesChanged()
	return s.notice(ctx, zone+" is no longer an arena.")
}

func (s *session) listArenas(ctx context.Context) error {
	all, err := s.gw.cfg.Arenas.All()
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return s.notice(ctx, "There are no arenas. "+arenaHelp)
	}
	lines := make([]string, 0, len(all))
	for _, a := range all {
		lines = append(lines, describeArena(a))
	}
	return s.notice(ctx, strings.Join(lines, "; "))
}

// describeArena is an arena in a line, e.g. "Field (race): first to a goal, 90 seconds, 1 spawn, 1 goal",
// followed by anything it still needs.
func describeArena(a arenas.Arena) string {
	win := "last one standing"
	switch a.Win {
	case arenas.WinGoal:
		win = "first to a goal"
	case arenas.WinPoints:
		win = fmt.Sprintf("first to %d points", a.Points)
	}
	timer := "no time limit"
	if a.Seconds > 0 {
		timer = fmt.Sprintf("%d seconds", a.Seconds)
	}
	spawns := count(len(a.Spawns), "spawn")
	if teams := a.Teams(); len(teams) > 0 {
		names := make([]string, len(teams))
		for i, t := range teams {
			names[i] = string(t)
		}
		spawns += " (" + strings.Join(names, ", ") + ")"
	}
	name := a.Zone
	if a.Game != arenas.Custom {
		name += " (" + a.Game.String() + ")"
	}
	line := fmt.Sprintf("%s: %s, %s, %s, %s", name, win, timer, spawns, count(len(a.Goals), "goal"))
	if err := a.Ready(); err != nil {
		line += ". " + sentence(err)
	}
	return line
}

// count is "no goals", "1 goal" or "3 goals".
func count(n int, thing string) string {
	switch n {
	case 0:
		return "no " + thing + "s"
	case 1:
		return "1 " + thing
	}
	return fmt.Sprintf("%d %ss", n, thing)
}

// sentence turns an error into a sentence for players.
func sentence(err error) string {
	r := []rune(err.Error())
	r[0] = unicode.ToUpper(r[0])
	return string(r) + "."
}

// arenaMarks is every arena's spawns and goals for the wire, by lowercased zone name.
func (g *Gateway) arenaMarks() map[string][]*pb.ArenaMark {
	all, err := g.cfg.Arenas.All()
	if err != nil {
		g.cfg.Logger.Error("reading arenas", "err", err)
	}
	out := make(map[string][]*pb.ArenaMark, len(all))
	for _, a := range all {
		key := strings.ToLower(a.Zone)
		for _, m := range a.Spawns {
			out[key] = append(out[key], wireMark(m, false))
		}
		for _, m := range a.Goals {
			out[key] = append(out[key], wireMark(m, true))
		}
	}
	return out
}

func wireMark(m arenas.Mark, goal bool) *pb.ArenaMark {
	return &pb.ArenaMark{X: m.X, Y: m.Y, Z: m.Z, Team: string(m.Team), Goal: goal}
}
