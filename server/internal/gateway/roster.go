package gateway

import (
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
)

// avatar is how other players see an in-world session.
type avatar struct {
	entity     uint32
	pos        motion.Vec // last accepted feet position
	yaw, pitch float32
	moved      bool // since the last tick
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
	return &pb.OtherPlayer{Entity: a.entity, Name: s.player.Name, Color: s.player.Color, Pose: a.pose()}
}

// addPlayer puts a session in the world, introducing it and everyone already there to each other.
func (g *Gateway) addPlayer(s *session) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entity++
	a := &avatar{entity: g.entity, pos: s.body.Pos()}
	arrived := playersJoined(s.other(a))
	present := playersJoined()
	for o, oa := range g.players {
		o.push(arrived)
		present.GetPlayersJoined().Players = append(present.GetPlayersJoined().Players, o.other(oa))
	}
	if len(g.players) > 0 {
		s.push(present)
	}
	g.players[s] = a
}

func (g *Gateway) movePlayer(s *session, at motion.Vec, yaw, pitch float32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.players[s]; a != nil {
		a.pos, a.yaw, a.pitch, a.moved = at, yaw, pitch, true
	}
}

func (g *Gateway) removePlayer(s *session) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil {
		return
	}
	delete(g.players, s)
	left := &pb.ServerMessage{Body: &pb.ServerMessage_PlayerLeft{PlayerLeft: &pb.PlayerLeft{Entity: a.entity}}}
	for o := range g.players {
		o.push(left)
	}
}

// Tick sends each player the others that moved since the last tick; call it from the sim loop.
func (g *Gateway) Tick() {
	g.mu.Lock()
	defer g.mu.Unlock()
	var moved []*avatar
	for _, a := range g.players {
		if a.moved {
			moved = append(moved, a)
			a.moved = false
		}
	}
	if len(moved) == 0 {
		return
	}
	for s, own := range g.players {
		if msg := playersMoved(moved, own); msg != nil {
			s.push(msg)
		}
	}
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
	for s := range g.players {
		s.push(msg)
	}
}
