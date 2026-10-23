package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"scabbage/internal/agent"
	"scabbage/internal/chat"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rate"
	"scabbage/internal/world"
)

// facings name the way a yaw looks, a quarter turn apart: yaw 0 looks toward -z, turning left.
var facings = [4]string{"north", "west", "south", "east"}

// decodeArgs reads a tool's arguments, refusing fields it doesn't know.
func decodeArgs(args json.RawMessage, v any) error {
	d := json.NewDecoder(bytes.NewReader(args))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return agent.Refusal("bad arguments: " + err.Error())
	}
	return nil
}

// agentBlock describes a block an agent may place, with what sets it apart from a plain cube.
type agentBlock struct {
	Name       string `json:"name"`
	Solid      bool   `json:"solid"`
	SeeThrough bool   `json:"see_through,omitempty"`
	Glows      bool   `json:"glows,omitempty"`
	Shape      string `json:"shape,omitempty"`  // slab, stairs_east, torch, flat, …; a cube when empty
	Usable     bool   `json:"usable,omitempty"` // switched by hand, like a lever
}

// placeableBlocks lists the blocks fill and place take, stairs turned each way included.
func placeableBlocks() map[string]any {
	out := []agentBlock{}
	for _, b := range world.Blocks() {
		if b.ID == world.Air || b.ID == world.Bedrock || !world.Placeable(b.ID) {
			continue
		}
		shape := ""
		if b.Shape != 0 {
			shape = strings.ToLower(strings.TrimPrefix(pb.Shape(b.Shape).String(), "SHAPE_"))
		}
		out = append(out, agentBlock{Name: b.Name, Solid: b.Solid, SeeThrough: b.Transparent && b.Solid,
			Glows: b.Light > 0, Shape: shape, Usable: b.Usable})
	}
	return map[string]any{"blocks": out, "air": `"air" isn't listed but works: fill with it clears, place with it breaks`}
}

type agentPlayer struct {
	Name    string  `json:"name"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
	Mode    string  `json:"mode"`
	Flying  bool    `json:"flying"`
	Facing  string  `json:"facing"`
	Zone    string  `json:"zone,omitempty"`
	BlockAt [3]int  `json:"block"` // the block the feet are in, as the 📍 coordinates show it
}

func (g *Gateway) describeLocked(s *session, a *avatar) agentPlayer {
	quarter := int(math.Round(float64(a.yaw) / (math.Pi / 2)))
	p := agentPlayer{
		Name: s.player.Name, X: tenths(a.pos.X), Y: tenths(a.pos.Y), Z: tenths(a.pos.Z),
		Mode: modeName(a.mode), Flying: a.flying, Facing: facings[(quarter%4+4)%4],
		BlockAt: [3]int{int(math.Floor(a.pos.X)), int(math.Floor(a.pos.Y)), int(math.Floor(a.pos.Z))},
	}
	if z, ok := g.cfg.Zones.At(int32(p.BlockAt[0]), int32(p.BlockAt[2])); ok {
		p.Zone = z.Name
	}
	return p
}

func tenths(v float64) float64 { return math.Round(v*10) / 10 }

func (g *Gateway) whereAmI(s *session) (any, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.players[s]; a != nil {
		return g.describeLocked(s, a), nil
	}
	return nil, agent.Refusal("you just left the world")
}

func (g *Gateway) onlinePlayers() any {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := []agentPlayer{}
	for s, a := range g.players {
		out = append(out, g.describeLocked(s, a))
	}
	return map[string][]agentPlayer{"players": out}
}

type cell struct{ X, Y, Z *int32 }

func (c *cell) valid() bool { return c != nil && c.X != nil && c.Y != nil && c.Z != nil }

// look counts a box's blocks and maps its surface: the top non-air block of each column.
func (g *Gateway) look(args json.RawMessage) (any, error) {
	var a struct{ From, To *cell }
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if !a.From.valid() || !a.To.valid() {
		return nil, agent.Refusal("look needs from and to, each with x, y and z")
	}
	lo := [3]int32{min(*a.From.X, *a.To.X), min(*a.From.Y, *a.To.Y), min(*a.From.Z, *a.To.Z)}
	hi := [3]int32{max(*a.From.X, *a.To.X), max(*a.From.Y, *a.To.Y), max(*a.From.Z, *a.To.Z)}
	if n := (int64(hi[0]-lo[0]) + 1) * (int64(hi[1]-lo[1]) + 1) * (int64(hi[2]-lo[2]) + 1); n > agent.MaxBoxBlocks {
		return nil, agent.Refusal(fmt.Sprintf("that box has %d blocks; look takes at most 32,768", n))
	}
	counts := map[string]int{}
	heights, tops := [][]*int32{}, [][]*string{}
	for z := lo[2]; z <= hi[2]; z++ {
		hrow, trow := []*int32{}, []*string{}
		for x := lo[0]; x <= hi[0]; x++ {
			var top *int32
			var name *string
			for y := hi[1]; y >= lo[1]; y-- {
				b, err := g.cfg.World.Block(x, y, z)
				if err != nil && !errors.Is(err, world.ErrOutside) {
					return nil, err
				}
				counts[b.String()]++
				if top == nil && b != world.Air {
					top, name = &y, ptr(b.String())
				}
			}
			hrow, trow = append(hrow, top), append(trow, name)
		}
		heights, tops = append(heights, hrow), append(tops, trow)
	}
	return map[string]any{"from": lo, "to": hi, "counts": counts, "heights": heights, "tops": tops}, nil
}

func ptr[T any](v T) *T { return &v }

// agentSay chats as the player, marked so everyone can tell an agent said it.
func (g *Gateway) agentSay(s *session, args json.RawMessage) (any, error) {
	var a struct{ Text string }
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	text, ok := chat.Clean(a.Text)
	if !ok {
		return nil, agent.Refusal("say needs some text")
	}
	if !s.agent.mayTalk(time.Now()) {
		return nil, agent.Refusal("slow down: chat is limited to one line every two seconds")
	}
	s.log.Info("agent chat", "name", s.player.Name, "text", text)
	g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.Chat{
		Name: s.player.Name, Color: s.player.Color, Text: "🤖 " + text,
	}}})
	return map[string]string{"said": text}, nil
}

// mayTalk spends from the agent's own chat allowance, as fast as a player may chat.
func (l *agentLink) mayTalk(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.talk == nil {
		l.talk = rate.New(now, chat.PerSecond, chat.Burst)
	}
	return l.talk.Take(now)
}
