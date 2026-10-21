package gateway

import (
	"context"
	"math"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// maxChunksPerRequest bounds the work and reply size of one request.
const maxChunksPerRequest = 64

// maxChunkRange is how many chunk columns from the player chunks are served: the client's
// largest view distance (12), its extra meshing ring, and slack for moves still in flight.
const maxChunkRange = 15

// worldInfo describes the world so the client can stream and draw it.
func (g *Gateway) worldInfo() *pb.WorldInfo {
	w := g.cfg.World
	x, y, z := w.Spawn()
	palette := g.palette()
	return &pb.WorldInfo{
		Radius:   w.Meta().Radius,
		Height:   world.MaxY,
		Spawn:    &pb.Vec3{X: float32(x), Y: float32(y), Z: float32(z)},
		Blocks:   palette.Blocks,
		Textures: palette.Textures,
		Recipes:  recipesProto(g.cfg.Recipes),
	}
}

// nearPlayer reports whether chunk column x,z is within reach chunks of the player's.
func (s *session) nearPlayer(x, z, reach int32) bool {
	at := s.body.Pos()
	dx := int64(x) - int64(math.Floor(at.X/world.Size))
	dz := int64(z) - int64(math.Floor(at.Z/world.Size))
	return dx*dx+dz*dz <= int64(reach)*int64(reach)
}

// sendChunks answers a request with one batch in the order asked, minus far-away chunks.
func (s *session) sendChunks(ctx context.Context, req *pb.RequestChunks) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no world")
	}
	if len(req.Chunks) > maxChunksPerRequest {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "too many chunks requested")
	}
	batch := &pb.ChunkBatch{Chunks: make([]*pb.ChunkData, 0, len(req.Chunks))}
	for _, p := range req.Chunks {
		if !s.nearPlayer(p.X, p.Z, maxChunkRange) {
			continue
		}
		data, err := s.gw.cfg.World.EncodedChunk(world.ChunkPos{X: p.X, Y: p.Y, Z: p.Z})
		if err != nil {
			return err
		}
		batch.Chunks = append(batch.Chunks, &pb.ChunkData{Pos: p, Data: data})
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_ChunkBatch{ChunkBatch: batch}})
}
