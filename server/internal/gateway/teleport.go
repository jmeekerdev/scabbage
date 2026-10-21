package gateway

import (
	"context"
	"math"
	"strconv"
	"strings"

	"scabbage/internal/motion"
)

// tpTarget is where /tp goes: a player by name, or a block column's middle.
type tpTarget struct {
	name    string
	x, y, z float64
	ground  bool // y wasn't given, so land on the column's top
}

// parseTp reads "NAME", "X Z" or "X Y Z".
func parseTp(arg string) (tpTarget, bool) {
	words := strings.Fields(arg)
	if len(words) == 1 {
		_, err := strconv.ParseFloat(words[0], 64)
		return tpTarget{name: words[0]}, err != nil
	}
	if len(words) < 2 || len(words) > 3 {
		return tpTarget{}, false
	}
	nums := make([]float64, len(words))
	for i, w := range words {
		n, err := strconv.ParseFloat(w, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return tpTarget{}, false
		}
		nums[i] = n
	}
	t := tpTarget{x: math.Floor(nums[0]) + 0.5, z: math.Floor(nums[len(nums)-1]) + 0.5, ground: len(nums) == 2}
	if !t.ground {
		t.y = nums[1]
	}
	return t, true
}

// tpCommand moves the admin to a place or beside another player.
func (s *session) tpCommand(ctx context.Context, arg string) error {
	t, ok := parseTp(arg)
	if !ok {
		return s.notice(ctx, "Try /tp X Z, /tp X Y Z or /tp NAME.")
	}
	if t.name != "" {
		return s.tpToPlayer(ctx, t.name)
	}
	x, z := int32(math.Floor(t.x)), int32(math.Floor(t.z))
	if t.ground {
		top, err := s.gw.cfg.World.Top(x, z)
		if err != nil {
			return s.notice(ctx, "That's outside the world.")
		}
		t.y = float64(top)
	} else if _, err := s.gw.cfg.World.Block(x, int32(math.Floor(t.y)), z); err != nil {
		return s.notice(ctx, "That's outside the world.")
	}
	s.moveTo(motion.Vec{X: t.x, Y: t.y, Z: t.z})
	return nil
}

func (s *session) tpToPlayer(ctx context.Context, name string) error {
	g := s.gw
	g.mu.Lock()
	found := false
	for o, a := range g.players {
		if !found && strings.EqualFold(o.player.Name, name) {
			s.relocateLocked(spot{at: a.pos})
			found = true
		}
	}
	g.mu.Unlock()
	if !found {
		return s.notice(ctx, "No one called "+strconv.Quote(name)+" is here.")
	}
	return nil
}

func (s *session) moveTo(at motion.Vec) {
	s.gw.mu.Lock()
	defer s.gw.mu.Unlock()
	s.relocateLocked(spot{at: at})
}
