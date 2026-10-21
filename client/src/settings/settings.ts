import type { KeyValueStore } from "../join/identity";

export const PRESETS = ["low", "medium", "high"] as const;
export type Preset = (typeof PRESETS)[number];

/** Frame-rate caps offered to players; 0 means "match the display". */
export const FPS_CAPS = [30, 60, 0] as const;
export type FpsCap = (typeof FPS_CAPS)[number];

/** View distance in chunks, measured from the player. */
export const VIEW_DISTANCE = { min: 2, max: 12 } as const;

export interface Settings {
  viewDistance: number;
  preset: Preset;
  fpsCap: FpsCap;
  sound: boolean;
  music: boolean;
  /** Lighten graphics automatically when frames fall behind; off once the player picks. */
  auto: boolean;
}

const KEY = "scabbage.settings";

const VIEW_DISTANCE_FOR: Record<Preset, number> = { low: 4, medium: 6, high: 9 };

/** Starting settings for a device tier; touch devices cap frames to save battery and heat. */
export function defaultSettings(tier: Preset, touch: boolean): Settings {
  return {
    viewDistance: VIEW_DISTANCE_FOR[tier],
    preset: tier,
    fpsCap: touch || tier === "low" ? 30 : 0,
    sound: true,
    music: false,
    auto: true,
  };
}

export function loadSettings(store: KeyValueStore, defaults: Settings): Settings {
  try {
    return sanitize(JSON.parse(store.getItem(KEY) ?? "null"), defaults);
  } catch {
    return defaults;
  }
}

export function saveSettings(store: KeyValueStore, s: Settings): void {
  store.setItem(KEY, JSON.stringify(s));
}

/** Keeps each valid field of untrusted saved data and replaces the rest with defaults. */
export function sanitize(raw: unknown, d: Settings): Settings {
  const r = (typeof raw === "object" && raw) || {};
  const pick = <K extends keyof Settings>(k: K) => (r as Partial<Settings>)[k];
  const vd = pick("viewDistance");
  const flag = (k: "sound" | "music" | "auto") => {
    const v = pick(k);
    return typeof v === "boolean" ? v : d[k];
  };
  return {
    viewDistance:
      typeof vd === "number" && Number.isFinite(vd)
        ? Math.min(VIEW_DISTANCE.max, Math.max(VIEW_DISTANCE.min, Math.round(vd)))
        : d.viewDistance,
    preset: PRESETS.find((p) => p === pick("preset")) ?? d.preset,
    fpsCap: FPS_CAPS.find((f) => f === pick("fpsCap")) ?? d.fpsCap,
    sound: flag("sound"),
    music: flag("music"),
    auto: flag("auto"),
  };
}

export interface Quality {
  maxPixelRatio: number;
  antialias: boolean;
}

const QUALITY: Record<Preset, Quality> = {
  low: { maxPixelRatio: 0.75, antialias: false },
  medium: { maxPixelRatio: 1, antialias: false },
  high: { maxPixelRatio: 1.5, antialias: true }, // 1.5x is 56% of the pixels of 2x
};

export function qualityFor(preset: Preset): Quality {
  return QUALITY[preset];
}

/** Render resolution scale: the preset's limit, never sharper than the screen itself. */
export function pixelRatioFor(preset: Preset, devicePixelRatio: number): number {
  return Math.min(devicePixelRatio, QUALITY[preset].maxPixelRatio);
}

/** Skips animation frames to hold a frame-rate cap without drifting below it. */
export class FrameLimiter {
  private next = -Infinity;

  constructor(public fps: number) {}

  ready(nowMs: number): boolean {
    if (this.fps <= 0) return true;
    const interval = 1000 / this.fps;
    // Small slack absorbs rAF timestamp jitter so a 60 Hz display reliably yields 30 fps.
    if (nowMs < this.next - 2) return false;
    this.next = nowMs - this.next > interval ? nowMs + interval : this.next + interval;
    return true;
  }
}
