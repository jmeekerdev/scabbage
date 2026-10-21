package gateway

import (
	"math"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// keepChunks is how many chunks around a player stay loaded: the farthest view plus a margin.
const keepChunks = 14

// Nearby snapshots where players are and reports which chunks are near enough to keep loaded.
func (g *Gateway) Nearby() func(world.ChunkPos) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	at := make([]motion.Vec, 0, len(g.players))
	for _, a := range g.players {
		at = append(at, a.pos)
	}
	return nearAny(at, keepChunks)
}

// nearAny reports whether a chunk's column is within r chunks of one of the positions.
func nearAny(at []motion.Vec, r int32) func(world.ChunkPos) bool {
	cols := make([][2]int32, len(at))
	for i, v := range at {
		cols[i] = [2]int32{chunkIndex(v.X), chunkIndex(v.Z)}
	}
	return func(p world.ChunkPos) bool {
		for _, c := range cols {
			if abs32(p.X-c[0]) <= r && abs32(p.Z-c[1]) <= r {
				return true
			}
		}
		return false
	}
}

func chunkIndex(v float64) int32 { return int32(math.Floor(v / world.Size)) }

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
