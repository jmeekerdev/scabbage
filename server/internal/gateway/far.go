package gateway

import (
	"context"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// maxFarRange is how many chunk columns from the player far land is sketched: three times the
// client's largest view distance, and slack for moves still in flight.
const maxFarRange = 40

// sendFar answers a request with one FarLand in the order asked, minus columns too far away or
// outside the border.
func (s *session) sendFar(ctx context.Context, req *pb.RequestFar) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no world")
	}
	if len(req.Columns) > maxChunksPerRequest {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "too much far land requested")
	}
	land := &pb.FarLand{Tiles: make([]*pb.FarTile, 0, len(req.Columns))}
	for _, p := range req.Columns {
		if !s.nearPlayer(p.X, p.Z, maxFarRange) {
			continue
		}
		if tile, ok := s.gw.cfg.World.FarTile(p.X, p.Z); ok {
			land.Tiles = append(land.Tiles, farTileProto(p, tile))
		}
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_FarLand{FarLand: land}})
}

func farTileProto(p *pb.ColumnPos, t world.FarTile) *pb.FarTile {
	out := &pb.FarTile{Pos: p, Tops: make([]uint32, len(t.Tops)), Blocks: make([]uint32, len(t.Blocks))}
	for i := range t.Tops {
		out.Tops[i], out.Blocks[i] = uint32(t.Tops[i]), uint32(t.Blocks[i])
	}
	return out
}
