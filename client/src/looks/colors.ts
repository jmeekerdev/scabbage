import type { Rgb } from "./look";

/** A six-digit sRGB hex colour, as written in the look book. */
export function hex(h: string): Rgb {
  const n = parseInt(h, 16);
  return { r: (n >> 16) / 255, g: ((n >> 8) & 255) / 255, b: (n & 255) / 255 };
}

/** The look book's palette slots, each a hex colour. */
export interface BookColors {
  grass: string;
  dirt: string;
  stone?: string;
  sand?: string;
  water?: string;
  log?: string;
  leaves?: string;
  pine?: string;
  planks?: string;
  brick?: string;
  glass?: string;
  redFlower?: string;
  yellowFlower?: string;
  gravel?: string;
}

// Which textures each slot colours.
const TEXTURES: Record<keyof BookColors, string[]> = {
  grass: ["grass_top"],
  dirt: ["dirt"],
  stone: ["stone"],
  sand: ["sand"],
  water: ["water"],
  log: ["log_top", "log_side"],
  leaves: ["leaves"],
  pine: ["pine_leaves"],
  planks: ["planks", "door", "door_open"],
  brick: ["brick"],
  glass: ["glass"],
  redFlower: ["red_flower", "red_flower_top"],
  yellowFlower: ["yellow_flower", "yellow_flower_top"],
  gravel: ["gravel"],
};

const mix = (a: Rgb, b: Rgb, t: number): Rgb => ({
  r: a.r + (b.r - a.r) * t,
  g: a.g + (b.g - a.g) * t,
  b: a.b + (b.b - a.b) * t,
});

/** A look's surface palette by texture name, from the look book's slots. */
export function blockPalette(colors: BookColors): Record<string, Rgb> {
  const out: Record<string, Rgb> = {};
  for (const [slot, h] of Object.entries(colors) as [keyof BookColors, string][])
    for (const name of TEXTURES[slot]) out[name] = hex(h);
  out.grass_side = mix(hex(colors.dirt), hex(colors.grass), 0.3);
  return out;
}
