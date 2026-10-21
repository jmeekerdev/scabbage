package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/logic"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rounds"
	"scabbage/internal/script"
	"scabbage/internal/scripts"
	"scabbage/internal/world"
)

const (
	// scriptRange is how far, in blocks along each axis, a script reaches from its block.
	scriptRange = 16
	// scriptHearing is how far away players hear what a script says.
	scriptHearing = 32
	// scriptErrorGap is how often one script block may tell its owner it went wrong.
	scriptErrorGap = 5 * time.Second
	scriptColor    = 0x7fd4ff
)

var (
	errTooFar       = errors.New("that is too far from the script block")
	errNotHere      = errors.New("nobody by that name is near the script block")
	errNoBlock      = errors.New("there is no block called that")
	errNotYours     = errors.New("the script's owner can't build there")
	errScriptBlock  = errors.New("scripts can't change script blocks or bedrock")
	errStandingHere = errors.New("someone is standing there")
	errOutside      = errors.New("that is outside the world")
	errNotCreative  = errors.New("only admins' scripts give items to survival players")
	errNoRound      = errors.New("no round is being played in this script's arena")
	errNotPlaying   = errors.New("that player isn't playing in this round")
)

// roundEvent is a round starting or ending in the arena of the zone named zone.
type roundEvent struct {
	zone string
	rounds.Event
}

// openScript shows the sender the Lua in a script block they can reach.
func (s *session) openScript(ctx context.Context, m *pb.OpenScript) error {
	at, ok := s.scriptInReach(m.GetPos())
	if !ok || !s.edits.Take(time.Now()) {
		return nil
	}
	s.gw.editMu.Lock()
	saved, _, err := s.gw.scripts.Source(at)
	s.gw.editMu.Unlock()
	if err != nil {
		return err
	}
	return s.write(ctx, s.scriptShown(at, saved.Source, saved.Err, false))
}

// saveScript stores new Lua in a script block, if the sender may build there.
func (s *session) saveScript(ctx context.Context, m *pb.SaveScript) error {
	at, ok := s.scriptInReach(m.GetPos())
	if !ok || !s.edits.Take(time.Now()) || !s.gw.cfg.Zones.CanBuild(s.player.ID, s.player.Admin, at.X, at.Z) {
		return nil
	}
	s.gw.editMu.Lock()
	err := s.gw.scripts.Save(at, m.Source, s.player.ID, time.Now())
	s.gw.editMu.Unlock()
	saved := !errors.Is(err, script.ErrTooLong)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if !saved {
		return s.write(ctx, s.scriptShown(at, m.Source, msg, false))
	}
	s.log.Info("script saved", "name", s.player.Name, "x", at.X, "y", at.Y, "z", at.Z, "bytes", len(m.Source))
	return s.write(ctx, s.scriptShown(at, m.Source, msg, true))
}

func (s *session) scriptShown(at world.Pos, src, errText string, saved bool) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_ScriptShown{ScriptShown: &pb.ScriptShown{
		Pos: &pb.BlockPos{X: at.X, Y: at.Y, Z: at.Z}, Source: src, Error: errText, Saved: saved,
		Editable: s.gw.cfg.Zones.CanBuild(s.player.ID, s.player.Admin, at.X, at.Z),
	}}}
}

// scriptInReach is p if it holds a script block the sender can reach.
func (s *session) scriptInReach(p *pb.BlockPos) (world.Pos, bool) {
	at := world.Pos{X: p.GetX(), Y: p.GetY(), Z: p.GetZ()}
	if s.console || !edit.InReach(s.body.Pos(), at.X, at.Y, at.Z) {
		return at, false
	}
	b, err := s.gw.cfg.World.Block(at.X, at.Y, at.Z)
	return at, err == nil && b.Item() == world.ScriptBlock
}

// touchScriptLocked runs on_touch for the player who used the script block at p; call with editMu held.
func (g *Gateway) touchScriptLocked(s *session, p logic.Pos) {
	g.reportScriptsLocked(g.scripts.Touch(world.Pos(p), s.player.Name, time.Now()), s)
}

// powerScriptsLocked runs on_power for each script block in changes; call with editMu held.
func (g *Gateway) powerScriptsLocked(changes []logic.Change) {
	now := time.Now()
	for _, c := range changes {
		if c.Block.Item() == world.ScriptBlock {
			g.reportScriptsLocked(g.scripts.Power(world.Pos(c.Pos), c.Block != world.ScriptBlock, now), nil)
		}
	}
}

// tellScriptsAboutRoundsLocked runs on_round_start and on_round_end in every script block inside
// the arenas whose rounds started or ended; call with editMu held.
func (g *Gateway) tellScriptsAboutRoundsLocked(now time.Time) {
	g.mu.Lock()
	events := g.roundEvents
	g.roundEvents = nil
	g.mu.Unlock()
	for _, ev := range events {
		zone, ok := g.cfg.Zones.Named(ev.zone)
		if !ok {
			continue
		}
		inside, err := g.scripts.Inside(zone.X0, zone.Z0, zone.X1, zone.Z1)
		if err != nil {
			g.cfg.Logger.Error("finding scripts in an arena", "zone", ev.zone, "err", err)
			continue
		}
		for _, at := range inside {
			if ev.Start {
				g.reportScriptsLocked(g.scripts.RoundStart(at, ev.Names, now), nil)
			} else {
				g.reportScriptsLocked(g.scripts.RoundEnd(at, ev.Names, now), nil)
			}
		}
	}
}

// ForgetScripts drops loaded scripts, e.g. when the world unloads; they start afresh when next used.
func (g *Gateway) ForgetScripts() {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	g.scripts.Forget()
}

// reportScriptsLocked tells each failed script's owner, and whoever touched it, what went wrong.
func (g *Gateway) reportScriptsLocked(failures []scripts.Failure, toucher *session) {
	now := time.Now()
	for _, f := range failures {
		if now.Before(g.scriptErrAt[f.Pos]) {
			continue
		}
		g.scriptErrAt[f.Pos] = now.Add(scriptErrorGap)
		msg := noticeMessage(fmt.Sprintf("The script block at %d %d %d went wrong: %v", f.Pos.X, f.Pos.Y, f.Pos.Z, f.Err))
		g.mu.Lock()
		for s := range g.players {
			if s == toucher || s.player.ID == f.Owner {
				s.push(msg)
			}
		}
		g.mu.Unlock()
	}
}

// scriptHost is the world as one script block's Lua sees it, acting for the script's owner.
// Scripts only run with editMu held, so its methods may change blocks directly.
type scriptHost struct {
	g     *Gateway
	at    world.Pos
	owner int64
	admin bool
}

func (g *Gateway) scriptHost(at world.Pos, owner int64) scripts.Host {
	h := scriptHost{g: g, at: at, owner: owner}
	if players, err := g.cfg.Access.Players(); err == nil {
		for _, p := range players {
			h.admin = h.admin || (p.ID == owner && p.Admin)
		}
	}
	return h
}

// Say shows text to the players near the script block.
func (h scriptHost) Say(text string) {
	msg := &pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.Chat{Name: "Script", Color: scriptColor, Text: text}}}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	for s, a := range h.g.players {
		if h.within(a.pos, scriptHearing) {
			s.push(msg)
		}
	}
}

// SetBlock places a built-in block, or "air", where the script's owner could build.
func (h scriptHost) SetBlock(x, y, z int32, name string) error {
	b, ok := world.Builtin(name)
	switch {
	case !ok || !world.Placeable(b) || b == world.Bedrock || b.Item() == world.ScriptBlock:
		return errNoBlock
	case !h.within(motion.Vec{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}, scriptRange):
		return errTooFar
	case !h.g.cfg.Zones.CanBuild(h.owner, h.admin, x, z):
		return errNotYours
	}
	current, err := h.g.cfg.World.Block(x, y, z)
	switch {
	case errors.Is(err, world.ErrOutside):
		return errOutside
	case err != nil:
		return err
	case current == b:
		return nil
	case current == world.Bedrock || current.Item() == world.ScriptBlock:
		return errScriptBlock
	case world.Info(b).Solid && edit.Occupied(x, y, z, h.g.bodies()):
		return errStandingHere
	}
	if err := h.g.cfg.World.SetBlock(x, y, z, b); err != nil {
		return err
	}
	h.g.broadcast(blockChanged(x, y, z, b))
	if logic.Involves(current) || logic.Involves(b) {
		return h.g.powerLocked(logic.Pos{X: x, Y: y, Z: z})
	}
	return nil
}

// GiveItem hands a nearby player blocks they could have anyway: creative players get anything,
// and only admins' scripts give to survival players.
func (h scriptHost) GiveItem(player, item string, count int) error {
	b, ok := world.Builtin(item)
	if !ok || b == world.Air || !world.Placeable(b) || b == world.Bedrock {
		return errNoBlock
	}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	s, a := h.nearLocked(player)
	switch {
	case a == nil:
		return errNotHere
	case a.mode != creative && !h.admin:
		return errNotCreative
	}
	a.inv.Add(b, count)
	s.push(inventoryMessage(&a.inv))
	return nil
}

// Teleport moves a nearby player anywhere inside the world.
func (h scriptHost) Teleport(player string, x, y, z float64) error {
	if _, err := h.g.cfg.World.Block(int32(math.Floor(x)), int32(math.Floor(y)), int32(math.Floor(z))); err != nil {
		return errOutside
	}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	s, a := h.nearLocked(player)
	if a == nil {
		return errNotHere
	}
	s.relocateLocked(spot{at: motion.Vec{X: x, Y: y, Z: z}})
	return nil
}

// AddPoints gives a player points in the round of the arena the script block is in.
func (h scriptHost) AddPoints(player string, n int) error {
	zone, ok := h.g.cfg.Zones.At(h.at.X, h.at.Z)
	if !ok {
		return errNoRound
	}
	h.g.mu.Lock()
	defer h.g.mu.Unlock()
	r := h.g.rounds[strings.ToLower(zone.Name)]
	if r == nil {
		return errNoRound
	}
	effects, playing := r.AddPoints(player, n)
	if !playing {
		return errNotPlaying
	}
	h.g.playLocked(r, effects)
	return nil
}

// nearLocked finds the in-world player called name within reach of the script; call with g.mu held.
func (h scriptHost) nearLocked(name string) (*session, *avatar) {
	for s, a := range h.g.players {
		if strings.EqualFold(s.player.Name, name) && h.within(a.pos, scriptRange) {
			return s, a
		}
	}
	return nil, nil
}

// within reports whether p is no more than r blocks from the script block along every axis.
func (h scriptHost) within(p motion.Vec, r float64) bool {
	c := motion.Vec{X: float64(h.at.X) + 0.5, Y: float64(h.at.Y), Z: float64(h.at.Z) + 0.5}
	return math.Abs(p.X-c.X) <= r && math.Abs(p.Y-c.Y) <= r && math.Abs(p.Z-c.Z) <= r
}
