// Package world holds voxel chunks, terrain generation, the world border and chunk storage.
package world

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

// Blocks returns every known block, indexed by ID.
func Blocks() []BlockInfo { return blocks }

// Info returns a block's properties; unknown IDs behave like air.
func Info(b Block) BlockInfo {
	if !b.Valid() {
		return blocks[Air]
	}
	return blocks[b]
}

// Valid reports whether b is a known block.
func (b Block) Valid() bool { return int(b) < len(blocks) }

func (b Block) String() string { return Info(b).Name }
