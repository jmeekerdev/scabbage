package world

import "testing"

// realTop is the y above column x,z's highest block that isn't a plant, and that block.
func realTop(at func(x, y, z int32) Block, x, z int32) (int32, Block) {
	for y := int32(MaxY - 1); y > MinY; y-- {
		if b := at(x, y, z); b != Air && !Info(b).Plant() {
			return y + 1, b
		}
	}
	return MinY, Air
}

func TestFarTilesFollowTheGround(t *testing.T) {
	g := NewGenV3(42)
	at := cachedBlocks(g)
	leafy := 0
	for tx := int32(-2); tx < 2; tx++ {
		for tz := int32(-2); tz < 2; tz++ {
			tile := g.FarTile(tx, tz)
			for i := range FarCells * FarCells {
				x, z := FarSample(tx, tz, i)
				want, b := realTop(at, x, z)
				got := tile.Tops[i]
				switch tile.Blocks[i] {
				case Leaves, PineLeaves:
					leafy++
					if got < want-4*Metre || got > want+4*Metre {
						t.Fatalf("crown at %d,%d is %d high, the tree %d", x, z, got, want)
					}
				case b:
					onRock := isStone(b) && got < want
					if b != Leaves && b != PineLeaves && b != Log && !onRock && got != want {
						t.Fatalf("ground at %d,%d is %d high, want %d", x, z, got, want)
					}
				default:
					if b != Leaves && b != PineLeaves && b != Log && !isStone(b) {
						t.Fatalf("far %v at %d,%d over %v", tile.Blocks[i], x, z, b)
					}
				}
			}
		}
	}
	if leafy == 0 {
		t.Fatal("no tree crowns in 16 tiles")
	}
}

// isStone covers boulders and rocks, which far land leaves out.
func isStone(b Block) bool { return b == Stone || b == Cobblestone }

func TestFarTilesAreDeterministic(t *testing.T) {
	a, b := NewGenV3(7), NewGenV3(7)
	if a.FarTile(3, -5) != b.FarTile(3, -5) {
		t.Fatal("same seed gave different far land")
	}
}

func BenchmarkFarTile(b *testing.B) {
	g := NewGenV3(42)
	for i := range b.N {
		g.FarTile(int32(i%64), int32(i/64%64))
	}
}
