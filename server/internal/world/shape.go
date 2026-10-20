package world

// Shape is a block's geometry; the numbers match pb.Shape and client/src/world/shapes.ts.
type Shape uint8

const (
	Cube Shape = iota
	Slab
	StairsNorth // the tall half is toward -z; east, south and west follow in order
	StairsEast
	StairsSouth
	StairsWest
	TorchShape
	Flat // a thin layer on the floor, like wire
)

// Box is part of a block's collision volume, in block-local coordinates from 0 to 1.
type Box struct{ Min, Max [3]float64 }

var FullBox = Box{Max: [3]float64{1, 1, 1}}

var shapeBoxes = [...][]Box{
	Cube:        {FullBox},
	Slab:        {lowerHalf},
	StairsNorth: {lowerHalf, {Min: [3]float64{0, 0.5, 0}, Max: [3]float64{1, 1, 0.5}}},
	StairsEast:  {lowerHalf, {Min: [3]float64{0.5, 0.5, 0}, Max: [3]float64{1, 1, 1}}},
	StairsSouth: {lowerHalf, {Min: [3]float64{0, 0.5, 0.5}, Max: [3]float64{1, 1, 1}}},
	StairsWest:  {lowerHalf, {Min: [3]float64{0, 0.5, 0}, Max: [3]float64{0.5, 1, 1}}},
	TorchShape:  nil,
	Flat:        {{Max: [3]float64{1, 0.125, 1}}},
}

var lowerHalf = Box{Max: [3]float64{1, 0.5, 1}}

// Boxes is what players collide with; non-solid blocks have none.
func (i BlockInfo) Boxes() []Box {
	if !i.Solid || int(i.Shape) >= len(shapeBoxes) {
		return nil
	}
	return shapeBoxes[i.Shape]
}

// Targetable reports whether players can aim at the block to break it.
func (i BlockInfo) Targetable() bool { return i.Solid || i.Shape == TorchShape || i.Shape == Flat }
