/** Textures for the logic machines: doors, pistons, timers, counters, teleporters, note blocks, portals. */
import type { Painter, RGB } from "./textures";

const shade = ([r, g, b]: RGB, k: number): [number, number, number, number] => [
  Math.min(255, Math.round(r * k)),
  Math.min(255, Math.round(g * k)),
  Math.min(255, Math.round(b * k)),
  255,
];
const CLEAR: [number, number, number, number] = [0, 0, 0, 0];
const WOOD: RGB = [166, 132, 80];
const DARK_WOOD: RGB = [110, 82, 48];
const STONE: RGB = [125, 125, 125];
const edge = (x: number, y: number, w = 1) => x < w || y < w || x > 15 - w || y > 15 - w;

/** A wooden door with two panels and a brass knob; open, only its frame is left. */
const door: Painter = (x, y, n) => {
  if (edge(x, y, 2)) return shade(DARK_WOOD, 0.9 + n() * 0.2);
  if (x >= 11 && x <= 12 && y >= 7 && y <= 8) return [230, 190, 70, 255];
  const panel = y === 8 || x === 7;
  return shade(WOOD, (panel ? 0.75 : 1) * (0.94 + n() * 0.12));
};
const doorFrame: Painter = (x, y, n) => (edge(x, y, 2) ? shade(DARK_WOOD, 0.9 + n() * 0.2) : CLEAR);

const pistonSide: Painter = (x, y, n) => shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);

/** Planks around an iron pusher that shines while pushing. */
const pistonFront =
  (iron: RGB): Painter =>
  (x, y, n) =>
    Math.max(Math.abs(x - 7.5), Math.abs(y - 7.5)) < 4
      ? shade(iron, 0.95 + n() * 0.1)
      : shade(WOOD, (edge(x, y) ? 0.7 : 1) * (0.94 + n() * 0.12));

/** A clock face on stone: dark between beats, bright on them. */
const timer =
  (face: RGB): Painter =>
  (x, y, n) => {
    const r = Math.hypot(x - 7.5, y - 7.5);
    if (r > 6) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    const hand = (x === 7 || x === 8) && y >= 7 && y <= 12;
    const otherHand = (y === 7 || y === 8) && x >= 7 && x <= 11;
    return hand || otherHand || r > 5 ? [40, 40, 40, 255] : shade(face, 0.97 + n() * 0.06);
  };

// 3×5 digits, top row first; each row's bits are read left to right.
const DIGITS = [
  [7, 5, 5, 5, 7],
  [2, 6, 2, 2, 7],
  [7, 1, 7, 4, 7],
  [7, 1, 7, 1, 7],
  [5, 5, 7, 1, 1],
  [7, 4, 7, 1, 7],
  [7, 4, 7, 5, 7],
  [7, 1, 2, 2, 2],
  [7, 5, 7, 5, 7],
  [7, 5, 7, 1, 7],
];

/** Whether the pixel is lit in digit d drawn at double size in the middle of the face. */
export function digitPixel(d: number, x: number, y: number): boolean {
  const [col, row] = [Math.floor((x - 5) / 2), 4 - Math.floor((y - 3) / 2)];
  if (col < 0 || col > 2 || row < 0 || row > 4) return false;
  return ((DIGITS[d]![row]! >> (2 - col)) & 1) === 1;
}

// The counts a counter shows on its way back down from 9, giving power all along.
export const COUNT_DOWN = [1, 2, 3, 4, 5, 6, 7, 8];

/** A stone display showing its count in green, brighter while it gives power. */
const counter =
  (d: number, giving: boolean): Painter =>
  (x, y, n) => {
    if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    if (!digitPixel(d, x, y)) return [20, 30, 20, 255];
    return giving ? [140, 255, 120, 255] : [60, 200, 60, 255];
  };

/** Purple rings that swirl toward the middle. */
const teleporter: Painter = (x, y, n) => {
  const r = Math.hypot(x - 7.5, y - 7.5);
  const ring = Math.floor(r + Math.atan2(y - 7.5, x - 7.5) / Math.PI) % 2 === 0;
  return shade(ring ? [170, 80, 230] : [90, 30, 140], 0.9 + n() * 0.2);
};

/** A gold frame around a shimmering green middle. */
const portal: Painter = (x, y, n) => {
  if (edge(x, y, 2)) return shade([220, 180, 60], edge(x, y) ? 0.7 : 0.95 + n() * 0.1);
  return shade([60, 220, 140], 0.7 + ((x + y) % 4) * 0.08 + n() * 0.1);
};

// One color per pitch, low to high, like a rainbow xylophone.
const PITCH_COLORS: RGB[] = [
  [230, 60, 60],
  [240, 140, 50],
  [240, 220, 60],
  [80, 200, 70],
  [60, 200, 210],
  [70, 110, 230],
  [150, 80, 220],
  [235, 110, 190],
];

/** A wooden box with a note in its pitch's color. */
const noteBlock =
  (color: RGB): Painter =>
  (x, y, n) => {
    const head = Math.hypot(x - 6.5, y - 5) < 2.2;
    const stem = x === 8 && y >= 5 && y <= 12;
    const flag = (x === 9 || x === 10) && y >= 10 && y <= 12 - (x - 9);
    if (head || stem || flag) return shade(color, 0.95 + n() * 0.1);
    return shade(DARK_WOOD, (edge(x, y) ? 0.7 : 1) * (0.94 + n() * 0.12));
  };

export const MACHINE_PAINTERS: Record<string, Painter> = {
  door,
  door_open: doorFrame,
  piston_side: pistonSide,
  piston_front: pistonFront([150, 150, 160]),
  piston_front_on: pistonFront([230, 230, 240]),
  timer: timer([150, 150, 130]),
  timer_on: timer([255, 240, 150]),
  ...Object.fromEntries(DIGITS.map((_, d) => [`counter_${d}`, counter(d, d === 9)])),
  ...Object.fromEntries(COUNT_DOWN.map((d) => [`counter_down_${d}`, counter(d, true)])),
  teleporter,
  portal,
  ...Object.fromEntries(PITCH_COLORS.map((c, i) => [`note_${i}`, noteBlock(c)])),
};
