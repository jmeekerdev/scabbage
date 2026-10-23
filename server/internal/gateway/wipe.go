package gateway

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/inventory"
	"scabbage/internal/logic"
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

const (
	worldHelp = "World: /world reset starts this world over from its seed, /world new [SEED] makes a new one. " +
		"Another admin approves with /world approve CODE, or any admin stops it with /world deny."
	wipeWait  = 10 * time.Minute // how long a request waits for approval
	kickDelay = time.Second      // lets everyone read the news before they reconnect to the new world
	codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ"
)

var errRoundOn = errors.New("a game is being played; wait for it to end")

// wipeRequest is an admin asking to start the world over, waiting for another admin's approval.
type wipeRequest struct {
	fresh   bool // a new world, rather than this one's seed again
	seed    int64
	by      int64
	byName  string
	code    string
	expires time.Time
}

func (r wipeRequest) what() string {
	if r.fresh {
		return fmt.Sprintf("make a new world (seed %d)", r.seed)
	}
	return "start the world over from its seed"
}

// worldCommand runs an admin's /world command.
func (s *session) worldCommand(ctx context.Context, arg string) error {
	f := strings.Fields(strings.ToLower(arg))
	switch {
	case len(f) == 1 && f[0] == "reset":
		return s.askWipe(ctx, false, s.gw.cfg.World.Meta().Seed)
	case len(f) == 1 && f[0] == "new":
		return s.askWipe(ctx, true, world.RandomSeed())
	case len(f) == 2 && f[0] == "new":
		seed, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil || seed == 0 {
			return s.notice(ctx, "SEED is a whole number other than 0.")
		}
		return s.askWipe(ctx, true, seed)
	case len(f) == 2 && f[0] == "approve":
		return s.approveWipe(ctx, f[1])
	case len(f) == 1 && f[0] == "deny":
		return s.denyWipe(ctx)
	}
	return s.notice(ctx, worldHelp)
}

// askWipe records a request and tells every admin how to approve it.
func (s *session) askWipe(ctx context.Context, fresh bool, seed int64) error {
	alone, err := s.gw.onlyAdmin(s.player.ID)
	if err != nil {
		return err
	}
	g := s.gw
	g.mu.Lock()
	if r := g.wipe; r != nil && time.Now().Before(r.expires) {
		g.mu.Unlock()
		return s.notice(ctx, fmt.Sprintf("%s already asked to %s; /world deny it first.", r.byName, r.what()))
	}
	r := &wipeRequest{fresh: fresh, seed: seed, by: s.player.ID, byName: s.player.Name,
		code: g.codeLocked(), expires: time.Now().Add(wipeWait)}
	g.wipe = r
	text := fmt.Sprintf("⚠️ %s asks to %s. Every block built, zone, map, script and piece of art goes; "+
		"inventories and quests stay, and the database is backed up first. ", r.byName, r.what())
	if alone {
		text += fmt.Sprintf("You're the only admin, so approve it yourself with /world approve %s within 10 minutes, or /world deny.", r.code)
	} else {
		text += fmt.Sprintf("Another admin approves with /world approve %s within 10 minutes, or /world deny.", r.code)
	}
	for a := range g.admins {
		a.push(noticeMessage(text))
	}
	g.mu.Unlock()
	s.log.Warn("world wipe asked", "fresh", fresh, "seed", seed, "by", s.player.Name)
	return nil
}

// codeLocked makes a short code to type when approving; call with mu held.
func (g *Gateway) codeLocked() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = codeChars[g.rng.IntN(len(codeChars))]
	}
	return string(b)
}

// onlyAdmin reports whether player is the server's one admin.
func (g *Gateway) onlyAdmin(player int64) (bool, error) {
	players, err := g.cfg.Access.Players()
	if err != nil {
		return false, err
	}
	for _, p := range players {
		if p.Admin && p.ID != player {
			return false, nil
		}
	}
	return true, nil
}

// approveWipe starts the world over if code matches the waiting request and another admin made it.
func (s *session) approveWipe(ctx context.Context, code string) error {
	alone, err := s.gw.onlyAdmin(s.player.ID)
	if err != nil {
		return err
	}
	g := s.gw
	g.mu.Lock()
	r := g.wipe
	switch {
	case r == nil || time.Now().After(r.expires):
		g.wipe = nil
		g.mu.Unlock()
		return s.notice(ctx, "No one is asking to wipe the world. "+worldHelp)
	case !strings.EqualFold(code, r.code):
		g.mu.Unlock()
		return s.notice(ctx, "That code doesn't match. Check the code in the request.")
	case r.by == s.player.ID && !alone:
		g.mu.Unlock()
		return s.notice(ctx, "Another admin has to approve this.")
	}
	g.wipe = nil
	g.mu.Unlock()
	s.log.Warn("world wipe approved", "fresh", r.fresh, "seed", r.seed, "asked", r.byName, "approved", s.player.Name)
	if err := g.wipeWorld(*r); err != nil {
		s.log.Error("world wipe failed", "err", err)
		return s.notice(ctx, "The world wasn't wiped: "+err.Error())
	}
	return nil
}

func (s *session) denyWipe(ctx context.Context) error {
	g := s.gw
	g.mu.Lock()
	r := g.wipe
	g.wipe = nil
	if r != nil {
		for a := range g.admins {
			a.push(noticeMessage(fmt.Sprintf("%s stopped the request to %s.", s.player.Name, r.what())))
		}
	}
	g.mu.Unlock()
	if r == nil {
		return s.notice(ctx, "No one is asking to wipe the world.")
	}
	return nil
}

// wipeWorld backs up the database, then starts the world over and sends everyone into it afresh.
func (g *Gateway) wipeWorld(r wipeRequest) error {
	if g.roundOn() {
		return errRoundOn
	}
	if err := g.cfg.World.Flush(); err != nil {
		return err
	}
	if err := g.cfg.Backup(); err != nil {
		return fmt.Errorf("the backup failed, so nothing changed: %w", err)
	}
	g.editMu.Lock()
	defer g.editMu.Unlock()
	if err := g.wipeStores(r.seed); err != nil {
		return err
	}
	x, y, z := g.cfg.World.Spawn()
	g.restartPlayers(motion.Vec{X: x, Y: y, Z: z}, r)
	return nil
}

func (g *Gateway) roundOn() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.rounds) > 0
}

// wipeStores clears everything kept about the world, keeping players, inventories and journals.
func (g *Gateway) wipeStores(seed int64) error {
	steps := []func() error{
		func() error { return g.cfg.World.Wipe(seed) },
		g.scripts.Wipe,
		g.cfg.Zones.Wipe,
		g.cfg.Art.Wipe,
		g.forgetPlaces,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	world.ForgetCustomBlocks()
	return g.loadArt()
}

// forgetPlaces starts every saved player at the spawn, without checkpoints or custom blocks.
func (g *Gateway) forgetPlaces() error {
	players, err := g.cfg.Access.Players()
	if err != nil {
		return err
	}
	for _, p := range players {
		v, err := g.cfg.Access.Saved(p.ID)
		if err != nil {
			return err
		}
		inv, _ := inventory.Decode(v.Inventory)
		inv.ForgetCustom()
		v.Placed, v.Checkpoint, v.Inventory = false, nil, inv.Encode()
		if err := g.cfg.Access.Store(p.ID, v); err != nil {
			return err
		}
	}
	return nil
}

// restartPlayers clears what the gateway holds about the old world, then disconnects everyone so
// they rejoin the new one at its spawn.
func (g *Gateway) restartPlayers(spawn motion.Vec, r wipeRequest) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cfg.Mobs.Clear()
	g.drops = drops.Field{}
	g.circuits = newCircuits()
	g.scriptErrAt = map[world.Pos]time.Time{}
	g.arenas = map[string]arenaView{}
	news := noticeMessage(fmt.Sprintf("🌍 The world is starting over! %s asked to %s. See you there in a moment.", r.byName, r.what()))
	all := map[*session]bool{} // admins in the world are in both sets
	for s, a := range g.players {
		a.pos, a.checkpoint = spawn, checkpoint{}
		a.inv.ForgetCustom()
		all[s] = true
	}
	for s := range g.admins {
		all[s] = true
	}
	for s := range all {
		s.push(news)
	}
	time.AfterFunc(kickDelay, func() {
		for s := range all {
			s.cancel()
		}
	})
}

func newCircuits() circuits {
	return circuits{letGo: map[logic.Pos]time.Time{}, held: map[logic.Pos]bool{}}
}
