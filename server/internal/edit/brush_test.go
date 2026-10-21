package edit

import (
	"testing"
	"time"

	"scabbage/internal/world"
)

func TestBrushCoversACubeFromItsLowestCorner(t *testing.T) {
	cells := Brush{X: 4, Y: 9, Z: -2, Size: 2, Block: world.Planks}.Cells()
	if len(cells) != 8 {
		t.Fatalf("cells = %d, want 8", len(cells))
	}
	for _, c := range cells {
		if c.X < 4 || c.X > 5 || c.Y < 9 || c.Y > 10 || c.Z < -2 || c.Z > -1 || c.Block != world.Planks {
			t.Fatalf("cell %+v outside the brush", c)
		}
	}
}

func TestBrushSizeIsKeptBetweenOneAndAnOldBlock(t *testing.T) {
	if n := len((Brush{Size: 0}).Cells()); n != 1 {
		t.Fatalf("size 0 = %d cells, want 1", n)
	}
	if n := len((Brush{Size: 50}).Cells()); n != MaxBrush*MaxBrush*MaxBrush {
		t.Fatalf("size 50 = %d cells", n)
	}
}

func TestBrushReachIsMeasuredToItsMiddle(t *testing.T) {
	far := Brush{X: 5*world.Metre + 1, Y: 15, Z: 0, Size: MaxBrush}
	if !far.InReach(feet) {
		t.Fatal("a whole brush just past 5 m should be in reach")
	}
	if (Brush{X: 7 * world.Metre, Y: 15, Z: 0, Size: MaxBrush}).InReach(feet) {
		t.Fatal("a brush 7 m away is out of reach")
	}
}

func TestDigTimeIsTheSlowestBlockTimesTheBrushWidth(t *testing.T) {
	stone := time.Duration(world.Info(world.Stone).MineMs) * time.Millisecond
	got := DigTime([]world.Block{world.Dirt, world.Stone, world.Torch}, 3)
	if got != 3*stone {
		t.Fatalf("dig time = %v, want %v", got, 3*stone)
	}
	if DigTime(nil, 3) != 0 {
		t.Fatal("nothing to dig takes no time")
	}
}
