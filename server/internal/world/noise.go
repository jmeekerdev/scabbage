package world

import "math"

// noise is seeded 2D Perlin noise. Every product is wrapped in float64() so the compiler cannot
// fuse multiply-adds, which keeps results identical across CPU architectures.
type noise struct{ seed uint64 }

func newNoise(seed int64) noise { return noise{mix(uint64(seed))} }

// fbm sums octaves of Perlin noise, normalized to [-1, 1].
func (n noise) fbm(x, z float64, octaves int) float64 {
	sum, amp, norm, freq := 0.0, 1.0, 0.0, 1.0
	for i := range octaves {
		sum += float64(amp * perlin(mix(n.seed+uint64(i)), float64(x*freq), float64(z*freq)))
		norm += amp
		amp /= 2
		freq *= 2
	}
	return max(-1, min(1, sum/norm))
}

func perlin(seed uint64, x, z float64) float64 {
	fx, fz := math.Floor(x), math.Floor(z)
	ix, iz := int32(fx), int32(fz)
	dx, dz := x-fx, z-fz
	u, v := fade(dx), fade(dz)
	a := grad(hash(seed, ix, iz), dx, dz)
	b := grad(hash(seed, ix+1, iz), dx-1, dz)
	c := grad(hash(seed, ix, iz+1), dx, dz-1)
	d := grad(hash(seed, ix+1, iz+1), dx-1, dz-1)
	return lerp(lerp(a, b, u), lerp(c, d, u), v)
}

// grad dots one of 8 fixed gradient directions with (dx, dz), using only additions.
func grad(h uint64, dx, dz float64) float64 {
	switch h & 7 {
	case 0:
		return dx + dz
	case 1:
		return dx - dz
	case 2:
		return -dx + dz
	case 3:
		return -dx - dz
	case 4:
		return dx
	case 5:
		return -dx
	case 6:
		return dz
	default:
		return -dz
	}
}

// fade is smoothstep: 3t² - 2t³.
func fade(t float64) float64 { return float64(t*t) * (3 - float64(2*t)) }

func lerp(a, b, t float64) float64 { return a + float64(t*(b-a)) }

// hash mixes a seed and a column into well-distributed bits.
func hash(seed uint64, x, z int32) uint64 {
	return mix(seed ^ mix(uint64(uint32(x))<<32|uint64(uint32(z))))
}

// mix is the splitmix64 finalizer.
func mix(v uint64) uint64 {
	v ^= v >> 30
	v *= 0xbf58476d1ce4e5b9
	v ^= v >> 27
	v *= 0x94d049bb133111eb
	return v ^ v>>31
}
