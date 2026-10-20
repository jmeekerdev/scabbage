package gateway

import (
	"context"
	"math"
	"strings"
	"time"

	"scabbage/internal/arenas"
	"scabbage/internal/logic"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rounds"
)

// enterPortal joins the player to the round of the arena whose portal they stepped on, if any.
func (g *Gateway) enterPortal(s *session, p logic.Pos) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	arena, portal, ok := g.portalAtLocked(p)
	if a == nil || !ok || a.round != nil {
		return
	}
	key := strings.ToLower(arena.Zone)
	r := g.rounds[key]
	if r == nil {
		r = rounds.New(arena)
		g.rounds[key] = r
	}
	effects := r.Join(a.entity, s.player.Name, portal, time.Now())
	if r.Has(a.entity) {
		a.round = r
	}
	g.playLocked(r, effects)
}

// portalAtLocked finds the arena with a portal at p.
func (g *Gateway) portalAtLocked(p logic.Pos) (arenas.Arena, arenas.Mark, bool) {
	for _, v := range g.arenas {
		for _, m := range v.arena.Portals {
			if m.At(p.X, p.Y, p.Z) {
				return v.arena, m, true
			}
		}
	}
	return arenas.Arena{}, arenas.Mark{}, false
}

// playLocked carries out a round's effects, then shows its players the board and forgets it once empty.
func (g *Gateway) playLocked(r *rounds.Round, effects []rounds.Effect) {
	for _, e := range effects {
		s, _ := g.playerByEntityLocked(e.Player)
		if s == nil {
			continue // they left the world
		}
		if e.Say != "" {
			s.push(noticeMessage(e.Say))
		}
		if e.Move != nil {
			s.relocateLocked(motion.Vec{X: float64(e.Move.X) + 0.5, Y: float64(e.Move.Y), Z: float64(e.Move.Z) + 0.5})
		}
	}
	board := boardMessage(r.Board(time.Now()))
	for s, a := range g.players {
		switch {
		case a.round != r:
		case r.Has(a.entity):
			s.push(board)
		default:
			a.round = nil
			s.push(&pb.ServerMessage{Body: &pb.ServerMessage_Scoreboard{Scoreboard: &pb.Scoreboard{}}})
		}
	}
	if r.Empty() {
		delete(g.rounds, strings.ToLower(r.Zone()))
	}
}

func boardMessage(b rounds.Board) *pb.ServerMessage {
	out := &pb.Scoreboard{Zone: b.Zone, Status: b.Status, Seconds: uint32(max(b.Seconds, 0))}
	for _, s := range b.Scores {
		out.Scores = append(out.Scores, &pb.Score{
			Name: s.Name, Team: string(s.Team), Points: uint32(s.Points), Out: s.Out, Flag: string(s.Flag),
		})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_Scoreboard{Scoreboard: out}}
}

// stepRoundsLocked runs every round's countdown and clock.
func (g *Gateway) stepRoundsLocked(now time.Time) {
	for _, r := range g.rounds {
		if effects := r.Step(now); len(effects) > 0 {
			g.playLocked(r, effects)
		}
	}
}

// followLocked tells a player's round where they now stand.
func (g *Gateway) followLocked(a *avatar) {
	if a.round == nil {
		return
	}
	x, y, z := int32(math.Floor(a.pos.X)), int32(math.Floor(a.pos.Y)), int32(math.Floor(a.pos.Z))
	zone, ok := g.cfg.Zones.At(x, z)
	inside := ok && strings.EqualFold(zone.Name, a.round.Zone())
	if effects := a.round.Stand(a.entity, x, y, z, inside); len(effects) > 0 {
		g.playLocked(a.round, effects)
	}
}

// fallInRound lets the player's round deal with them fainting, reporting whether it did.
func (g *Gateway) fallInRound(s *session) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.players[s]; a != nil && a.round != nil {
		return g.fallLocked(a)
	}
	return false
}

func (g *Gateway) fallLocked(a *avatar) bool {
	r := a.round
	effects, ok := r.Fall(a.entity)
	if ok {
		g.playLocked(r, effects)
	}
	return ok
}

// leaveRoundLocked takes a player out of their round, if they are in one.
func (g *Gateway) leaveRoundLocked(a *avatar) bool {
	r := a.round
	if r == nil {
		return false
	}
	g.playLocked(r, r.Leave(a.entity))
	a.round = nil // in case they already left the world
	return true
}

// leaveRound runs /leave, which anyone may use.
func (s *session) leaveRound(ctx context.Context) error {
	g := s.gw
	g.mu.Lock()
	a := g.players[s]
	left := a != nil && g.leaveRoundLocked(a)
	g.mu.Unlock()
	if !left {
		return s.notice(ctx, "You aren't in a game.")
	}
	return nil
}

// relocateLocked asks the run goroutine to move the player; a newer move replaces one not yet made.
// Every caller holds gw.mu, so after the drain there is room.
func (s *session) relocateLocked(at motion.Vec) {
	select {
	case <-s.relocate:
	default:
	}
	s.relocate <- at
}
