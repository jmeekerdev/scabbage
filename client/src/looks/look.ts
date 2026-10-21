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

/** A light from one direction, shading faces between two colours by how much they face it. */
export interface LookSun {
  /** Toward the light; it needn't be normalised. */
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

/** Everything that decides how the world is drawn; one file per look in this folder. */
export interface Look {
  name: string;
  /** One line for the picker. */
  about: string;
  surface: {
    /** Draws block pictures; without them each block is one flat colour, and custom blocks keep theirs. */
    textured: boolean;
    /** Flat colours by texture name, for pictures whose average colour isn't wanted. */
    palette?: Record<string, Rgb>;
    /** How much each block's brightness varies, so same-kind neighbors don't form a grid. */
    tint: number;
  };
  /** Light colours multiply the surface as they are; only sky, fog and palette colours are sRGB. */
  light: {
    /** Fixed brightness per face direction. */
    faces: { top: number; bottom: number; x: number; z: number };
    /** Light left in the most tucked-in corners: grey darkens, a colour tints. */
    ao: Rgb;
    lamp: Rgb;
    /** Lamp light kept per metre away from it. */
    lampFade: number;
    /** Light from one fixed direction, on top of the face brightness. */
    sun?: LookSun;
  };
  shape: {
    /** Rounds block edges that face open air. */
    edges: boolean;
    /** How far smoothed corners may move, in blocks (at most 0.5). */
    softCorners: number;
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
}

export type LookCost = "fastest" | "light" | "heavier";

/** A rough cost for the picker: each feature beyond plain textured blocks adds to it. */
export function lookCost(look: Look): LookCost {
  const points = (look.shape.edges ? 1 : 0) + (look.shape.softCorners > 0 ? 1 : 0);
  return points === 0 ? "fastest" : points === 1 ? "light" : "heavier";
}
