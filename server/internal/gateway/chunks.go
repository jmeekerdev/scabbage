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
func worldInfo(w *world.World) *pb.WorldInfo {
	x, y, z := w.Spawn()
	info := &pb.WorldInfo{
		Radius: w.Meta().Radius,
		Height: world.MaxY,
		Spawn:  &pb.Vec3{X: float32(x), Y: float32(y), Z: float32(z)},
	}
	for _, b := range world.Blocks() {
		info.Blocks = append(info.Blocks, &pb.BlockType{
			Id: uint32(b.ID), Name: b.Name, Solid: b.Solid, Transparent: b.Transparent,
		})
	}
	return info
}

// nearPlayer reports whether chunk p's column is within maxChunkRange of the player's.
func (s *session) nearPlayer(p *pb.ChunkPos) bool {
	at := s.body.Pos()
	dx := int64(p.X) - int64(math.Floor(at.X/world.Size))
	dz := int64(p.Z) - int64(math.Floor(at.Z/world.Size))
	return dx*dx+dz*dz <= maxChunkRange*maxChunkRange
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
		if !s.nearPlayer(p) {
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
