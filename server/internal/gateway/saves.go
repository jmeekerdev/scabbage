package gateway

import (
	"errors"
	"time"

	"scabbage/internal/access"
	"scabbage/internal/inventory"
	"scabbage/internal/journal"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/vitals"
)

// saveEvery is how often changed players are saved while they stay online.
const saveEvery = 30 * time.Second

// snapshot is what a save keeps of a player; it is comparable, so unchanged players aren't rewritten.
type snapshot struct {
	pos        motion.Vec
	yaw, pitch float32
	flying     bool
	health     int
	inv        inventory.Inventory
}

// spot is where to move a player, and how they faced when it restores a save.
type spot struct {
	at      motion.Vec
	restore *pb.Restore
}

func (a *avatar) snapshot() snapshot {
	return snapshot{a.pos, a.yaw, a.pitch, a.flying, a.vitals.Health, a.inv}
}

// load puts the avatar in the state of a save it now matches.
func (a *avatar) load(v snapshot) {
	a.pos, a.yaw, a.pitch, a.flying, a.inv = v.pos, v.yaw, v.pitch, v.flying, v.inv
	a.vitals.Health = v.health
	a.vitals.Settle()
	a.saved = v
}

func (v snapshot) record() access.Save {
	return access.Save{
		Placed: true, X: v.pos.X, Y: v.pos.Y, Z: v.pos.Z, Yaw: v.yaw, Pitch: v.pitch,
		Flying: v.flying, Health: v.health, Inventory: v.inv.Encode(),
	}
}

func (v snapshot) restore() *pb.Restore {
	return &pb.Restore{Yaw: v.yaw, Pitch: v.pitch, Flying: v.flying}
}

// loadSave reads where the player left off; a save that no longer fits starts them at the spawn,
// and a damaged inventory starts them empty-handed.
func (s *session) loadSave() error {
	v, err := s.gw.cfg.Access.Saved(s.player.ID)
	if err != nil {
		return err
	}
	x, y, z := s.gw.cfg.World.Spawn()
	s.saved = snapshot{pos: motion.Vec{X: x, Y: y, Z: z}, health: vitals.Max}
	if at := (motion.Vec{X: v.X, Y: v.Y, Z: v.Z}); v.Placed && !motion.Blocked(at, s.gw.cfg.World.Boxes) {
		s.saved.pos, s.saved.yaw, s.saved.pitch, s.saved.flying = at, v.Yaw, v.Pitch, v.Flying
		s.resumed = true
	}
	if v.Health > 0 && v.Health <= vitals.Max {
		s.saved.health = v.Health
	}
	if s.journal, err = journal.Load(s.gw.cfg.ScriptDB, s.player.ID); err != nil {
		return err
	}
	s.saved.inv, err = inventory.Decode(v.Inventory)
	if errors.Is(err, inventory.ErrCorrupt) {
		s.log.Warn("inventory unreadable, starting empty", "player", s.player.Name)
		return nil
	}
	return err
}

// start is where the welcome puts the player; nil leaves them at the world spawn.
func (s *session) start() *pb.Teleport {
	if !s.resumed {
		return nil
	}
	return &pb.Teleport{Pos: vec3(s.saved.pos), Restore: s.saved.restore()}
}

// unstash puts the avatar back as it was before its game, reporting whether there was one.
func (a *avatar) unstash() bool {
	if a.stash == nil {
		return false
	}
	saved := a.saved
	a.load(*a.stash)
	a.saved, a.stash = saved, nil
	return true
}

// restoreLocked returns a player who left their game to how they were before it.
func (s *session) restoreLocked(a *avatar) {
	if !a.unstash() {
		return
	}
	s.push(inventoryMessage(&a.inv))
	s.push(vitalsMessage(a, pb.Harm_HARM_UNSPECIFIED))
	s.relocateLocked(spot{at: a.pos, restore: a.snapshot().restore()})
}

// pendingSave is what changed about a player: their save, their journal or both.
type pendingSave struct {
	player  int64
	data    *access.Save
	journal *journal.Saved
}

// unsavedLocked collects changed players when it's time to save them; call with g.mu held.
func (g *Gateway) unsavedLocked(now time.Time) []pendingSave {
	if now.Sub(g.savedAt) < saveEvery {
		return nil
	}
	g.savedAt = now
	var out []pendingSave
	for s, a := range g.players {
		if p, ok := a.unsaved(s.player.ID); ok {
			out = append(out, p)
		}
	}
	return out
}

// unsaved is what changed of the avatar's save and journal; during a game the save stays the one from before it.
func (a *avatar) unsaved(player int64) (pendingSave, bool) {
	p := pendingSave{player: player}
	now := a.snapshot()
	if a.stash != nil {
		now = *a.stash
	}
	if now != a.saved {
		a.saved = now
		rec := now.record()
		p.data = &rec
	}
	if j, ok := a.journal.Unsaved(); ok {
		p.journal = &j
	}
	return p, p.data != nil || p.journal != nil
}

// save writes players outside g.mu, since the database can be slow.
func (g *Gateway) save(saves []pendingSave) {
	for _, p := range saves {
		if p.data != nil {
			if err := g.cfg.Access.Store(p.player, *p.data); err != nil {
				g.cfg.Logger.Error("saving player", "player", p.player, "err", err)
			}
		}
		if p.journal != nil {
			if err := journal.Store(g.cfg.ScriptDB, p.player, *p.journal); err != nil {
				g.cfg.Logger.Error("saving journal", "player", p.player, "err", err)
			}
		}
	}
}
