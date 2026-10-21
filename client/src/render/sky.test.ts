import { describe, expect, it } from "vitest";
import { lookById } from "../looks/registry";
import { DAY_MS, MIN_LIGHT, skyAt } from "./sky";

const TODAY = lookById("today").sky;

const at = (fraction: number) => skyAt(fraction * DAY_MS, TODAY);

describe("skyAt", () => {
  it("is fully bright and blue at noon", () => {
    const noon = at(0.25);
    expect(noon.daylight).toBe(1);
    expect(noon.color.b).toBeGreaterThan(noon.color.r);
    expect(noon.sun.y).toBeCloseTo(1);
  });

  it("is dim but never black at midnight", () => {
    const night = at(0.75);
    expect(night.daylight).toBe(MIN_LIGHT);
    expect(night.color.b).toBeLessThan(0.4);
    expect(night.sun.y).toBeCloseTo(-1);
  });

  it("glows warm around sunrise", () => {
    const dawn = at(0.01);
    expect(dawn.color.r).toBeGreaterThan(dawn.color.b);
  });

  it("changes smoothly and repeats every day", () => {
    const range = 1 - MIN_LIGHT;
    for (let f = 0; f < 1; f += 0.001) {
      expect(Math.abs(at(f + 0.001).daylight - at(f).daylight) / range).toBeLessThan(0.03);
    }
    expect(at(1.3).daylight).toBeCloseTo(at(0.3).daylight);
  });

  it("takes its colors from the look", () => {
    const red = { ...TODAY, day: { r: 1, g: 0, b: 0 } };
    expect(skyAt(0.25 * DAY_MS, red).color).toEqual({ r: 1, g: 0, b: 0 });
  });
});
