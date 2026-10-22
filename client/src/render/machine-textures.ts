/** Textures for the logic machines: doors, pistons, diodes, gates, delays, relays, capacitors, toggles, radios, sensors, timers, counters, teleporters, note blocks, portals, scripts. */
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
const SLATE: RGB = [70, 72, 80];
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

/** Dark slate with a pale stripe running back to front, so its sides show which way it points. */
const diodeSide: Painter = (x, y, n) =>
  y === 7 || y === 8 ? [200, 190, 170, 255] : shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

/** Slate around a red light that glows while it gives power. */
const diodeFront =
  (light: RGB): Painter =>
  (x, y, n) =>
    Math.hypot(x - 7.5, y - 7.5) < 3.5
      ? shade(light, 0.95 + n() * 0.1)
      : shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

/** A clock face on stone: dark between beats, bright on them. */
export const timer =
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

// 3×5 letters for gate labels, rows as in DIGITS.
const LETTERS: Record<string, number[]> = {
  A: [2, 5, 7, 5, 5],
  D: [6, 5, 5, 5, 6],
  N: [5, 7, 7, 5, 5],
  O: [7, 5, 5, 5, 7],
  R: [6, 5, 6, 5, 5],
  T: [7, 2, 2, 2, 2],
  X: [5, 5, 2, 5, 5],
};

/** Whether the pixel is lit in `text` written in 3×5 letters, centered across rows 6 to 10. */
export function labelPixel(text: string, x: number, y: number): boolean {
  const from = Math.floor((16 - (4 * text.length - 1)) / 2);
  const [col, row] = [x - from, 10 - y];
  if (col < 0 || col % 4 === 3 || row < 0 || row > 4) return false;
  const letter = LETTERS[text[Math.floor(col / 4)] ?? ""];
  return !!letter && ((letter[row]! >> (2 - (col % 4))) & 1) === 1;
}

// A gate's modes, in the order tapping cycles them; mirrors world.GateModes.
export const GATE_MODES = ["and", "or", "xor", "nand", "nor", "not"];

/** Slate labeled with the gate's mode, the letters glowing while it gives power. */
const gate =
  (mode: string, ink: RGB): Painter =>
  (x, y, n) =>
    labelPixel(mode.toUpperCase(), x, y)
      ? shade(ink, 0.95 + n() * 0.1)
      : shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

/** Which of two rows of 2-pixel pips, top row first, covers the pixel, or -1 for none. */
export function pipAt(x: number, y: number, perRow = 4): number {
  const row = y >= 11 && y <= 12 ? 0 : y >= 3 && y <= 4 ? 1 : -1;
  const from = Math.ceil((16 - (3 * perRow - 1)) / 2);
  const col = Math.floor((x - from) / 3);
  if (row < 0 || x < from || col >= perRow || (x - from) % 3 === 2) return -1;
  return row * perRow + col;
}

/** A diode's side with a pip lit for each quarter second the delay waits. */
const delay =
  (quarters: number): Painter =>
  (x, y, n) => {
    const pip = pipAt(x, y);
    if (pip < 0) return diodeSide(x, y, n);
    return pip < quarters ? shade([240, 200, 90], 0.95 + n() * 0.1) : shade(SLATE, 0.55);
  };

/** A diode's side with a blue pip lit for each second the capacitor holds its power. */
const capacitor =
  (seconds: number): Painter =>
  (x, y, n) => {
    const pip = pipAt(x, y, 5);
    if (pip < 0) return diodeSide(x, y, n);
    return pip < seconds ? shade([110, 200, 250], 0.95 + n() * 0.1) : shade(SLATE, 0.55);
  };

/** Slate around a rocker switch whose top half glows while the toggle gives power. */
const toggle =
  (giving: boolean): Painter =>
  (x, y, n) => {
    if (x < 5 || x > 10 || y < 3 || y > 12) return shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);
    const lit = y >= 8 === giving;
    return lit
      ? shade(giving ? [255, 70, 50] : [110, 30, 30], 0.95 + n() * 0.1)
      : shade(SLATE, 0.5);
  };

/** A diode's side crossed by a second stripe, for the side input that lets power through. */
const relaySide: Painter = (x, y, n) =>
  (x === 7 || x === 8) && y > 1 && y < 14 ? [200, 190, 170, 255] : diodeSide(x, y, n);

/** Stone around a dark window holding an amber figure: a person, or a four-legged creature. */
const detector =
  (shape: (x: number, y: number) => boolean): Painter =>
  (x, y, n) => {
    if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    return shape(x, y) ? shade([250, 180, 60], 0.95 + n() * 0.1) : [30, 32, 38, 255];
  };
const person = (x: number, y: number) =>
  Math.hypot(x - 7.5, y - 10) < 2 || (x >= 6 && x <= 9 && y >= 3 && y <= 8);
const creature = (x: number, y: number) =>
  (x >= 3 && x <= 10 && y >= 6 && y <= 9) ||
  (x >= 10 && x <= 12 && y >= 8 && y <= 11) ||
  ((x === 4 || x === 9) && y >= 3 && y < 6);

/** Stone around a gray sky with a cloud and falling drops, for the weather sensor. */
const weatherFace: Painter = (x, y, n) => {
  if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
  const cloud = Math.hypot(x - 5.5, y - 10) < 2.6 || Math.hypot(x - 9.5, y - 10.5) < 2.4;
  if (cloud) return shade([225, 228, 235], 0.93 + n() * 0.1);
  const drop = y >= 3 && y <= 7 && (x + y) % 4 === 0 && (x - y + 16) % 3 !== 0;
  return drop ? shade([110, 170, 240], 0.95 + n() * 0.1) : shade([95, 105, 120], 0.95 + n() * 0.1);
};

// A clock sensor's sky color, and where its sun or moon sits, for each part of the day.
const CLOCK_FACES: Record<string, { sky: RGB; at: [number, number]; moon: boolean }> = {
  day: { sky: [110, 170, 230], at: [7.5, 9], moon: false },
  night: { sky: [25, 30, 70], at: [7.5, 9], moon: true },
  dawn: { sky: [240, 150, 90], at: [4, 6.5], moon: false },
  dusk: { sky: [170, 80, 120], at: [11, 6.5], moon: false },
  noon: { sky: [70, 150, 245], at: [7.5, 11.5], moon: false },
  midnight: { sky: [8, 8, 20], at: [7.5, 11.5], moon: true },
};
export const CLOCK_MODES = Object.keys(CLOCK_FACES);

/** A window on its part of the day: the sky's color over a strip of ground, with the sun or moon. */
const clockFace =
  (mode: string): Painter =>
  (x, y, n) => {
    const { sky, at, moon } = CLOCK_FACES[mode];
    if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    if (Math.hypot(x - at[0], y - at[1]) < 2)
      return moon ? [230, 230, 240, 255] : [255, 220, 60, 255];
    return y < 5 ? shade([70, 120, 50], 0.9 + n() * 0.2) : shade(sky, 0.97 + n() * 0.06);
  };

/** Slate around an eye whose pupil glows while it sees a block. */
const detectorEye =
  (pupil: RGB): Painter =>
  (x, y, n) => {
    const r = Math.hypot((x - 7.5) / 1.5, y - 7.5);
    if (r < 1.6) return shade(pupil, 0.95 + n() * 0.1);
    if (r < 3.5) return [230, 230, 220, 255];
    return shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);
  };

/** Wireless channel colors; mirrors the wool colors. */
const CHANNEL_COLORS: RGB[] = [
  [233, 236, 236],
  [240, 118, 19],
  [189, 68, 179],
  [58, 175, 217],
  [248, 198, 39],
  [112, 185, 25],
  [237, 141, 172],
  [62, 68, 71],
  [142, 142, 134],
  [21, 137, 145],
  [121, 42, 172],
  [53, 57, 157],
  [114, 71, 40],
  [84, 109, 27],
  [161, 39, 34],
  [30, 31, 35],
];
export const CHANNELS = CHANNEL_COLORS.length;

/** Stone around the channel's color: rings spreading out for a transmitter, a dot for a receiver. */
const radio =
  (color: RGB, sends: boolean): Painter =>
  (x, y, n) => {
    if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    const r = Math.hypot(x - 7.5, y - 7.5);
    const inked = sends ? Math.floor(r) % 3 !== 2 : r < 3.5 || (r > 4.5 && r < 5.5);
    return inked ? shade(color, 0.95 + n() * 0.1) : [35, 35, 40, 255];
  };

/** Slate with a little aerial light on top, glowing while the block is on. */
const radioTop =
  (light: RGB): Painter =>
  (x, y, n) =>
    Math.hypot(x - 7.5, y - 7.5) < 2.5
      ? shade(light, 0.95 + n() * 0.1)
      : shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

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

// "</>" in pixels, for script blocks.
const CODE_SIGN = new Set(
  [
    [5, 5],
    [4, 6],
    [3, 7],
    [4, 8],
    [5, 9],
    [9, 5],
    [8, 6],
    [8, 7],
    [7, 8],
    [7, 9],
    [10, 5],
    [11, 6],
    [12, 7],
    [11, 8],
    [10, 9],
  ].map(([x, y]) => `${x},${y}`),
);

/** A little screen in a stone frame showing "</>", lit up while powered. */
const scriptBlock =
  (glow: RGB): Painter =>
  (x, y, n) => {
    if (edge(x, y, 2)) return shade(STONE, edge(x, y) ? 0.6 : 0.85 + n() * 0.3);
    if (CODE_SIGN.has(`${x},${y}`)) return shade(glow, 0.95 + n() * 0.1);
    return shade([24, 32, 56], 0.9 + n() * 0.2);
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
  diode_side: diodeSide,
  diode_front: diodeFront([110, 30, 30]),
  diode_front_on: diodeFront([255, 70, 50]),
  ...Object.fromEntries(
    GATE_MODES.flatMap((m) => [
      [`gate_${m}`, gate(m, [200, 190, 170])],
      [`gate_${m}_on`, gate(m, [255, 90, 60])],
    ]),
  ),
  ...Object.fromEntries(Array.from({ length: 8 }, (_, i) => [`delay_${i + 1}`, delay(i + 1)])),
  relay_side: relaySide,
  ...Object.fromEntries(
    Array.from({ length: 10 }, (_, i) => [`capacitor_${i + 1}`, capacitor(i + 1)]),
  ),
  toggle: toggle(false),
  toggle_1: toggle(true),
  ...Object.fromEntries(
    CHANNEL_COLORS.flatMap((c, i) => [
      [`transmitter_${i}`, radio(c, true)],
      [`receiver_${i}`, radio(c, false)],
    ]),
  ),
  radio_top: radioTop([110, 30, 30]),
  radio_top_on: radioTop([255, 70, 50]),
  player_detector: detector(person),
  creature_detector: detector(creature),
  ...Object.fromEntries(CLOCK_MODES.map((m) => [`clock_${m}`, clockFace(m)])),
  weather_sensor: weatherFace,
  block_detector_eye: detectorEye([110, 30, 30]),
  block_detector_eye_on: detectorEye([255, 70, 50]),
  timer: timer([150, 150, 130]),
  timer_on: timer([255, 240, 150]),
  ...Object.fromEntries(DIGITS.map((_, d) => [`counter_${d}`, counter(d, d === 9)])),
  ...Object.fromEntries(COUNT_DOWN.map((d) => [`counter_down_${d}`, counter(d, true)])),
  teleporter,
  portal,
  ...Object.fromEntries(PITCH_COLORS.map((c, i) => [`note_${i}`, noteBlock(c)])),
  script_block: scriptBlock([90, 170, 220]),
  script_block_on: scriptBlock([160, 240, 255]),
};
