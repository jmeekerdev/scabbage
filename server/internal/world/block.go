// Package world holds voxel chunks, terrain generation, the world border and chunk storage.
package world

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
)

// Block is a block type ID. IDs are stored in chunks, so existing values must never change.
type Block uint16

// Starter blocks. Append new blocks at the end.
const (
	Air Block = iota
	Stone
	Dirt
	Grass
	Sand
	Water
	Log
	Leaves
	Planks
	Cobblestone
	Glass
	Brick
	Bedrock
)

// BlockInfo describes how a block behaves.
type BlockInfo struct {
	ID          Block
	Name        string
	Solid       bool // collides with players
	Transparent bool // neighbors' faces stay visible
}

var blocks = []BlockInfo{
	{Air, "air", false, true},
	{Stone, "stone", true, false},
	{Dirt, "dirt", true, false},
	{Grass, "grass", true, false},
	{Sand, "sand", true, false},
	{Water, "water", false, true},
	{Log, "log", true, false},
	{Leaves, "leaves", true, true},
	{Planks, "planks", true, false},
	{Cobblestone, "cobblestone", true, false},
	{Glass, "glass", true, true},
	{Brick, "brick", true, false},
	{Bedrock, "bedrock", true, false},
}

// FirstCustom is the lowest ID for blocks players paint; built-ins stay below it.
const FirstCustom Block = 256

var (
	ErrBuiltinID = errors.New("custom block IDs start at FirstCustom")
	customMu     sync.Mutex                          // serializes Register
	custom       atomic.Pointer[map[Block]BlockInfo] // copy-on-write, so lookups never lock
)

// Register adds or replaces a custom block, e.g. one approved by an admin.
func Register(b BlockInfo) error {
	if b.ID < FirstCustom {
		return ErrBuiltinID
	}
	customMu.Lock()
	defer customMu.Unlock()
	next := map[Block]BlockInfo{b.ID: b}
	if old := custom.Load(); old != nil {
		for id, info := range *old {
			if id != b.ID {
				next[id] = info
			}
		}
	}
	custom.Store(&next)
	return nil
}

// Blocks returns every known block in ID order: built-ins (indexed by ID), then custom blocks.
func Blocks() []BlockInfo {
	out := slices.Clone(blocks)
	if m := custom.Load(); m != nil {
		start := len(out)
		for _, info := range *m {
			out = append(out, info)
		}
		slices.SortFunc(out[start:], func(a, b BlockInfo) int { return int(a.ID) - int(b.ID) })
	}
	return out
}

// IsBuiltinName reports whether a built-in block is called name.
func IsBuiltinName(name string) bool {
	for _, b := range blocks {
		if b.Name == name {
			return true
		}
	}
	return false
}

// Info returns a block's properties; unknown IDs behave like air.
func Info(b Block) BlockInfo {
	if int(b) < len(blocks) {
		return blocks[b]
	}
	if m := custom.Load(); m != nil {
		if info, ok := (*m)[b]; ok {
			return info
		}
	}
	return blocks[Air]
}

// Valid reports whether b is a known block.
func (b Block) Valid() bool { return b == Air || Info(b).ID == b }

func (b Block) String() string { return Info(b).Name }
