package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"scabbage/internal/art"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// artCache keeps custom block pixels and repainted textures so joining needs no queries.
type artCache struct {
	pixels   map[world.Block][]byte
	textures []*pb.Texture
}

// loadArt registers every approved custom block and refreshes the cache.
func (g *Gateway) loadArt() error {
	blocks, err := g.cfg.Art.Blocks()
	if err != nil {
		return err
	}
	textures, err := g.cfg.Art.Textures()
	if err != nil {
		return err
	}
	cache := artCache{pixels: map[world.Block][]byte{}}
	for _, b := range blocks {
		info := world.BlockInfo{ID: b.BlockID, Name: b.Name, Solid: true, Transparent: b.Transparent()}
		if err := world.Register(info); err != nil {
			return err
		}
		cache.pixels[b.BlockID] = b.Pixels
	}
	for _, t := range textures {
		cache.textures = append(cache.textures, &pb.Texture{Name: t.Name, Pixels: t.Pixels})
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.art = cache
	return nil
}

// palette lists every block, with custom blocks' pixels, and the repainted textures.
func (g *Gateway) palette() *pb.BlocksChanged {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := &pb.BlocksChanged{Textures: g.art.textures}
	for _, b := range world.Blocks() {
		out.Blocks = append(out.Blocks, &pb.BlockType{
			Id: uint32(b.ID), Name: b.Name, Solid: b.Solid, Transparent: b.Transparent, Pixels: g.art.pixels[b.ID],
			Shape: pb.Shape(b.Shape), Light: uint32(b.Light), MineMs: uint32(b.MineMs),
			Usable: b.Usable, Item: heldAs(b.ID), Turned: b.Facing,
		})
	}
	return out
}

// heldAs is the block players hold for b, or 0 when it is b itself.
func heldAs(b world.Block) uint32 {
	if item := b.Item(); item != b {
		return uint32(item)
	}
	return 0
}

func (s *session) submitArt(ctx context.Context, m *pb.SubmitArt) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no paintbrush")
	}
	if !s.talk.Take(time.Now()) {
		return s.notice(ctx, "Slow down a little!")
	}
	// art.Kind uses the same numbers as pb.ArtKind.
	p, err := s.gw.cfg.Art.Submit(s.player.ID, s.player.Admin, art.Kind(m.Kind), m.Name, m.Pixels)
	if isArtMistake(err) {
		return s.notice(ctx, capitalize(err.Error())+".")
	}
	if err != nil {
		return err
	}
	s.log.Info("art submitted", "kind", p.Kind, "name", p.Name, "approved", p.Approved)
	if !p.Approved {
		return s.notice(ctx, "Sent! An admin will take a look.")
	}
	if err := s.gw.artApproved(p); err != nil {
		return err
	}
	return s.notice(ctx, decisionText(p, true))
}

func (s *session) decideArt(ctx context.Context, d *pb.DecideArt) error {
	if err := s.requireAdmin(); err != nil {
		return err
	}
	p, err := s.gw.cfg.Art.Decide(int64(d.Id), d.Approve)
	if errors.Is(err, art.ErrNotFound) {
		return s.sendArtPending(ctx) // someone else decided first
	}
	if errors.Is(err, art.ErrFull) {
		return s.notice(ctx, capitalize(err.Error())+".")
	}
	if err != nil {
		return err
	}
	s.log.Info("art decided", "id", d.Id, "approve", d.Approve, "by", s.player.Name)
	if d.Approve {
		if err := s.gw.artApproved(p); err != nil {
			return err
		}
	}
	s.gw.tell(p.PlayerID, decisionText(p, d.Approve))
	return nil
}

// artApproved shows newly approved art to everyone in the world.
func (g *Gateway) artApproved(p art.Piece) error {
	if p.Kind == art.Skin {
		g.setSkin(p.PlayerID, p.Pixels)
		return nil
	}
	if err := g.loadArt(); err != nil {
		return err
	}
	g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_BlocksChanged{BlocksChanged: g.palette()}})
	return nil
}

// setSkin gives every session of a player a new skin and shows it to everyone else.
func (g *Gateway) setSkin(playerID int64, skin []byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for s, a := range g.players {
		if s.player.ID != playerID {
			continue
		}
		s.skin = skin
		msg := &pb.ServerMessage{Body: &pb.ServerMessage_SkinChanged{SkinChanged: &pb.SkinChanged{Entity: a.entity, Skin: skin}}}
		for o := range g.players {
			if o != s {
				o.push(msg)
			}
		}
	}
}

// tell sends a notice to every in-world session of a player.
func (g *Gateway) tell(playerID int64, text string) {
	msg := &pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.Chat{Text: text, Notice: true}}}
	g.mu.Lock()
	defer g.mu.Unlock()
	for s := range g.players {
		if s.player.ID == playerID {
			s.push(msg)
		}
	}
}

// ArtChanged pushes the art waiting for approval to every connected admin.
func (g *Gateway) ArtChanged() {
	msg, err := g.artPendingMessage()
	if err != nil {
		g.cfg.Logger.Error("listing pending art", "err", err)
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for s := range g.admins {
		s.push(msg)
	}
}

func (s *session) sendArtPending(ctx context.Context) error {
	msg, err := s.gw.artPendingMessage()
	if err != nil {
		return err
	}
	return s.write(ctx, msg)
}

func (g *Gateway) artPendingMessage() (*pb.ServerMessage, error) {
	pieces, err := g.cfg.Art.Pending()
	if err != nil {
		return nil, err
	}
	list := &pb.ArtRequests{Requests: make([]*pb.ArtRequest, 0, len(pieces))}
	for _, p := range pieces {
		list.Requests = append(list.Requests, &pb.ArtRequest{
			Id: uint64(p.ID), Kind: pb.ArtKind(p.Kind), Name: p.Name, Author: p.Author, AuthorColor: p.Color,
			Pixels: p.Pixels, CreatedMs: float64(p.Created.UnixMilli()),
		})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_ArtRequests{ArtRequests: list}}, nil
}

// isArtMistake reports errors the player can fix, which become notices rather than disconnects.
func isArtMistake(err error) bool {
	for _, e := range []error{art.ErrBadKind, art.ErrBadSize, art.ErrBadName, art.ErrNameTaken,
		art.ErrTooMany, art.ErrFull, art.ErrNotAdmin} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func decisionText(p art.Piece, approved bool) string {
	switch {
	case p.Kind == art.Skin && p.Pixels == nil:
		return "You're back to your original look."
	case p.Kind == art.Skin && approved:
		return "Your new look is ready!"
	case p.Kind == art.Skin:
		return "An admin said no to your new look this time."
	case p.Kind == art.Texture && p.Pixels == nil:
		return fmt.Sprintf("%s is back to how it was.", p.Name)
	case p.Kind == art.Texture:
		return fmt.Sprintf("Repainted %s.", p.Name)
	case approved:
		return fmt.Sprintf("Your block %q is ready to build with!", p.Name)
	default:
		return fmt.Sprintf("An admin said no to your block %q this time.", p.Name)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
