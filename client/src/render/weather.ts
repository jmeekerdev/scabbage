import { METER } from "../world/constants";
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
export const SNOW_LINE = 56 * METER;

const FLASH_SLOT_MS = 5000;
const FLASH_MS = 200;

export interface WeatherState {
  kind: Weather;
  wet: number; // 0 dry to 1 falling hard
  storm: number; // 0 calm to 1 stormy
}

/** Weather an admin holds from sinceMs until untilMs (Infinity for good). */
export interface Held {
  kind: Weather;
  sinceMs: number;
  untilMs: number;
}

/** The weather at a time: held weather while it lasts, fading in and out over FADE_MS. */
export function weatherAt(nowMs: number, held?: Held): WeatherState {
  if (!held || nowMs < held.sinceMs) return clockWeather(nowMs);
  const start = (ms: number) => Math.min(1, (ms - held.sinceMs) / FADE_MS);
  if (nowMs < held.untilMs) return blend(clockWeather(held.sinceMs), full(held.kind), start(nowMs));
  const left = blend(clockWeather(held.sinceMs), full(held.kind), start(held.untilMs));
  return blend(left, clockWeather(nowMs), Math.min(1, (nowMs - held.untilMs) / FADE_MS));
}

/** The wall clock's weather, fading from the spell before over FADE_MS. */
function clockWeather(nowMs: number): WeatherState {
  const spell = Math.floor(nowMs / SPELL_MS);
  const t = Math.min(1, (nowMs - spell * SPELL_MS) / FADE_MS);
  return blend(full(kindOf(spell - 1)), full(kindOf(spell)), t);
}

const full = (kind: Weather): WeatherState => ({
  kind,
  wet: kind === Weather.CLEAR ? 0 : 1,
  storm: kind === Weather.STORM ? 1 : 0,
});

/** Part way from a to b, taking b's kind. */
function blend(a: WeatherState, b: WeatherState, t: number): WeatherState {
  return {
    kind: b.kind,
    wet: a.wet + (b.wet - a.wet) * t,
    storm: a.storm + (b.storm - a.storm) * t,
  };
}

export function snowsAt(y: number): boolean {
  return y >= SNOW_LINE;
}

/** How bright a lightning flash is at a time, 0 most of the time; the same for everyone. */
export function flashAt(nowMs: number, held?: Held): number {
  const storm = weatherAt(nowMs, held).storm;
  if (storm === 0) return 0;
  const slot = Math.floor(nowMs / FLASH_SLOT_MS);
  const roll = mix(slot);
  if (roll % 4 !== 0) return 0;
  const since = nowMs - slot * FLASH_SLOT_MS - ((roll >>> 8) % (FLASH_SLOT_MS - FLASH_MS));
  return since >= 0 && since < FLASH_MS ? (1 - since / FLASH_MS) * storm : 0;
}

/** The sky darkened and grayed by rain, and lit up by a lightning flash from 0 to 1. */
export function weatherSky(sky: SkyState, w: WeatherState, flash: number): SkyState {
  if (w.wet === 0 && flash === 0) return sky;
  const dim = 1 - 0.3 * w.wet - 0.25 * w.storm;
  const lit = MIN_LIGHT + (sky.daylight - MIN_LIGHT) * dim;
  const { r, g, b } = sky.color;
  const gray = (0.3 * r + 0.55 * g + 0.15 * b) * dim;
  const cloud = 0.7 * w.wet;
  const tone = (c: number) => c + (gray - c) * cloud + (1 - c) * 0.6 * flash;
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
