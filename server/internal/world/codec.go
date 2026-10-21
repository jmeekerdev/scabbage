package world

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// encodingVersion leads every encoded chunk so the format can evolve.
const encodingVersion = 2 // 2: 32-block chunks

var errCorrupt = errors.New("corrupt chunk data")

// Encode serializes a chunk as palette + run-length runs (used for storage and the wire).
//
// Layout: version, uvarint palette size, palette IDs, then (uvarint run length, uvarint
// palette index) pairs in storage order. A one-entry palette has no runs.
func Encode(c *Chunk) []byte {
	if c.blocks == nil {
		return EncodeBlocks([]Block{c.fill})
	}
	return EncodeBlocks(c.blocks[:])
}

// EncodeBlocks serializes any list of blocks in Encode's layout.
func EncodeBlocks(blocks []Block) []byte {
	type run struct {
		length int
		block  Block
	}
	var runs []run
	var palette []Block
	for i, b := range blocks {
		if i > 0 && b == blocks[i-1] {
			runs[len(runs)-1].length++
			continue
		}
		runs = append(runs, run{1, b})
		if paletteIndex(palette, b) < 0 {
			palette = append(palette, b)
		}
	}
	out := []byte{encodingVersion}
	out = binary.AppendUvarint(out, uint64(len(palette)))
	for _, b := range palette {
		out = binary.AppendUvarint(out, uint64(b))
	}
	if len(palette) == 1 {
		return out
	}
	for _, r := range runs {
		out = binary.AppendUvarint(out, uint64(r.length))
		out = binary.AppendUvarint(out, uint64(paletteIndex(palette, r.block)))
	}
	return out
}

// paletteIndex is a linear search; chunk palettes are almost always tiny.
func paletteIndex(palette []Block, b Block) int {
	for i, p := range palette {
		if p == b {
			return i
		}
	}
	return -1
}

// Decode parses data produced by Encode, rejecting anything malformed.
func Decode(data []byte) (*Chunk, error) {
	r := reader{data: data}
	palette, err := readPalette(&r, Volume)
	if err != nil {
		return nil, err
	}
	c := NewChunk(palette[0])
	if len(palette) > 1 {
		c.blocks = new([Volume]Block)
		if err := decodeRuns(&r, palette, c.blocks[:]); err != nil {
			return nil, err
		}
	}
	return c, r.finish()
}

// DecodeBlocks parses data produced by EncodeBlocks, which must hold exactly n blocks.
func DecodeBlocks(data []byte, n int) ([]Block, error) {
	r := reader{data: data}
	palette, err := readPalette(&r, n)
	if err != nil {
		return nil, err
	}
	blocks := make([]Block, n)
	if len(palette) == 1 {
		for i := range blocks {
			blocks[i] = palette[0]
		}
	} else if err := decodeRuns(&r, palette, blocks); err != nil {
		return nil, err
	}
	return blocks, r.finish()
}

// readPalette reads the version and a palette of at most most blocks.
func readPalette(r *reader, most int) ([]Block, error) {
	if r.byte() != encodingVersion {
		return nil, fmt.Errorf("%w: unknown version", errCorrupt)
	}
	n := r.uvarint()
	if n == 0 || n > uint64(most) {
		return nil, fmt.Errorf("%w: palette size %d", errCorrupt, n)
	}
	palette := make([]Block, n)
	for i := range palette {
		id := r.uvarint()
		if id > 0xFFFF {
			return nil, fmt.Errorf("%w: block id %d", errCorrupt, id)
		}
		palette[i] = Block(id)
	}
	return palette, nil
}

func decodeRuns(r *reader, palette []Block, blocks []Block) error {
	for i := 0; i < len(blocks); {
		run, idx := r.uvarint(), r.uvarint()
		if r.err || run == 0 || run > uint64(len(blocks)-i) || idx >= uint64(len(palette)) {
			return fmt.Errorf("%w: bad run at %d", errCorrupt, i)
		}
		for end := i + int(run); i < end; i++ {
			blocks[i] = palette[idx]
		}
	}
	return nil
}

// reader is a byte cursor that latches the first error instead of returning one per call.
type reader struct {
	data []byte
	off  int
	err  bool
}

func (r *reader) byte() byte {
	if r.off >= len(r.data) {
		r.err = true
		return 0
	}
	r.off++
	return r.data[r.off-1]
}

func (r *reader) uvarint() uint64 {
	v, n := binary.Uvarint(r.data[min(r.off, len(r.data)):])
	if n <= 0 {
		r.err = true
		return 0
	}
	r.off += n
	return v
}

// finish fails if reading went wrong or left bytes over.
func (r *reader) finish() error {
	if r.err || r.off != len(r.data) {
		return fmt.Errorf("%w: bad length", errCorrupt)
	}
	return nil
}
