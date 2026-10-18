import { describe, expect, it } from "vitest";
import { memoryStore } from "../join/identity";
import {
  defaultSettings,
  FrameLimiter,
  loadSettings,
  pixelRatioFor,
  qualityFor,
  sanitize,
  saveSettings,
  VIEW_DISTANCE,
} from "./settings";

describe("settings", () => {
  it("defaults to lighter settings on touch devices", () => {
    const phone = defaultSettings(true);
    const desktop = defaultSettings(false);
    expect(phone.viewDistance).toBeLessThan(desktop.viewDistance);
    expect(phone.fpsCap).toBe(30);
    expect(desktop.fpsCap).toBe(0);
  });

  it("round-trips through the store", () => {
    const store = memoryStore();
    const s = { viewDistance: 5, preset: "low", fpsCap: 60 } as const;
    saveSettings(store, s);
    expect(loadSettings(store, defaultSettings(false))).toEqual(s);
  });

  it("falls back to defaults for missing or corrupt data", () => {
    const store = memoryStore();
    const d = defaultSettings(false);
    expect(loadSettings(store, d)).toEqual(d);
    store.setItem("scabbage.settings", "{not json");
    expect(loadSettings(store, d)).toEqual(d);
  });

  it("repairs each bad field independently", () => {
    const d = defaultSettings(false);
    expect(sanitize({ viewDistance: 999, preset: "ultra", fpsCap: 45 }, d)).toEqual({
      ...d,
      viewDistance: VIEW_DISTANCE.max,
    });
    expect(sanitize({ viewDistance: 3.7, preset: "low" }, d)).toEqual({
      ...d,
      viewDistance: 4,
      preset: "low",
    });
    expect(sanitize("nope", d)).toEqual(d);
  });
});

describe("graphics presets", () => {
  it("scale resolution and antialiasing with the preset", () => {
    expect(qualityFor("low").antialias).toBe(false);
    expect(qualityFor("high").antialias).toBe(true);
    expect(qualityFor("low").maxPixelRatio).toBeLessThan(qualityFor("high").maxPixelRatio);
  });

  it("never renders above the device's own pixel ratio", () => {
    expect(pixelRatioFor("high", 1)).toBe(1);
    expect(pixelRatioFor("high", 3)).toBe(2);
    expect(pixelRatioFor("low", 3)).toBeLessThan(1);
  });
});

describe("FrameLimiter", () => {
  const frames = (limiter: FrameLimiter, hz: number, seconds: number) => {
    let drawn = 0;
    for (let i = 0; i < hz * seconds; i++) if (limiter.ready((i * 1000) / hz)) drawn++;
    return drawn;
  };

  it("draws every frame when uncapped", () => {
    expect(frames(new FrameLimiter(0), 144, 1)).toBe(144);
  });

  it("halves a 60 Hz display to 30 fps", () => {
    expect(frames(new FrameLimiter(30), 60, 1)).toBe(30);
  });

  it("holds close to the cap on high-refresh displays", () => {
    expect(frames(new FrameLimiter(60), 144, 2)).toBeGreaterThanOrEqual(118);
    expect(frames(new FrameLimiter(60), 144, 2)).toBeLessThanOrEqual(121);
  });

  it("does not burst after a pause", () => {
    const limiter = new FrameLimiter(30);
    limiter.ready(0);
    expect(limiter.ready(5000)).toBe(true);
    expect(limiter.ready(5016)).toBe(false);
  });
});
