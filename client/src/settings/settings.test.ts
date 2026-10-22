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
  viewMetres,
} from "./settings";

describe("settings", () => {
  it("defaults to lighter settings on weaker devices", () => {
    const phone = defaultSettings("medium", true);
    const desktop = defaultSettings("high", false);
    expect(defaultSettings("low", false).viewDistance).toBeLessThan(phone.viewDistance);
    expect(phone.viewDistance).toBeLessThan(desktop.viewDistance);
    expect(phone.preset).toBe("medium");
    expect(desktop.auto).toBe(true);
  });

  it("caps the frame rate on touch and low-end devices", () => {
    expect(defaultSettings("medium", true).fpsCap).toBe(30);
    expect(defaultSettings("low", false).fpsCap).toBe(30);
    expect(defaultSettings("medium", false).fpsCap).toBe(0);
  });

  it("round-trips through the store", () => {
    const store = memoryStore();
    const s = {
      viewDistance: 5,
      preset: "low",
      fpsCap: 60,
      sound: false,
      music: true,
      auto: false,
      look: "today",
      buttons: true,
    } as const;
    saveSettings(store, s);
    expect(loadSettings(store, defaultSettings("high", false))).toEqual(s);
  });

  it("falls back to defaults for missing or corrupt data", () => {
    const store = memoryStore();
    const d = defaultSettings("high", false);
    expect(loadSettings(store, d)).toEqual(d);
    store.setItem("scabbage.settings", "{not json");
    expect(loadSettings(store, d)).toEqual(d);
  });

  it("repairs each bad field independently", () => {
    const d = defaultSettings("high", false);
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

  it("keeps sound on and music off unless chosen otherwise", () => {
    const d = defaultSettings("medium", true);
    expect(d.sound).toBe(true);
    expect(d.music).toBe(false);
    expect(sanitize({ sound: "yes", music: 1 }, d)).toEqual(d);
    expect(sanitize({ sound: false, music: true }, d)).toEqual({ ...d, sound: false, music: true });
    expect(sanitize({ auto: false }, d)).toEqual({ ...d, auto: false });
  });

  it("shows on-screen buttons only on touch screens unless chosen", () => {
    const d = defaultSettings("medium", false);
    expect(d.buttons).toBe(false);
    expect(defaultSettings("medium", true).buttons).toBe(true);
    expect(sanitize({ buttons: "yes" }, d)).toEqual(d);
    expect(sanitize({ buttons: true }, d)).toEqual({ ...d, buttons: true });
  });

  it("starts with the default look and forgets looks that no longer exist", () => {
    const d = defaultSettings("medium", false);
    expect(d.look).toBe("today");
    expect(sanitize({ look: "gone" }, d)).toEqual(d);
    expect(sanitize({ look: 7 }, d)).toEqual(d);
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
    expect(pixelRatioFor("high", 3)).toBe(1.5); // antialiasing covers the rest, at well under 2x the pixels
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

describe("viewMetres", () => {
  it("turns a view distance in chunks into whole metres", () => {
    expect(viewMetres(6)).toBe(64);
    expect(viewMetres(VIEW_DISTANCE.max)).toBe(128);
  });
});
