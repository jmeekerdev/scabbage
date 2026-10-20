/** Procedural 16×16 pixel-art block textures: no image files to download. */
import { MACHINE_PAINTERS } from "./machine-textures";

export const TEXTURE_SIZE = 16;

type RGBA = [number, number, number, number];
export type RGB = [number, number, number];
export type Painter = (x: number, y: number, noise: (salt?: number) => number) => RGBA;

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

/** Wool colors; each paints the server's "<color>_wool" block. */
const WOOL: [string, [number, number, number]][] = [
  ["white", [233, 236, 236]],
  ["orange", [240, 118, 19]],
  ["magenta", [189, 68, 179]],
  ["light_blue", [58, 175, 217]],
  ["yellow", [248, 198, 39]],
  ["lime", [112, 185, 25]],
  ["pink", [237, 141, 172]],
  ["gray", [62, 68, 71]],
  ["light_gray", [142, 142, 134]],
  ["cyan", [21, 137, 145]],
  ["purple", [121, 42, 172]],
  ["blue", [53, 57, 157]],
  ["brown", [114, 71, 40]],
  ["green", [84, 109, 27]],
  ["red", [161, 39, 34]],
  ["black", [30, 31, 35]],
];
export const WOOL_COLORS = WOOL.map(([name]) => name);

/** Soft knitted wool: a faint diagonal weave over fine noise. */
const wool =
  (rgb: [number, number, number]): Painter =>
  (x, y, n) =>
    shade(rgb, ((x + y) % 4 < 2 ? 0.95 : 1.03) * (0.94 + n() * 0.12));

const STONE: [number, number, number] = [125, 125, 125];
const edge = (x: number, y: number) => x === 0 || y === 0 || x === 15 || y === 15;
const inSquare = (x: number, y: number, r: number) =>
  Math.max(Math.abs(x - 7.5), Math.abs(y - 7.5)) < r;

/** A lamp's glass between wooden bars, glowing or dark. */
const lampGlass =
  (glass: [number, number, number]): Painter =>
  (x, y, n) => {
    if (edge(x, y)) return shade([92, 70, 44], 0.9 + n() * 0.2);
    return x % 5 === 0 || y % 5 === 0 ? shade(glass, 0.8) : shade(glass, 0.96 + n() * 0.08);
  };

/** Wire is a cross of red that brightens when powered; the rest is see-through. */
const wire =
  (rgb: [number, number, number]): Painter =>
  (x, y, n) =>
    Math.abs(x - 7.5) < 2 || Math.abs(y - 7.5) < 2 ? shade(rgb, 0.9 + n() * 0.2) : [0, 0, 0, 0];

/** A stone block with a coloured square in the middle: red when off, green when on. */
const knob =
  (rgb: [number, number, number], size: number): Painter =>
  (x, y, n) =>
    inSquare(x, y, size)
      ? shade(rgb, 0.95 + n() * 0.1)
      : shade(STONE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

const plate =
  (k: number): Painter =>
  (x, y, n) =>
    shade([160, 160, 150], (edge(x, y) ? 0.75 : 1) * k * (0.95 + n() * 0.1));

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
  lamp: (x, y, n) => {
    const frame = x === 0 || y === 0 || x === 15 || y === 15;
    const bars = x % 5 === 0 || y % 5 === 0;
    if (frame) return shade([92, 70, 44], 0.9 + n() * 0.2);
    return bars ? shade([200, 150, 60], 0.95) : shade([255, 226, 140], 0.96 + n() * 0.08);
  },
  wire: wire([110, 20, 20]),
  wire_on: wire([255, 50, 40]),
  switch: knob([200, 40, 40], 4),
  switch_on: knob([60, 200, 60], 4),
  button: knob([150, 150, 150], 3),
  button_on: knob([250, 210, 60], 3),
  pressure_plate: plate(1),
  pressure_plate_on: plate(0.8),
  signal_lamp: lampGlass([110, 90, 60]),
  signal_lamp_on: lampGlass([255, 226, 140]),
  // The torch box spans columns 6–9 and rows 0–9, so only that part is painted.
  torch: (x, y, n) => {
    if (x < 6 || x > 9 || y > 9) return [0, 0, 0, 0];
    if (y < 6) return shade([110, 80, 45], 0.9 + n() * 0.2);
    const core = x > 6 && x < 9 && y > 6 && y < 9;
    return core ? [255, 244, 170, 255] : shade([255, 178, 50], 0.95 + n() * 0.1);
  },
  ...Object.fromEntries(WOOL.map(([name, rgb]) => [`${name}_wool`, wool(rgb)])),
  ...MACHINE_PAINTERS,
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
