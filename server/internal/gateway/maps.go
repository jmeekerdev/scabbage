package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"scabbage/internal/access"
	"scabbage/internal/logic"
	"scabbage/internal/maps"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
	"scabbage/internal/zones"
)

const (
	mapHelp = "Maps: /map save ZONE, /map start ZONE, /map reset ZONE, /map export ZONE, " +
		"/map import NAME, /map sample NAME, /map remove ZONE, /map forget ZONE, /map list"
	maxBatchBlocks = 1024 // more changes than this are sent as chunks to fetch again
	// maxAdminMessageBytes lets admins send .scabmap files.
	maxAdminMessageBytes = maps.MaxFileBytes + 1024
)

// mapCommand runs an admin's /map command.
func (s *session) mapCommand(ctx context.Context, arg string) error {
	f := strings.Fields(arg)
	if len(f) == 0 || f[0] == "list" {
		return s.listMaps(ctx)
	}
	if len(f) != 2 {
		return s.notice(ctx, mapHelp)
	}
	switch f[0] {
	case "save":
		return s.saveMap(ctx, f[1])
	case "start":
		return s.markStart(ctx, f[1])
	case "reset":
		return s.resetMap(ctx, f[1])
	case "export":
		return s.exportMap(ctx, f[1])
	case "import":
		return s.notice(ctx, "Type /map import NAME in the game's chat box to pick a .scabmap file.")
	case "sample":
		return s.placeMap(ctx, f[1], maps.Sample(s.groundBelow()), 0)
	case "forget":
		return s.forgetMap(ctx, f[1])
	case "remove":
		return s.removeMap(ctx, f[1])
	}
	return s.notice(ctx, mapHelp)
}

func zoneArea(z zones.Zone) world.Area { return world.Area{X0: z.X0, Z0: z.Z0, X1: z.X1, Z1: z.Z1} }

// saveMap keeps a zone's blocks and scripts as its map, keeping the start already marked.
func (s *session) saveMap(ctx context.Context, name string) error {
	z, ok := s.gw.cfg.Zones.Named(name)
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", name))
	}
	a := zoneArea(z)
	if a.Width() > maps.MaxSide || a.Depth() > maps.MaxSide {
		return s.notice(ctx, fmt.Sprintf("Maps can be at most %d blocks across.", maps.MaxSide))
	}
	old, _, err := s.gw.cfg.Maps.Get(z.Name)
	if err != nil {
		return err
	}
	m, err := s.gw.captureMap(z)
	if err != nil {
		return err
	}
	if old.Start != nil && m.Width == old.Width && m.Depth == old.Depth {
		m.Start = old.Start
	}
	if err := s.gw.cfg.Maps.Put(z.Name, m); errors.Is(err, zones.ErrNoZone) {
		return s.notice(ctx, fmt.Sprintf("The built-in %s zone can't keep a map; /zone add one of your own.", z.Name))
	} else if err != nil {
		return err
	}
	s.log.Info("map saved", "zone", z.Name, "scripts", len(m.Scripts), "by", s.player.Name)
	return s.notice(ctx, fmt.Sprintf("💾 Saved the map %s (%d×%d, %d script blocks). /map reset %s puts it back like this.",
		z.Name, m.Width, m.Depth, len(m.Scripts), z.Name))
}

// captureMap copies a zone's blocks and the scripts in its script blocks.
func (g *Gateway) captureMap(z zones.Zone) (maps.Map, error) {
	m, err := g.captureArea(zoneArea(z))
	m.Name, m.Mode = z.Name, z.Mode
	return m, err
}

// captureArea copies the blocks in area a and the scripts in its script blocks.
func (g *Gateway) captureArea(a world.Area) (maps.Map, error) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	blocks, err := g.cfg.World.Copy(a)
	if err != nil {
		return maps.Map{}, err
	}
	m := maps.Map{Width: a.Width(), Depth: a.Depth(), Blocks: blocks}
	inside, err := g.scripts.Inside(a.X0, a.Z0, a.X1, a.Z1)
	if err != nil {
		return maps.Map{}, err
	}
	for _, at := range inside {
		saved, ok, err := g.scripts.Source(at)
		if err != nil {
			return maps.Map{}, err
		}
		if ok && blocks[a.Index(at.X, at.Y, at.Z)].Item() == world.ScriptBlock {
			rel := world.Pos{X: at.X - a.X0, Y: at.Y, Z: at.Z - a.Z0}
			m.Scripts = append(m.Scripts, maps.Script{Pos: rel, Source: saved.Source, Owner: saved.Owner})
		}
	}
	return m, nil
}

// markStart makes where the admin stands the place a zone's map starts.
func (s *session) markStart(ctx context.Context, name string) error {
	z, ok := s.gw.cfg.Zones.Named(name)
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", name))
	}
	m, saved, err := s.gw.cfg.Maps.Get(z.Name)
	if err != nil {
		return err
	}
	if !saved {
		return s.notice(ctx, fmt.Sprintf("Save the map first with /map save %s.", z.Name))
	}
	at := s.body.Pos()
	if !z.Contains(int32(math.Floor(at.X)), int32(math.Floor(at.Z))) {
		return s.notice(ctx, fmt.Sprintf("Stand inside %s to mark its start.", z.Name))
	}
	m.Start = &maps.Start{X: at.X - float64(z.X0), Y: at.Y, Z: at.Z - float64(z.Z0), Yaw: s.yaw()}
	if err := s.gw.cfg.Maps.Put(z.Name, m); err != nil {
		return err
	}
	return s.notice(ctx, fmt.Sprintf("🏁 %s will start here.", z.Name))
}

func (s *session) yaw() float32 {
	s.gw.mu.Lock()
	defer s.gw.mu.Unlock()
	if a := s.gw.players[s]; a != nil {
		return a.yaw
	}
	return 0
}

// resetMap puts a zone back the way its map was saved and sends the players inside to its start.
func (s *session) resetMap(ctx context.Context, name string) error {
	z, ok := s.gw.cfg.Zones.Named(name)
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", name))
	}
	m, saved, err := s.gw.cfg.Maps.Get(z.Name)
	if err != nil {
		return err
	}
	if !saved {
		return s.notice(ctx, fmt.Sprintf("%s has no saved map. /map save %s keeps it as it is now.", z.Name, z.Name))
	}
	if a := zoneArea(z); a.Width() != m.Width || a.Depth() != m.Depth {
		return s.notice(ctx, fmt.Sprintf("%s changed size since its map was saved; /map save %s again.", z.Name, z.Name))
	}
	n, err := s.gw.putMap(z, m, 0)
	if err != nil {
		return err
	}
	s.log.Info("map reset", "zone", z.Name, "blocks", n, "by", s.player.Name)
	return s.notice(ctx, fmt.Sprintf("%s is back as it was saved (%d blocks put back).", z.Name, n))
}

// putMap lays m over zone z, then sends the players inside to its start. Scripts act for owner, or
// for whoever saved each when owner is 0. It returns how many blocks changed.
func (g *Gateway) putMap(z zones.Zone, m maps.Map, owner int64) (int, error) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	n, err := g.pasteLocked(zoneArea(z), m, owner)
	if err == nil {
		g.startOver(z, g.startOf(z, m))
	}
	return n, err
}

// pasteLocked lays m's blocks and scripts over area a for everyone and re-powers the circuits it
// touched, returning how many blocks changed; call with editMu held.
func (g *Gateway) pasteLocked(a world.Area, m maps.Map, owner int64) (int, error) {
	changes, err := g.cfg.World.Paste(a, m.Blocks)
	if err != nil {
		return 0, err
	}
	g.broadcastPaste(changes)
	if err := g.putScriptsLocked(a, m.Scripts, owner); err != nil {
		return len(changes), err
	}
	for _, c := range changes {
		if logic.Involves(c.Block) || logic.Involves(c.Was) {
			if err := g.powerLocked(logic.Pos(c.Pos)); err != nil {
				return len(changes), err
			}
		}
	}
	return len(changes), nil
}

// broadcastPaste sends a few changes block by block, and many as the chunks to fetch again.
func (g *Gateway) broadcastPaste(changes []world.Change) {
	if len(changes) == 0 {
		return
	}
	if len(changes) <= maxBatchBlocks {
		batch := &pb.BlockBatch{Blocks: make([]*pb.BlockChanged, 0, len(changes))}
		for _, c := range changes {
			batch.Blocks = append(batch.Blocks, blockChanged(c.X, c.Y, c.Z, c.Block).GetBlockChanged())
		}
		g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_BlockBatch{BlockBatch: batch}})
		return
	}
	g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_ChunksStale{ChunksStale: &pb.ChunksStale{Chunks: staleChunks(changes)}}})
}

// staleChunks lists each chunk holding a change once, in the changes' order.
func staleChunks(changes []world.Change) []*pb.ChunkPos {
	seen := map[world.ChunkPos]bool{}
	var out []*pb.ChunkPos
	for _, c := range changes {
		p, _, _, _ := world.ChunkOf(c.X, c.Y, c.Z)
		if !seen[p] {
			seen[p] = true
			out = append(out, &pb.ChunkPos{X: p.X, Y: p.Y, Z: p.Z})
		}
	}
	return out
}

// putScriptsLocked saves a map's scripts in area a and forgets any others there; call with editMu held.
func (g *Gateway) putScriptsLocked(a world.Area, scripts []maps.Script, owner int64) error {
	keep := map[world.Pos]bool{}
	now := time.Now()
	for _, sc := range scripts {
		at := world.Pos{X: a.X0 + sc.X, Y: sc.Y, Z: a.Z0 + sc.Z}
		keep[at] = true
		by := sc.Owner
		if owner != 0 {
			by = owner
		}
		if err := g.scripts.Save(at, sc.Source, by, now); err != nil {
			g.cfg.Logger.Debug("map script didn't start", "x", at.X, "y", at.Y, "z", at.Z, "err", err)
		}
	}
	inside, err := g.scripts.Inside(a.X0, a.Z0, a.X1, a.Z1)
	if err != nil {
		return err
	}
	for _, at := range inside {
		if !keep[at] {
			if err := g.scripts.Delete(at); err != nil {
				return err
			}
		}
	}
	return nil
}

// startOf is where a map begins in its zone: its marked start, or on top of the zone's middle.
func (g *Gateway) startOf(z zones.Zone, m maps.Map) spot {
	if st := m.Start; st != nil {
		at := motion.Vec{X: float64(z.X0) + st.X, Y: st.Y, Z: float64(z.Z0) + st.Z}
		return spot{at: at, restore: &pb.Restore{Yaw: st.Yaw}}
	}
	x, zz := (z.X0+z.X1)/2, (z.Z0+z.Z1)/2
	top, err := g.cfg.World.Top(x, zz)
	if err != nil {
		top = world.MaxY
	}
	return spot{at: motion.Vec{X: float64(x) + 0.5, Y: float64(top), Z: float64(zz) + 0.5}}
}

// startOver sends everyone inside zone z to its start and forgets checkpoints there.
func (g *Gateway) startOver(z zones.Zone, start spot) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for s, a := range g.players {
		if a.checkpoint.set && z.Contains(a.checkpoint.at.X, a.checkpoint.at.Z) {
			a.checkpoint = checkpoint{}
		}
		if z.Contains(int32(math.Floor(a.pos.X)), int32(math.Floor(a.pos.Z))) {
			s.relocateLocked(start)
			s.push(noticeMessage(fmt.Sprintf("🔄 %s starts over!", z.Name)))
		}
	}
}

// exportMap sends the admin a zone's saved map as a .scabmap file.
func (s *session) exportMap(ctx context.Context, name string) error {
	z, ok := s.gw.cfg.Zones.Named(name)
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", name))
	}
	data, saved, err := s.gw.cfg.Maps.File(z.Name)
	if err != nil {
		return err
	}
	if !saved {
		return s.notice(ctx, fmt.Sprintf("%s has no saved map yet; /map save %s first.", z.Name, z.Name))
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_MapExported{MapExported: &pb.MapExported{Name: z.Name, Data: data}}})
}

// importMap places an admin's .scabmap file as a new zone around them.
func (s *session) importMap(ctx context.Context, m *pb.ImportMap) error {
	if !s.player.Admin || s.console {
		return s.notice(ctx, "Only admins can import maps.")
	}
	if len(m.Data) > maps.MaxFileBytes {
		return s.notice(ctx, fmt.Sprintf("Map files can be at most %d MB.", maps.MaxFileBytes>>20))
	}
	mp, unknown, err := maps.Decode(m.Data)
	if errors.Is(err, maps.ErrBadFile) {
		return s.notice(ctx, "That isn't a map file this server can read.")
	}
	if err != nil {
		return err
	}
	return s.placeMap(ctx, m.Name, mp, unknown)
}

// placeMap lays a map down as a new zone centred on the admin, if that covers no other zone.
func (s *session) placeMap(ctx context.Context, name string, m maps.Map, unknown int) error {
	at := s.body.Pos()
	a := m.Area(int32(math.Floor(at.X))-m.Width/2, int32(math.Floor(at.Z))-m.Depth/2)
	if problem := s.gw.placeProblem(name, a); problem != "" {
		return s.notice(ctx, problem)
	}
	land, err := s.gw.captureArea(a)
	if err != nil {
		return err
	}
	if err := s.gw.cfg.Zones.Put(name, a.X0, a.Z0, a.X1, a.Z1); errors.Is(err, zones.ErrBadName) {
		return s.notice(ctx, "Zone names are 1 to 24 letters.")
	} else if err != nil {
		return err
	}
	if m.Mode != access.ModeDefault {
		if err := s.gw.cfg.Zones.SetMode(name, m.Mode); err != nil {
			return err
		}
	}
	z, _ := s.gw.cfg.Zones.Named(name)
	m.Name = z.Name
	for i := range m.Scripts {
		m.Scripts[i].Owner = s.player.ID
	}
	if err := s.gw.cfg.Maps.Put(z.Name, m); err != nil {
		return err
	}
	if err := s.gw.cfg.Maps.PutUnder(z.Name, land); err != nil {
		return err
	}
	s.gw.zonesChanged()
	if _, err := s.gw.putMap(z, m, s.player.ID); err != nil {
		return err
	}
	s.log.Info("map placed", "zone", z.Name, "width", m.Width, "depth", m.Depth, "by", s.player.Name)
	text := fmt.Sprintf("🗺️ Placed %s (%d×%d) around you. /map reset %s puts it back like this, "+
		"and /map remove %s brings back the land that was here.", z.Name, m.Width, m.Depth, z.Name, z.Name)
	if unknown > 0 {
		text += fmt.Sprintf(" %d blocks this server doesn't have became air.", unknown)
	}
	return s.notice(ctx, text)
}

// placeProblem explains why a new map can't go in area a, or is "" when it can.
func (g *Gateway) placeProblem(name string, a world.Area) string {
	if _, taken := g.cfg.Zones.Named(name); taken {
		return fmt.Sprintf("There's already a zone called %s.", name)
	}
	if !g.cfg.World.InBorder(a) {
		return "That map would poke out of the world. Move further in and try again."
	}
	for _, z := range g.cfg.Zones.All() {
		if z.X0 <= a.X1 && a.X0 <= z.X1 && z.Z0 <= a.Z1 && a.Z0 <= z.Z1 {
			return fmt.Sprintf("That would cover the zone %s. Move somewhere clear and try again.", z.Name)
		}
	}
	return ""
}

// groundBelow is the y of the block under the admin's feet, kept clear of the world's floor and sky.
func (s *session) groundBelow() int32 {
	y := int32(math.Floor(s.body.Pos().Y)) - 1
	return min(max(y, 8*world.Metre), world.MaxY-40*world.Metre)
}

// removeMap puts back the land a placed map covered, then removes its zone, map and scripts.
func (s *session) removeMap(ctx context.Context, name string) error {
	z, ok := s.gw.cfg.Zones.Named(name)
	if !ok {
		return s.notice(ctx, fmt.Sprintf("No zone is called %q.", name))
	}
	land, kept, err := s.gw.cfg.Maps.Under(z.Name)
	if err != nil {
		return err
	}
	if !kept || land.Width != zoneArea(z).Width() || land.Depth != zoneArea(z).Depth() {
		return s.notice(ctx, fmt.Sprintf("/map remove is only for maps placed with /map import or /map sample; "+
			"%s's land from before wasn't kept. /zone remove %s drops the zone and leaves the blocks.", z.Name, z.Name))
	}
	if err := s.gw.liftMap(z, land); err != nil {
		return err
	}
	if err := s.gw.cfg.Zones.Remove(z.Name); err != nil {
		return err
	}
	s.gw.zonesChanged()
	s.log.Info("map removed", "zone", z.Name, "by", s.player.Name)
	return s.notice(ctx, fmt.Sprintf("Removed %s and put back the land that was there.", z.Name))
}

// liftMap lays the kept land back over zone z, keeping its scripts' owners, and stands
// the players inside on top of it.
func (g *Gateway) liftMap(z zones.Zone, land maps.Map) error {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	if _, err := g.pasteLocked(zoneArea(z), land, 0); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for s, a := range g.players {
		x, zz := int32(math.Floor(a.pos.X)), int32(math.Floor(a.pos.Z))
		if !z.Contains(x, zz) {
			continue
		}
		if top, err := g.cfg.World.Top(x, zz); err == nil {
			s.relocateLocked(spot{at: motion.Vec{X: a.pos.X, Y: float64(top), Z: a.pos.Z}})
		}
	}
	return nil
}

func (s *session) forgetMap(ctx context.Context, name string) error {
	err := s.gw.cfg.Maps.Remove(name)
	if errors.Is(err, maps.ErrNoMap) {
		return s.notice(ctx, fmt.Sprintf("%q has no saved map.", name))
	}
	if err != nil {
		return err
	}
	return s.notice(ctx, fmt.Sprintf("Forgot the map of %s; the zone and its blocks stay.", name))
}

func (s *session) listMaps(ctx context.Context) error {
	names, err := s.gw.cfg.Maps.Names()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return s.notice(ctx, "No maps are saved. "+mapHelp)
	}
	return s.notice(ctx, "Maps: "+strings.Join(names, ", "))
}
