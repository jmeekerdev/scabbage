/** One full day–night cycle; every client derives the time from its clock, so all agree. */
export const DAY_MS = 20 * 60_000;
export const MIN_LIGHT = 0.35; // night stays bright enough to keep playing

export interface Rgb {
  r: number;
  g: number;
  b: number;
}

export interface SkyState {
  daylight: number; // MIN_LIGHT at night up to 1 by day
  color: Rgb; // sky and fog, sRGB 0–1
  sun: { x: number; y: number }; // direction around the z axis; the moon sits opposite
}

const DAY: Rgb = { r: 0.53, g: 0.81, b: 0.92 };
const NIGHT: Rgb = { r: 0.04, g: 0.06, b: 0.14 };
const SUNSET: Rgb = { r: 0.98, g: 0.6, b: 0.35 };

/** The sky at a wall-clock time: sunrise at the start of each cycle, noon a quarter in. */
export function skyAt(nowMs: number): SkyState {
  const angle = ((nowMs % DAY_MS) / DAY_MS) * Math.PI * 2;
  const height = Math.sin(angle);
  const day = smoothstep(-0.2, 0.2, height);
  const glow = Math.max(0, 1 - Math.abs(height) / 0.3) ** 2 * 0.7;
  return {
    daylight: MIN_LIGHT + (1 - MIN_LIGHT) * day,
    color: mix(mix(NIGHT, DAY, day), SUNSET, glow),
    sun: { x: Math.cos(angle), y: height },
  };
}

function smoothstep(edge0: number, edge1: number, x: number): number {
  const t = Math.min(1, Math.max(0, (x - edge0) / (edge1 - edge0)));
  return t * t * (3 - 2 * t);
}

function mix(a: Rgb, b: Rgb, t: number): Rgb {
  return { r: a.r + (b.r - a.r) * t, g: a.g + (b.g - a.g) * t, b: a.b + (b.b - a.b) * t };
}
