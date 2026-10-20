import { describe, expect, it } from "vitest";
import { AutoQuality, lighter } from "./auto-quality";
import { defaultSettings, VIEW_DISTANCE, type Settings } from "./settings";

const high = defaultSettings("high", false);

/** Feeds frames at fps for seconds and returns the first suggestion, if any. */
function run(auto: AutoQuality, s: Settings, fps: number, seconds: number, startMs = 0) {
  for (let i = 0; i <= fps * seconds; i++) {
    const next = auto.frame(startMs + (i * 1000) / fps, s);
    if (next) return next;
  }
  return undefined;
}

describe("lighter", () => {
  it("lowers the preset before the view distance", () => {
    expect(lighter(high)).toEqual({ ...high, preset: "medium" });
    const low = { ...high, preset: "low" as const };
    expect(lighter(low)).toEqual({ ...low, viewDistance: high.viewDistance - 1 });
  });

  it("stops above the shortest view distance", () => {
    const floor = { ...high, preset: "low" as const, viewDistance: VIEW_DISTANCE.min + 1 };
    expect(lighter(floor)).toBeUndefined();
  });
});

describe("AutoQuality", () => {
  it("leaves smooth games alone", () => {
    expect(run(new AutoQuality(), high, 60, 30)).toBeUndefined();
  });

  it("lowers settings when frames fall well behind", () => {
    expect(run(new AutoQuality(), high, 20, 30)).toEqual(lighter(high));
  });

  it("only judges frames drawn back to back", () => {
    const auto = new AutoQuality();
    for (let i = 0; i <= 20 * 30; i++) {
      const next = auto.frame((i * 1000) / 20, high, i % 3 === 0);
      expect(next).toBeUndefined();
    }
  });

  it("ignores the slow first seconds while the world loads", () => {
    expect(run(new AutoQuality(), high, 20, 9)).toBeUndefined();
  });

  it("measures against the frame-rate cap", () => {
    const capped = { ...high, fpsCap: 30 as const };
    expect(run(new AutoQuality(), capped, 30, 30)).toBeUndefined();
  });

  it("does nothing once the player picks settings themselves", () => {
    expect(run(new AutoQuality(), { ...high, auto: false }, 10, 30)).toBeUndefined();
  });

  it("does not count paused tabs as slow", () => {
    const auto = new AutoQuality();
    run(auto, high, 60, 12);
    expect(run(auto, high, 60, 6, 14_000)).toBeUndefined();
  });
});
