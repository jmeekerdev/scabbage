import { ShaderMaterial } from "three";
import { describe, expect, it } from "vitest";
import type { Look, LookSun } from "../looks/look";
import { lookById } from "../looks/registry";
import { applyLook, lookFeatures, lookUniforms, meshOptions, setLookUniforms } from "./look-shader";

const today = lookById("today");
const sun: LookSun = {
  dir: { x: 0, y: 2, z: 0 },
  lit: { r: 1, g: 0.9, b: 0.8 },
  shade: { r: 0.6, g: 0.5, b: 0.9 },
  wrap: 0.3,
  bands: 0,
  shine: 0,
};
const withSun = (s: Partial<LookSun>): Look => ({
  ...today,
  light: { ...today.light, sun: { ...sun, ...s } },
});

describe("lookFeatures", () => {
  it("switches nothing on for Today, so it compiles as before", () => {
    expect(lookFeatures(today)).toEqual({});
  });

  it("switches on only the features a look uses", () => {
    expect(lookFeatures({ ...today, surface: { ...today.surface, textured: false } })).toEqual({
      FLAT: "",
    });
    expect(lookFeatures(withSun({}))).toEqual({ SUN: "" });
    expect(lookFeatures(withSun({ bands: 3, shine: 0.1 }))).toEqual({
      SUN: "",
      BANDS: "",
      SHINE: "",
    });
  });
});

describe("lookFeatures shape", () => {
  it("switches on edge and soft-corner shading with the mesh data they need", () => {
    const shaped = { ...today, shape: { edges: true, softCorners: 0.5 } };
    expect(lookFeatures(shaped)).toEqual({ EDGES: "", SOFT: "" });
    expect(meshOptions(shaped)).toEqual({ edges: true, softCorners: 0.5 });
    expect(meshOptions(today)).toEqual({ edges: false, softCorners: 0 });
  });

  it("caps soft corners at half a block", () => {
    expect(meshOptions({ ...today, shape: { edges: false, softCorners: 2 } }).softCorners).toBe(
      0.5,
    );
  });
});

describe("setLookUniforms", () => {
  it("puts the look's numbers in the shared uniforms, light colours as plain multipliers", () => {
    setLookUniforms(today);
    expect(lookUniforms.tint.value).toBe(0.06);
    expect(lookUniforms.faces.value.toArray()).toEqual([0.8, 1, 0.65, 0.5]);
    expect(lookUniforms.ao.value.toArray()).toEqual([0.5, 0.5, 0.5]);
    expect(lookUniforms.lampFade.value).toBe(0.82);
  });

  it("points the sun along its direction, normalised", () => {
    setLookUniforms(withSun({ bands: 3 }));
    expect(lookUniforms.sunDir.value.toArray()).toEqual([0, 1, 0]);
    expect(lookUniforms.sunBands.value).toBe(3);
  });
});

describe("applyLook", () => {
  it("recompiles a material only when its features change", () => {
    const m = new ShaderMaterial({ defines: { CUTOUT: "" } });
    const version = m.version;
    applyLook(m, today);
    expect(m.version).toBe(version);
    applyLook(m, withSun({}));
    expect(m.defines).toEqual({ CUTOUT: "", SUN: "" });
    expect(m.version).toBe(version + 1);
    applyLook(m, today);
    expect(m.defines).toEqual({ CUTOUT: "" });
  });
});
