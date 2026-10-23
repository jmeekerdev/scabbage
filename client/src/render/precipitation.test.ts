import { Scene, type ShaderMaterial } from "three";
import { describe, expect, it } from "vitest";
import { glslErrors } from "./glsl-check";
import { Precipitation, ROOF_SPAN } from "./precipitation";
import { SNOW_LINE, Weather } from "./weather";

const rain = { kind: Weather.RAIN, wet: 1, storm: 0 };
const dry = { kind: Weather.CLEAR, wet: 0, storm: 0 };
const ground = { x: 10, y: 120, z: -4 };
const open = { topAt: () => -1, version: 0 };

function view() {
  const scene = new Scene();
  const p = new Precipitation(scene, "medium");
  const [lines, points] = scene.children as [Scene, Scene];
  return { p, lines, points };
}

describe("Precipitation", () => {
  it("shows nothing under a clear sky", () => {
    const { p, lines, points } = view();
    p.update(0, ground, dry, open);
    expect(lines.visible || points.visible).toBe(false);
  });

  it("keeps raining under a roof, giving the shader each column's roof to stop drops at", () => {
    const { p, lines } = view();
    const roofed = {
      topAt: (x: number, z: number) => (x === 10 && z === -4 ? 130 : 5),
      version: 0,
    };
    p.update(0, ground, rain, roofed);
    expect(lines.visible).toBe(true);
    const heights = roofHeights(lines);
    const mid = ROOF_SPAN / 2;
    expect(heights[mid * ROOF_SPAN + mid]).toBe(131); // the eye's column, at the middle
    expect(heights[0]).toBe(6);
  });

  it("looks at the roofs again only when the world changes or the eye moves a block", () => {
    const { p, lines } = view();
    let top = 5;
    const world = { topAt: () => top, version: 1 };
    p.update(0, ground, rain, world);
    top = 9;
    p.update(16, { ...ground, x: ground.x + 0.2 }, rain, world);
    expect(roofHeights(lines)[0]).toBe(6);
    world.version++;
    p.update(32, ground, rain, world);
    expect(roofHeights(lines)[0]).toBe(10);
  });

  it("draws rain low down and snow high up", () => {
    const { p, lines, points } = view();
    p.update(0, ground, rain, open);
    expect([lines.visible, points.visible]).toEqual([true, false]);
    p.update(0, { ...ground, y: SNOW_LINE + 1 }, rain, open);
    expect([lines.visible, points.visible]).toEqual([false, true]);
  });

  it("moves the drops as they fall, so frames get drawn", () => {
    const { p, lines } = view();
    p.update(0, ground, rain, open);
    const before = lines.position.y;
    p.update(100, ground, rain, open);
    expect(lines.position.y).not.toBe(before);
  });

  it("draws more drops in storms and fewer on the low preset", () => {
    const { p, lines } = view();
    const drawn = () =>
      (lines as unknown as { geometry: { drawRange: { count: number } } }).geometry.drawRange.count;
    p.update(0, ground, rain, open);
    const steady = drawn();
    p.update(0, ground, { kind: Weather.STORM, wet: 1, storm: 1 }, open);
    expect(drawn()).toBeGreaterThan(steady);
    p.setPreset("low");
    p.update(0, ground, rain, open);
    expect(drawn()).toBeLessThan(steady);
  });

  it("compiles its rain and snow shaders", () => {
    const { lines, points } = view();
    const shader = (o: object) => (o as { material: ShaderMaterial }).material;
    expect([...glslErrors(shader(lines)), ...glslErrors(shader(points))]).toEqual([]);
  });
});

function roofHeights(o: object): Float32Array {
  const m = (o as { material: ShaderMaterial }).material;
  return m.uniforms.roofs.value.image.data as Float32Array;
}
