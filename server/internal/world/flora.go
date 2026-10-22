package world

const (
	meadowScale = 24.0 * Meter
	tuftsSparse = 20  // tufts per thousand blocks of grass on poor ground...
	tuftsLush   = 160 // ...and in the lushest meadows
	bloomScale  = 10.0 * Meter
	bloomCover  = 0.35 // bloom noise above this grows flowers
	bloomOdds   = 200  // flowers per thousand blocks of grass where they grow
	patchSize   = 8 * Meter
	oddColor    = 5 // one flower in this many ignores its patch's color
)

// plantAt picks what grows on column x,z: a flower, a tuft of grass or nothing.
func (g *GenV3) plantAt(x, z int32, c column) Block {
	if c.top != Grass {
		return Air
	}
	r := hash(g.plants, x, z)
	roll := r % 1000
	if roll < bloomOdds && g.blooms.fbm(float64(x)/bloomScale, float64(z)/bloomScale, 2) > bloomCover {
		return g.flowerAt(x, z, r>>10)
	}
	lush := max(0, g.meadow.fbm(float64(x)/meadowScale, float64(z)/meadowScale, 2))
	if roll < tuftsSparse+uint64(lush*(tuftsLush-tuftsSparse)) {
		return TallGrass
	}
	return Air
}

// flowerAt gives most flowers in a patch the patch's color, and a few another.
func (g *GenV3) flowerAt(x, z int32, r uint64) Block {
	pick := hash(g.plants, floorDiv(x, patchSize), floorDiv(z, patchSize)) >> 8
	if r%oddColor == 0 {
		pick = r >> 8
	}
	return RedFlower + Block(pick%uint64(len(flowerColors)))
}
