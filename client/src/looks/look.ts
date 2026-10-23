import { afterFeatures } from "../render/after-pass";

/** A color in sRGB, each channel 0–1. */
export interface Rgb {
  r: number;
  g: number;
  b: number;
}

/** How a look colors the sky and fog through the day; the fog takes the sky's color. */
export interface LookSky {
  day: Rgb;
  night: Rgb;
  sunset: Rgb;
  sun: Rgb;
  moon: Rgb;
}

/** A light from one direction, shading faces between two colors by how much they face it. */
export interface LookSun {
  /** Toward the light; it needn't be normalized. */
  dir: { x: number; y: number; z: number };
  lit: Rgb;
  shade: Rgb;
  /** How far light wraps past the side facing away, 0–1. */
  wrap: number;
  /** Steps between shade and lit; 0 for a smooth blend. */
  bands: number;
  /** Strength of a soft highlight; 0 for none. */
  shine: number;
}

/** Screen effects drawn over the finished frame; a look without any skips the pass entirely. */
export interface LookAfter {
  /** Lines where depth jumps, and with creases also where surfaces fold. */
  outline?: {
    /** Line width in pixels at 750 pixels tall. */
    width: number;
    /** 0–1, how much the line covers what is under it. */
    strength: number;
    /** The line color; without one, lines darken what is under them. */
    color?: Rgb;
    creases?: boolean;
  };
  /** A light blur over the whole frame, its radius in pixels at 700 tall. */
  soften?: number;
  /** Paper grain, blots and fibers, 0–1. */
  paper?: number;
  /** Blur away from the middle band, only in the overhead map view. */
  tiltShift?: number;
  /** Darkening toward the corners, 0–1. */
  vignette?: number;
  /** 1 leaves colors as they are, lower grays them, higher strengthens them. */
  saturation?: number;
}

/** Everything that decides how the world is drawn; one file per look in this folder. */
export interface Look {
  name: string;
  /** One line for the picker. */
  about: string;
  surface: {
    /** Draws block pictures; without them each block is one flat color, and custom blocks keep theirs. */
    textured: boolean;
    /** Flat colors by texture name, for pictures whose average color isn't wanted. */
    palette?: Record<string, Rgb>;
    /** Textures drawn without their see-through holes when untextured, like leaves as solid crowns. */
    solid?: string[];
    /** 0–1, how much of each block's picture shows as light and dark over its flat color. */
    detail?: number;
    /** How much each block's brightness varies, so same-kind neighbors don't form a grid. */
    tint: number;
    /** 0–1, color that drifts across the land in soft washes, fixed to the world. */
    wash?: number;
    /** 0–1, fine fabric fuzz, fixed to the world. */
    fiber?: number;
  };
  /** Light colors multiply the surface as they are; only sky, fog and palette colors are sRGB. */
  light: {
    /** Fixed brightness per face direction. */
    faces: { top: number; bottom: number; x: number; z: number };
    /** Light left in the most tucked-in corners: gray darkens, a color tints. */
    ao: Rgb;
    lamp: Rgb;
    /** Lamp light kept one meter away; farther, it falls with the square of the distance. */
    lampFade: number;
    /** Light from one fixed direction, on top of the face brightness. */
    sun?: LookSun;
    /** Light from 1 − lift at the low height to 1 + lift at the high one, heights in meters. */
    height?: { low: number; high: number; lift: number };
  };
  shape: {
    /** Rounds block edges that face open air. */
    edges: boolean;
    /** How far smoothed corners may move, in blocks (at most 0.5). */
    softCorners: number;
    /** 0–1, how much color darkens toward open edges, like pooled paint; needs edges. */
    pool?: number;
    /** sRGB color of running stitches along open edges; needs edges. */
    stitch?: Rgb;
  };
  sky: LookSky;
  fog: {
    /** Where fog begins, as a fraction of the view distance. */
    start: number;
  };
  water: {
    /** Multiplies the picture's opacity, or is the opacity itself in untextured looks. */
    opacity: number;
  };
  after?: LookAfter;
}

export type LookCost = "fastest" | "light" | "heavier";

/** A rough cost for the picker: each feature beyond plain textured blocks adds to it. */
export function lookCost(look: Look): LookCost {
  const after = afterFeatures(look, true);
  const points =
    (look.shape.edges ? 1 : 0) +
    (look.shape.softCorners > 0 ? 1 : 0) +
    (after && !("OVERLAY" in after) ? 1 : 0);
  return points === 0 ? "fastest" : points === 1 ? "light" : "heavier";
}
