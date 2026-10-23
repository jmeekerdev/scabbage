package gateway

import pb "scabbage/internal/pb/scabbage/v1"

// setAway records that the player's game paused or came back, and tells everyone else.
func (s *session) setAway(m *pb.SetAway) {
	g := s.gw
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil || a.away == m.Away {
		return
	}
	a.away = m.Away
	msg := &pb.ServerMessage{Body: &pb.ServerMessage_PlayerAway{PlayerAway: &pb.PlayerAway{Entity: a.entity, Away: a.away}}}
	for o := range g.players {
		if o != s {
			o.push(msg)
		}
	}
}

// prey is whether mobs may hunt the player: in survival, and not away.
func (a *avatar) prey() bool { return a.mode == survival && !a.away }

// everyoneAwayLocked is true when every player's game is paused, so mobs can rest.
func (g *Gateway) everyoneAwayLocked() bool {
	for _, a := range g.players {
		if !a.away {
			return false
		}
	}
	return true
}
