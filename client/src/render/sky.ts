import type { LookSky, Rgb } from "../looks/look";

/** One full day–night cycle; every client derives the time from its clock, so all agree. */
export const DAY_MS = 20 * 60_000;
export const MIN_LIGHT = 0.0875; // dark enough that lamps and torches matter

export interface SkyState {
  daylight: number; // MIN_LIGHT at night up to 1 by day
  color: Rgb; // sky and fog, sRGB 0–1
  sun: { x: number; y: number }; // direction around the z axis; the moon sits opposite
}

/** The sky at a wall-clock time in a look's colors: sunrise at the start of each cycle, noon a quarter in. */
export function skyAt(nowMs: number, colors: LookSky): SkyState {
  const angle = ((nowMs % DAY_MS) / DAY_MS) * Math.PI * 2;
  const height = Math.sin(angle);
  const day = smoothstep(-0.2, 0.2, height);
  const glow = Math.max(0, 1 - Math.abs(height) / 0.3) ** 2 * 0.7;
  return {
    daylight: MIN_LIGHT + (1 - MIN_LIGHT) * day,
    color: mix(mix(colors.night, colors.day, day), colors.sunset, glow),
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
