import { Scene, type ShaderMaterial } from "three";
import { describe, expect, it } from "vitest";
import { glslErrors } from "./glsl-check";
import { Precipitation, sheltered } from "./precipitation";
import { SNOW_LINE, Weather } from "./weather";

const rain = { kind: Weather.RAIN, wet: 1, storm: 0 };
const dry = { kind: Weather.CLEAR, wet: 0, storm: 0 };
const ground = { x: 10, y: 120, z: -4 };

function view() {
  const scene = new Scene();
  const p = new Precipitation(scene, "medium");
  const [lines, points] = scene.children as [Scene, Scene];
  return { p, lines, points };
}

describe("Precipitation", () => {
  it("shows nothing under a clear sky or a roof", () => {
    const { p, lines, points } = view();
    p.update(0, ground, dry, false);
    expect(lines.visible || points.visible).toBe(false);
    p.update(0, ground, rain, true);
    expect(lines.visible || points.visible).toBe(false);
  });

  it("draws rain low down and snow high up", () => {
    const { p, lines, points } = view();
    p.update(0, ground, rain, false);
    expect([lines.visible, points.visible]).toEqual([true, false]);
    p.update(0, { ...ground, y: SNOW_LINE + 1 }, rain, false);
    expect([lines.visible, points.visible]).toEqual([false, true]);
  });

  it("moves the drops as they fall, so frames get drawn", () => {
    const { p, lines } = view();
    p.update(0, ground, rain, false);
    const before = lines.position.y;
    p.update(100, ground, rain, false);
    expect(lines.position.y).not.toBe(before);
  });

  it("draws more drops in storms and fewer on the low preset", () => {
    const { p, lines } = view();
    const drawn = () =>
      (lines as unknown as { geometry: { drawRange: { count: number } } }).geometry.drawRange.count;
    p.update(0, ground, rain, false);
    const steady = drawn();
    p.update(0, ground, { kind: Weather.STORM, wet: 1, storm: 1 }, false);
    expect(drawn()).toBeGreaterThan(steady);
    p.setPreset("low");
    p.update(0, ground, rain, false);
    expect(drawn()).toBeLessThan(steady);
  });

  it("compiles its rain and snow shaders", () => {
    const { lines, points } = view();
    const shader = (o: object) => (o as { material: ShaderMaterial }).material;
    expect([...glslErrors(shader(lines)), ...glslErrors(shader(points))]).toEqual([]);
  });
});

describe("sheltered", () => {
  it("is true under any block overhead, near enough, and false in the open or unloaded sky", () => {
    const roofAt = (h: number) => (_x: number, y: number) => (y === h ? 1 : 0);
    expect(sheltered(roofAt(130), ground)).toBe(true);
    expect(sheltered(roofAt(300), ground)).toBe(false);
    expect(sheltered(() => undefined, ground)).toBe(false);
  });
});
