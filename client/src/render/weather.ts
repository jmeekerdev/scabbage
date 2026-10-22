import { METRE } from "../world/constants";
import { MIN_LIGHT, type SkyState } from "./sky";

/** What the sky is doing; numbered as in the server's weather package. */
export enum Weather {
  CLEAR = 0,
  RAIN = 1, // snow, high up
  STORM = 2, // heavy rain or snow with lightning
}

/** How long each weather lasts; server/internal/weather picks the same, so everyone agrees. */
export const SPELL_MS = 4 * 60_000;
/** How long one weather takes to give way to the next. */
export const FADE_MS = 20_000;
/** Above this height rain falls as snow. */
export const SNOW_LINE = 56 * METRE;

const FLASH_SLOT_MS = 5000;
const FLASH_MS = 200;

export interface WeatherState {
  kind: Weather;
  wet: number; // 0 dry to 1 falling hard
  storm: number; // 0 calm to 1 stormy
}

/** The weather at a wall-clock time, fading from the spell before over FADE_MS. */
export function weatherAt(nowMs: number): WeatherState {
  const spell = Math.floor(nowMs / SPELL_MS);
  const kind = kindOf(spell);
  const was = kindOf(spell - 1);
  const t = Math.min(1, (nowMs - spell * SPELL_MS) / FADE_MS);
  const blend = (f: (k: Weather) => number) => f(was) + (f(kind) - f(was)) * t;
  return {
    kind,
    wet: blend((k) => (k === Weather.CLEAR ? 0 : 1)),
    storm: blend((k) => (k === Weather.STORM ? 1 : 0)),
  };
}

export function snowsAt(y: number): boolean {
  return y >= SNOW_LINE;
}

/** How bright a lightning flash is at a time, 0 most of the time; the same for everyone. */
export function flashAt(nowMs: number): number {
  const storm = weatherAt(nowMs).storm;
  if (storm === 0) return 0;
  const slot = Math.floor(nowMs / FLASH_SLOT_MS);
  const roll = mix(slot);
  if (roll % 4 !== 0) return 0;
  const since = nowMs - slot * FLASH_SLOT_MS - ((roll >>> 8) % (FLASH_SLOT_MS - FLASH_MS));
  return since >= 0 && since < FLASH_MS ? (1 - since / FLASH_MS) * storm : 0;
}

/** The sky darkened and greyed by rain, and lit up by a lightning flash from 0 to 1. */
export function weatherSky(sky: SkyState, w: WeatherState, flash: number): SkyState {
  if (w.wet === 0 && flash === 0) return sky;
  const dim = 1 - 0.3 * w.wet - 0.25 * w.storm;
  const lit = MIN_LIGHT + (sky.daylight - MIN_LIGHT) * dim;
  const { r, g, b } = sky.color;
  const grey = (0.3 * r + 0.55 * g + 0.15 * b) * dim;
  const cloud = 0.7 * w.wet;
  const tone = (c: number) => c + (grey - c) * cloud + (1 - c) * 0.6 * flash;
  return {
    ...sky,
    daylight: lit + (1 - lit) * flash,
    color: { r: tone(r), g: tone(g), b: tone(b) },
  };
}

function kindOf(spell: number): Weather {
  const roll = mix(spell) % 100;
  if (roll < 50) return Weather.CLEAR;
  return roll < 85 ? Weather.RAIN : Weather.STORM;
}

/** Scrambles a number into an even spread, exactly as weather.go's mix does. */
function mix(n: number): number {
  let h = Math.imul(n, 0x9e3779b1);
  h ^= h >>> 16;
  h = Math.imul(h, 0x85ebca6b);
  h ^= h >>> 13;
  h = Math.imul(h, 0xc2b2ae35);
  return (h ^ (h >>> 16)) >>> 0;
}
