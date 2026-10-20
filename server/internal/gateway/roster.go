package gateway

import (
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/inventory"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/vitals"
)

// avatar is how other players see an in-world session.
type avatar struct {
	entity     uint32
	pos        motion.Vec // last accepted feet position
	yaw, pitch float32
	moved      bool        // since the last tick
	own        pb.GameMode // the player's mode, which a zone may override
	mode       pb.GameMode // the mode they play right now
	vitals     *vitals.State
	inv        inventory.Inventory
	invDirty   bool // changed since it was last saved
	digger     edit.Digger
	hitAt      time.Time // when they last hit a mob
}

func (a *avatar) pose() *pb.PlayerPose {
	return &pb.PlayerPose{
		Entity: a.entity,
		Pos:    &pb.Vec3{X: float32(a.pos.X), Y: float32(a.pos.Y), Z: float32(a.pos.Z)},
		Yaw:    a.yaw,
		Pitch:  a.pitch,
	}
}

func (s *session) other(a *avatar) *pb.OtherPlayer {
	return &pb.OtherPlayer{Entity: a.entity, Name: s.player.Name, Color: s.player.Color, Pose: a.pose(), Skin: s.skin}
}

// addPlayer puts a session in the world, introducing it and everyone already there to each other.
func (g *Gateway) addPlayer(s *session) {
	marks := g.arenaMarks()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entity++
	a := &avatar{
		entity: g.entity, pos: s.body.Pos(), own: g.modeOf(s.player), vitals: vitals.New(time.Now()), inv: s.carried,
	}
	arrived := playersJoined(s.other(a))
	present := playersJoined()
	for o, oa := range g.players {
		o.push(arrived)
		present.GetPlayersJoined().Players = append(present.GetPlayersJoined().Players, o.other(oa))
	}
	if len(g.players) > 0 {
		s.push(present)
	}
	a.mode = g.zoneMode(a)
	for _, msg := range []*pb.ServerMessage{g.itemsLyingAroundLocked(), g.mobsAroundLocked(), g.zonesMessage(s, marks, true)} {
		if msg != nil {
			s.push(msg)
		}
	}
	g.players[s] = a
}

func (g *Gateway) movePlayer(s *session, at motion.Vec, yaw, pitch float32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.players[s]; a != nil {
		a.pos, a.yaw, a.pitch, a.moved = at, yaw, pitch, true
		g.refreshModeLocked(s, a, false)
	}
}

func (g *Gateway) removePlayer(s *session) {
	if p, ok := g.dropPlayer(s); ok {
		g.save([]pendingSave{p})
	}
}

// dropPlayer takes a session out of the world, returning its inventory if that needs saving.
func (g *Gateway) dropPlayer(s *session) (pendingSave, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil {
		return pendingSave{}, false
	}
	delete(g.players, s)
	if len(g.players) == 0 {
		g.cfg.Mobs.Clear() // no one left to see them
	}
	left := &pb.ServerMessage{Body: &pb.ServerMessage_PlayerLeft{PlayerLeft: &pb.PlayerLeft{Entity: a.entity}}}
	for o := range g.players {
		o.push(left)
	}
	return a.unsaved(s.player.ID)
}

// Tick runs the world's players for one step; call it from the sim loop.
func (g *Gateway) Tick() {
	now := time.Now()
	g.save(g.step(now))
	if err := g.stepCircuits(now); err != nil {
		g.cfg.Logger.Error("circuits", "err", err)
	}
}

// step heals players, hands out items, runs mobs and sends each the others that moved since the last tick.
func (g *Gateway) step(now time.Time) []pendingSave {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.healLocked(now)
	g.collectLocked(now)
	g.stepMobsLocked(now)
	saves := g.unsavedLocked(now)
	var moved []*avatar
	for _, a := range g.players {
		if a.moved {
			moved = append(moved, a)
			a.moved = false
		}
	}
	if len(moved) > 0 {
		for s, own := range g.players {
			if msg := playersMoved(moved, own); msg != nil {
				s.push(msg)
			}
		}
	}
	return saves
}

// playersMoved lists every moved avatar except the recipient's own, or nil if none are left.
func playersMoved(moved []*avatar, own *avatar) *pb.ServerMessage {
	poses := make([]*pb.PlayerPose, 0, len(moved))
	for _, a := range moved {
		if a != own {
			poses = append(poses, a.pose())
		}
	}
	if len(poses) == 0 {
		return nil
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_PlayersMoved{PlayersMoved: &pb.PlayersMoved{Poses: poses}}}
}

func playersJoined(players ...*pb.OtherPlayer) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_PlayersJoined{PlayersJoined: &pb.PlayersJoined{Players: players}}}
}

// bodies returns every in-world player's position.
func (g *Gateway) bodies() []motion.Vec {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]motion.Vec, 0, len(g.players))
	for _, a := range g.players {
		out = append(out, a.pos)
	}
	return out
}

// broadcast queues msg for every in-world session.
func (g *Gateway) broadcast(msg *pb.ServerMessage) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.broadcastLocked(msg)
}

func (g *Gateway) broadcastLocked(msg *pb.ServerMessage) {
	for s := range g.players {
		s.push(msg)
	}
}
