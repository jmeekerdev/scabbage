/** Textures for the newer powered devices: sticky pistons, trapdoors, fence gates, launch pads, number displays, timer speeds and tripwires. */
import { digitPixel, timer } from "./machine-textures";
import type { Painter, RGB } from "./textures";

const shade = ([r, g, b]: RGB, k: number): [number, number, number, number] => [
  Math.min(255, Math.round(r * k)),
  Math.min(255, Math.round(g * k)),
  Math.min(255, Math.round(b * k)),
  255,
];
const dim = ([r, g, b]: RGB): RGB => [r * 0.55, g * 0.55, b * 0.55];
const CLEAR: [number, number, number, number] = [0, 0, 0, 0];
const WOOD: RGB = [166, 132, 80];
const DARK_WOOD: RGB = [110, 82, 48];
const IRON: RGB = [80, 80, 88];
const SLATE: RGB = [70, 72, 80];
const edge = (x: number, y: number, w = 1) => x < w || y < w || x > 15 - w || y > 15 - w;

/** Planks around a green, sticky pusher that shines while pushing. */
const stickyFront =
  (slime: RGB): Painter =>
  (x, y, n) =>
    Math.max(Math.abs(x - 7.5), Math.abs(y - 7.5)) < 4
      ? shade(slime, 0.9 + n() * 0.2)
      : shade(WOOD, (edge(x, y) ? 0.7 : 1) * (0.94 + n() * 0.12));

/** A planked hatch with iron hinges; open, only a thin rim shows where it is. */
const trapdoor: Painter = (x, y, n) => {
  if (edge(x, y)) return shade(DARK_WOOD, 0.9 + n() * 0.2);
  if ((y === 3 || y === 12) && x < 6) return shade(IRON, 0.9 + n() * 0.2);
  return shade(WOOD, (y % 5 === 0 ? 0.75 : 1) * (0.94 + n() * 0.12));
};
const trapdoorRim: Painter = (x, y, n) => (edge(x, y) ? shade(DARK_WOOD, 0.9 + n() * 0.2) : CLEAR);

/** Posts at the ends joined by two rails and a brace, see-through between; open, only posts. */
const fenceGate =
  (shut: boolean): Painter =>
  (x, y, n) => {
    const post = x < 2 || x > 13;
    const rail = (y >= 3 && y <= 4) || (y >= 11 && y <= 12) || Math.abs(x - y) < 1;
    return post || (shut && rail) ? shade(WOOD, 0.9 + n() * 0.2) : CLEAR;
  };

/** Whether the pixel is in an arrow pointing along (dx, dy) in texture pixels. */
export function arrowPixel(dx: number, dy: number, x: number, y: number): boolean {
  const along = (x - 7.5) * dx + (y - 7.5) * dy;
  const across = Math.abs((x - 7.5) * dy - (y - 7.5) * dx);
  if (along >= 0) return across <= 6.5 - along;
  return along >= -6 && across < 1.5;
}

/** Rings around the middle, for a pad that throws straight up. */
const ringPixel = (x: number, y: number) => Math.round(Math.hypot(x - 7.5, y - 7.5)) % 3 === 0;

/** Slate with a yellow mark that brightens while powered. */
const pad =
  (mark: (x: number, y: number) => boolean, on: boolean): Painter =>
  (x, y, n) =>
    mark(x, y)
      ? shade(on ? [255, 220, 60] : [150, 130, 60], 0.95 + n() * 0.1)
      : shade(SLATE, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);

/** Yellow and black stripes round a pad's thin edge. */
const padSide: Painter = (x, y, n) =>
  (x + y) % 4 < 2 ? shade([230, 200, 50], 0.95 + n() * 0.1) : [30, 30, 30, 255];

// A top face's texture x runs west and y south, so each facing's arrow points this way.
const PAD_ARROWS: [string, number, number][] = [
  ["north", 0, -1],
  ["east", -1, 0],
  ["south", 0, 1],
  ["west", 1, 0],
];

/** A big amber digit on dark glass in a metal frame. */
const display =
  (d: number): Painter =>
  (x, y, n) => {
    if (edge(x, y, 2)) return shade(IRON, edge(x, y) ? 0.7 : 0.9 + n() * 0.2);
    return digitPixel(d, x, y) ? [255, 180, 60, 255] : [25, 20, 15, 255];
  };

// Each timer speed's face, by quarter seconds per beat: red is fastest and blue slowest.
const TIMER_FACES: [number, RGB][] = [
  [1, [255, 90, 80]],
  [2, [255, 170, 80]],
  [8, [150, 230, 140]],
  [16, [140, 180, 255]],
];

/** A thin string crossing the block, pale until someone walks through it. */
const tripwire =
  (rgb: RGB): Painter =>
  (x, y, n) =>
    Math.abs(x - 7.5) < 1 || Math.abs(y - 7.5) < 1 ? shade(rgb, 0.92 + n() * 0.16) : CLEAR;

export const DEVICE_PAINTERS: Record<string, Painter> = {
  sticky_piston_front: stickyFront([90, 160, 70]),
  sticky_piston_front_on: stickyFront([150, 230, 110]),
  trapdoor,
  trapdoor_open: trapdoorRim,
  fence_gate: fenceGate(true),
  fence_gate_open: fenceGate(false),
  ...Object.fromEntries(
    PAD_ARROWS.flatMap(([way, dx, dy]) => [
      [`launch_pad_${way}`, pad((x, y) => arrowPixel(dx, dy, x, y), false)],
      [`launch_pad_${way}_on`, pad((x, y) => arrowPixel(dx, dy, x, y), true)],
    ]),
  ),
  launch_pad_up: pad(ringPixel, false),
  launch_pad_up_on: pad(ringPixel, true),
  launch_pad_side: padSide,
  ...Object.fromEntries(Array.from({ length: 10 }, (_, d) => [`display_${d}`, display(d)])),
  ...Object.fromEntries(
    TIMER_FACES.flatMap(([beats, rgb]) => [
      [`timer_${beats}`, timer(dim(rgb))],
      [`timer_${beats}_on`, timer(rgb)],
    ]),
  ),
  tripwire: tripwire([215, 215, 205]),
  tripwire_on: tripwire([255, 200, 90]),
};
