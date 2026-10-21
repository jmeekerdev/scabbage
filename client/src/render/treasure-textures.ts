/** Textures for adventures: chests, the boss altar, gold, gems and checkpoints. */
import type { Painter, RGB } from "./textures";

const shade = ([r, g, b]: RGB, k: number): [number, number, number, number] => [
  Math.min(255, Math.round(r * k)),
  Math.min(255, Math.round(g * k)),
  Math.min(255, Math.round(b * k)),
  255,
];
const WOOD: RGB = [150, 104, 56];
const IRON: RGB = [70, 70, 76];
const GOLD: RGB = [240, 196, 60];
const GEM: RGB = [70, 220, 200];
const COBBLE: RGB = [118, 118, 124];
const edge = (x: number, y: number) => x === 0 || y === 0 || x === 15 || y === 15;

/** Wooden planks bound with iron bands and a gold lock. */
const chestSide: Painter = (x, y, n) => {
  if (x >= 7 && x <= 8 && y >= 8 && y <= 10) return shade(GOLD, 0.95 + n() * 0.1);
  if (edge(x, y) || y === 9) return shade(IRON, 0.9 + n() * 0.2);
  return shade(WOOD, (y % 4 === 0 ? 0.8 : 1) * (0.92 + n() * 0.16));
};

/** A shut lid, or an open one showing the empty dark inside. */
const chestTop =
  (open: boolean): Painter =>
  (x, y, n) => {
    if (edge(x, y)) return shade(IRON, 0.9 + n() * 0.2);
    if (open && x > 1 && y > 1 && x < 14 && y < 14) return shade([40, 28, 18], 0.9 + n() * 0.2);
    return shade(WOOD, (x % 5 === 0 ? 0.8 : 1) * (0.92 + n() * 0.16));
  };

/** Carved stone with a glowing purple rune. */
const altarTop: Painter = (x, y, n) => {
  const r = Math.hypot(x - 7.5, y - 7.5);
  const rune =
    (r > 4 && r < 5.5) || (Math.abs(x - y) < 1 && r < 4) || (Math.abs(x + y - 15) < 1 && r < 4);
  return rune
    ? shade([200, 110, 255], 0.9 + n() * 0.2)
    : shade(COBBLE, edge(x, y) ? 0.6 : 0.8 + n() * 0.3);
};
const altarSide: Painter = (x, y, n) =>
  y >= 13 && x % 3 === 1
    ? shade([200, 110, 255], 0.9)
    : shade(COBBLE, edge(x, y) ? 0.6 : 0.8 + n() * 0.3);

/** Shiny bars with a bright glint along the top-left. */
const gold: Painter = (x, y, n) => {
  const glint = x + (15 - y) < 6;
  return shade(GOLD, (glint ? 1.15 : edge(x, y) ? 0.75 : 1) * (0.95 + n() * 0.1));
};

/** A faceted gem: lighter toward the middle, sparkling in a few spots. */
const gem: Painter = (x, y, n) => {
  const facet = Math.abs(x - 7.5) + Math.abs(y - 7.5);
  if (n(3) > 0.96) return [240, 255, 255, 255];
  return shade(GEM, facet < 4 ? 1.15 : facet < 8 ? 1 : 0.7);
};

/** A red flag on a pole over a white pad. */
const checkpoint: Painter = (x, y, n) => {
  if (x === 5 && y >= 2 && y <= 13) return shade([90, 60, 40], 0.9 + n() * 0.2);
  if (x >= 6 && x <= 11 && y >= 2 && y <= 6 - Math.abs(x - 8.5) / 3)
    return shade([230, 50, 50], 0.9 + n() * 0.2);
  return shade([235, 235, 225], edge(x, y) ? 0.7 : 0.9 + n() * 0.1);
};

export const TREASURE_PAINTERS: Record<string, Painter> = {
  chest_side: chestSide,
  chest_top: chestTop(false),
  chest_open_top: chestTop(true),
  altar_top: altarTop,
  altar_side: altarSide,
  gold_block: gold,
  gem_block: gem,
  checkpoint,
};
