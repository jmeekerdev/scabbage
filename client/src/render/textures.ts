/** Procedural 16×16 pixel-art block textures: no image files to download. */
export const TEXTURE_SIZE = 16;

type RGBA = [number, number, number, number];
type Painter = (x: number, y: number, noise: (salt?: number) => number) => RGBA;

const shade = ([r, g, b]: [number, number, number], k: number, a = 255): RGBA => [
  clamp(r * k),
  clamp(g * k),
  clamp(b * k),
  a,
];
const clamp = (v: number) => Math.max(0, Math.min(255, Math.round(v)));
const speckle =
  (rgb: [number, number, number], spread: number): Painter =>
  (_x, _y, n) =>
    shade(rgb, 1 - spread + n() * spread * 2);

const DIRT: [number, number, number] = [134, 96, 67];
const GRASS: [number, number, number] = [95, 159, 53];

// Row 0 is the bottom of the texture (v = 0), so side textures read bottom-up.
const PAINTERS: Record<string, Painter> = {
  missing: (x, y) => (((x >> 3) + (y >> 3)) % 2 ? [255, 0, 255, 255] : [20, 20, 20, 255]),
  stone: speckle([125, 125, 125], 0.12),
  dirt: speckle(DIRT, 0.15),
  grass_top: speckle(GRASS, 0.15),
  grass_side: (_x, y, n) =>
    y >= 12 + Math.floor(n(1) * 2) ? shade(GRASS, 0.9 + n() * 0.2) : shade(DIRT, 0.85 + n() * 0.3),
  sand: speckle([219, 207, 163], 0.06),
  water: (_x, _y, n) => shade([50, 105, 200], 0.92 + n() * 0.16, 170),
  log_side: (x, _y, n) => shade([102, 81, 51], (x % 4 === 0 ? 0.75 : 1) * (0.9 + n() * 0.2)),
  log_top: (x, y, n) => {
    const r = Math.hypot(x - 7.5, y - 7.5);
    return r > 6.5
      ? shade([102, 81, 51], 0.9)
      : shade([176, 144, 92], (Math.floor(r) % 2 ? 0.85 : 1) * (0.95 + n() * 0.1));
  },
  leaves: (_x, _y, n) => (n(2) < 0.22 ? [0, 0, 0, 0] : shade([58, 128, 38], 0.8 + n() * 0.4)),
  planks: (x, y, n) => {
    const seam = y % 4 === 3 || x === ((y >> 2) % 2) * 8;
    return shade([166, 132, 80], (seam ? 0.7 : 1) * (0.94 + n() * 0.12));
  },
  cobblestone: (x, y, n) => {
    const mortar = (x + (y >> 2) * 2) % 5 === 0 || y % 4 === 0;
    return shade([128, 128, 128], mortar ? 0.6 : 0.85 + n((x / 5) | 0) * 0.3);
  },
  glass: (x, y) => {
    const edge = x === 0 || y === 0 || x === 15 || y === 15;
    const glint = (x === y || x === y + 1) && x > 3 && x < 8;
    return edge || glint ? [220, 240, 250, 255] : [0, 0, 0, 0];
  },
  brick: (x, y, n) => {
    const row = y >> 2;
    const mortar = y % 4 === 0 || (x + (row % 2) * 4) % 8 === 0;
    return mortar
      ? shade([200, 195, 185], 0.95 + n() * 0.1)
      : shade([155, 64, 48], 0.88 + n() * 0.24);
  },
  bedrock: (_x, _y, n) => shade([60, 60, 60], 0.4 + n() * 0.9),
};

export const TEXTURE_NAMES = Object.keys(PAINTERS);

/** RGBA pixels for every texture, one 16×16 layer after another, in TEXTURE_NAMES order. */
export function paintTextures(): Uint8Array {
  const px = TEXTURE_SIZE * TEXTURE_SIZE;
  const out = new Uint8Array(TEXTURE_NAMES.length * px * 4);
  TEXTURE_NAMES.forEach((name, layer) => {
    for (let y = 0; y < TEXTURE_SIZE; y++) {
      for (let x = 0; x < TEXTURE_SIZE; x++) {
        const noise = (salt = 0) => hash(layer * 7919 + salt * 104729 + y * TEXTURE_SIZE + x);
        out.set(PAINTERS[name](x, y, noise), (layer * px + y * TEXTURE_SIZE + x) * 4);
      }
    }
  });
  return out;
}

/** A deterministic 0–1 value per integer (a small integer hash, not a quality RNG). */
function hash(n: number): number {
  let h = Math.imul(n ^ 0x9e3779b9, 0x85ebca6b);
  h = Math.imul(h ^ (h >>> 13), 0xc2b2ae35);
  return ((h ^ (h >>> 16)) >>> 0) / 0x100000000;
}
