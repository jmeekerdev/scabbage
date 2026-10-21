// Package maps keeps adventure maps: a zone's blocks and scripts saved so they can be put back
// after a play-through, or carried to another place or server as a .scabmap file.
package maps

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"scabbage/internal/access"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/script"
	"scabbage/internal/world"
)

const (
	// MaxSide is the widest a map may be along x or z, the same as the largest zone.
	MaxSide = 256
	// MaxFileBytes bounds a .scabmap file; maxRawBytes bounds what it unzips to.
	MaxFileBytes = 4 << 20
	maxRawBytes  = 32 << 20
	maxScripts   = 1024
)

// ErrBadFile means data isn't a map this server can read.
var ErrBadFile = errors.New("that isn't a map file this server can read")

// Map is a zone's worth of world with its script blocks; positions are from its corner.
type Map struct {
	Name         string
	Width, Depth int32
	Blocks       []world.Block // in the order of Area(0, 0)
	Scripts      []Script
	Start        *Start // nil when no start is marked
	Mode         access.Mode
}

// Script is a script block's Lua and who saved it.
type Script struct {
	world.Pos
	Source string
	Owner  int64
}

// Start is where players begin and which way they face.
type Start struct {
	X, Y, Z float64
	Yaw     float32
}

// Area is where the map lies with its corner at x0, z0.
func (m Map) Area(x0, z0 int32) world.Area {
	return world.Area{X0: x0, Z0: z0, X1: x0 + m.Width - 1, Z1: z0 + m.Depth - 1}
}

// Encode writes the map as a gzipped MapFile.
func Encode(m Map) ([]byte, error) {
	f := &pb.MapFile{Name: m.Name, Width: m.Width, Depth: m.Depth, Height: world.MaxY - world.MinY,
		Blocks: world.EncodeBlocks(m.Blocks), Palette: palette(m.Blocks), Mode: string(m.Mode)}
	for _, s := range m.Scripts {
		f.Scripts = append(f.Scripts, &pb.MapScript{X: s.X, Y: s.Y, Z: s.Z, Source: s.Source, Owner: s.Owner})
	}
	if m.Start != nil {
		f.Start = &pb.MapStart{X: m.Start.X, Y: m.Start.Y, Z: m.Start.Z, Yaw: m.Start.Yaw}
	}
	raw, err := proto.Marshal(f)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// palette names every block the map uses.
func palette(blocks []world.Block) []*pb.MapBlock {
	seen := map[world.Block]bool{}
	var out []*pb.MapBlock
	for _, b := range blocks {
		if !seen[b] {
			seen[b] = true
			out = append(out, &pb.MapBlock{Id: uint32(b), Name: world.Info(b).Name})
		}
	}
	return out
}

// Decode reads a gzipped MapFile, turning its blocks into this server's by name; blocks the
// server doesn't know become air, and their number is returned.
func Decode(data []byte) (Map, int, error) {
	f, err := unzip(data)
	if err != nil {
		return Map{}, 0, err
	}
	m := Map{Name: f.Name, Width: f.Width, Depth: f.Depth, Mode: access.Mode(f.Mode)}
	if err := m.checkSize(f.Height); err != nil {
		return Map{}, 0, err
	}
	if m.Blocks, err = world.DecodeBlocks(f.Blocks, m.Area(0, 0).Volume()); err != nil {
		return Map{}, 0, fmt.Errorf("%w: %v", ErrBadFile, err)
	}
	unknown := translate(m.Blocks, f.Palette)
	if m.Scripts, err = m.readScripts(f.Scripts); err != nil {
		return Map{}, 0, err
	}
	if s := f.Start; s != nil {
		m.Start = &Start{X: s.X, Y: s.Y, Z: s.Z, Yaw: s.Yaw}
		if !m.holds(s.X, s.Y, s.Z) {
			return Map{}, 0, fmt.Errorf("%w: the start is outside the map", ErrBadFile)
		}
	}
	if !m.Mode.Valid() {
		return Map{}, 0, fmt.Errorf("%w: unknown mode %q", ErrBadFile, f.Mode)
	}
	return m, unknown, nil
}

func unzip(data []byte) (*pb.MapFile, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadFile, err)
	}
	raw, err := io.ReadAll(io.LimitReader(zr, maxRawBytes+1))
	if err != nil || len(raw) > maxRawBytes {
		return nil, fmt.Errorf("%w: too big or damaged", ErrBadFile)
	}
	var f pb.MapFile
	if err := proto.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadFile, err)
	}
	return &f, nil
}

func (m Map) checkSize(height int32) error {
	switch {
	case height != world.MaxY-world.MinY:
		return fmt.Errorf("%w: it was made for worlds %d blocks tall", ErrBadFile, height)
	case m.Width < 1 || m.Depth < 1 || m.Width > MaxSide || m.Depth > MaxSide:
		return fmt.Errorf("%w: it is %d×%d", ErrBadFile, m.Width, m.Depth)
	}
	return nil
}

// translate swaps each block for this server's block of the same name, counting those it lacks.
func translate(blocks []world.Block, names []*pb.MapBlock) int {
	local := map[world.Block]world.Block{}
	for _, n := range names {
		local[world.Block(n.Id)], _ = world.Named(n.Name)
	}
	unknown := 0
	for i, b := range blocks {
		to, ok := local[b]
		if !ok || to == world.Air && b != world.Air {
			unknown++
		}
		blocks[i] = to
	}
	return unknown
}

func (m Map) readScripts(in []*pb.MapScript) ([]Script, error) {
	if len(in) > maxScripts {
		return nil, fmt.Errorf("%w: %d scripts", ErrBadFile, len(in))
	}
	out := make([]Script, 0, len(in))
	for _, s := range in {
		if !m.holds(float64(s.X), float64(s.Y), float64(s.Z)) || len(s.Source) > script.MaxSource {
			return nil, fmt.Errorf("%w: a script is outside the map or too long", ErrBadFile)
		}
		out = append(out, Script{Pos: world.Pos{X: s.X, Y: s.Y, Z: s.Z}, Source: s.Source, Owner: s.Owner})
	}
	return out, nil
}

// holds reports whether a position from the corner lies inside the map.
func (m Map) holds(x, y, z float64) bool {
	return x >= 0 && x < float64(m.Width) && z >= 0 && z < float64(m.Depth) && y >= world.MinY && y < world.MaxY
}
