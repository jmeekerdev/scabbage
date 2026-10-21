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

/** Everything that decides how the world is drawn; one file per look in this folder. */
export interface Look {
  name: string;
  /** One line for the picker. */
  about: string;
  surface: {
    /** How much each block's brightness varies, so same-kind neighbors don't form a grid. */
    tint: number;
  };
  light: {
    /** Fixed brightness per face direction. */
    faces: { top: number; bottom: number; x: number; z: number };
    /** How much tucked-in corners darken, 0–1. */
    ao: number;
    lamp: Rgb;
    /** Lamp light kept per metre away from it. */
    lampFade: number;
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
}

export type LookCost = "fastest" | "light" | "heavier";

/** A rough cost for the picker: each feature beyond plain textured blocks adds to it. */
export function lookCost(look: Look): LookCost {
  const points = (look.shape.edges ? 1 : 0) + (look.shape.softCorners > 0 ? 1 : 0);
  return points === 0 ? "fastest" : points === 1 ? "light" : "heavier";
}
