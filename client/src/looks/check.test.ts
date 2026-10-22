import { describe, expect, it } from "vitest";
import { checkLook } from "./check";
import type { Look } from "./look";
import { LOOKS, lookById } from "./registry";

const today = lookById("today");
const withSun = (sun: Partial<NonNullable<Look["light"]["sun"]>>): Look => ({
  ...today,
  light: {
    ...today.light,
    sun: {
      dir: { x: 0, y: 1, z: 0 },
      lit: { r: 1, g: 1, b: 1 },
      shade: { r: 0.5, g: 0.5, b: 0.5 },
      wrap: 0,
      bands: 0,
      shine: 0,
      ...sun,
    },
  },
});

describe("checkLook", () => {
  it("passes every look in the registry", () => {
    expect(LOOKS.size).toBeGreaterThan(1);
    for (const [id, look] of LOOKS) expect([id, checkLook(look)]).toEqual([id, []]);
  });

  it("needs a name and a line for the picker", () => {
    expect(checkLook({ ...today, name: "", about: " " })).toEqual([
      "name is empty",
      "about is empty",
    ]);
  });

  it("keeps sRGB colours within 0–1 but lets light colours go brighter", () => {
    const sky = { ...today.sky, day: { r: 1.2, g: 0, b: 0 } };
    expect(checkLook({ ...today, sky })).toEqual(["sky.day.r must be 0–1, is 1.2"]);
    expect(checkLook(withSun({ lit: { r: 1.3, g: 1, b: 1 } }))).toEqual([]);
    expect(checkLook(withSun({ lit: { r: -0.1, g: 1, b: 1 } }))).toEqual([
      "light.sun.lit.r must be 0–4, is -0.1",
    ]);
  });

  it("checks fractions and the soft corner limit", () => {
    const look: Look = {
      ...today,
      fog: { start: 1.5 },
      shape: { ...today.shape, softCorners: 0.6 },
      after: { vignette: -1 },
    };
    expect(checkLook(look)).toEqual([
      "shape.softCorners must be 0–0.5, is 0.6",
      "fog.start must be 0–1, is 1.5",
      "after.vignette must be 0–1, is -1",
    ]);
  });

  it("catches numbers that aren't numbers", () => {
    expect(checkLook({ ...today, water: { opacity: NaN } })).toEqual([
      "water.opacity must be 0–1, is NaN",
    ]);
  });

  it("wants bands of none or at least two, and a sun that points somewhere", () => {
    expect(checkLook(withSun({ bands: 1 }))).toEqual(["light.sun.bands must be 0 or a whole 2+"]);
    expect(checkLook(withSun({ bands: 2.5 }))).toEqual(["light.sun.bands must be 0 or a whole 2+"]);
    expect(checkLook(withSun({ dir: { x: 0, y: 0, z: 0 } }))).toEqual([
      "light.sun.dir points nowhere",
    ]);
  });

  it("wants a height gradient to rise", () => {
    const height = { low: 50, high: 40, lift: 0.1 };
    expect(checkLook({ ...today, light: { ...today.light, height } })).toEqual([
      "light.height.high must be above light.height.low",
    ]);
  });

  it("names palette and solid textures that don't exist", () => {
    const surface = {
      ...today.surface,
      palette: { grass_top: { r: 0, g: 1, b: 0 }, grss: { r: 0, g: 1, b: 0 } },
      solid: ["leaves", "leafs"],
    };
    expect(checkLook({ ...today, surface })).toEqual([
      "surface.palette has no texture grss",
      "surface.solid has no texture leafs",
    ]);
  });

  it("flags edge touches on a look without edges, which would never show", () => {
    const shape = { ...today.shape, edges: false, pool: 0.3, stitch: { r: 1, g: 1, b: 1 } };
    expect(checkLook({ ...today, shape })).toEqual([
      "shape.pool needs shape.edges",
      "shape.stitch needs shape.edges",
    ]);
  });

  it("wants outlines to have a width", () => {
    const after = { outline: { width: 0, strength: 0.5 } };
    expect(checkLook({ ...today, after })).toEqual(["after.outline.width must be 0.1–8, is 0"]);
  });
});
