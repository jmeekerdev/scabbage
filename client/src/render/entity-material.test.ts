import { describe, expect, it } from "vitest";
import type { Look } from "../looks/look";
import { lookById } from "../looks/registry";
import { entityFeatures, litMesh, litSprite, setEntityLook } from "./entity-material";

const today = lookById("today");
const banded: Look = {
  ...today,
  light: {
    ...today.light,
    sun: {
      dir: { x: 1, y: 2, z: 0 },
      lit: { r: 1, g: 1, b: 1 },
      shade: { r: 0.5, g: 0.5, b: 0.6 },
      wrap: 0,
      bands: 3,
      shine: 0.2,
    },
  },
  shape: { ...today.shape, edges: true, softCorners: 0.5 },
};

/** Runs a material's shader hook over a stand-in of three's built-in shader. */
function compiled(m: { onBeforeCompile: (s: never, r: never) => void }) {
  const shader = {
    uniforms: {} as Record<string, unknown>,
    vertexShader: "#include <common>\n#include <fog_vertex>",
    fragmentShader:
      "#include <common>\n#include <fog_pars_fragment>\n#include <opaque_fragment>\n#include <fog_fragment>",
  };
  m.onBeforeCompile(shader as never, undefined as never);
  return shader;
}

describe("entityFeatures", () => {
  it("takes only the look's light features, not its block shaping", () => {
    expect(entityFeatures(today)).toEqual({});
    expect(entityFeatures(banded)).toEqual({ SUN: "", BANDS: "" });
  });
});

describe("litMesh", () => {
  it("lights by the look's face and sun light instead of fixed shading", () => {
    const s = compiled(litMesh({}));
    expect(s.fragmentShader).toContain("lookLight(normalize(vLookNormal))");
    expect(s.vertexShader).toContain("vLookNormal =");
    expect(s.uniforms).toHaveProperty("faces");
    expect(s.uniforms).toHaveProperty("daylight");
  });

  it("hides behind the veil with the chunks", () => {
    expect(compiled(litMesh({})).fragmentShader).toContain("veiled(gl_FragColor)");
  });
});

describe("litSprite", () => {
  it("is lit as if facing up, with no normal to read", () => {
    const s = compiled(litSprite({}));
    expect(s.fragmentShader).toContain("lookLight(vec3(0.0, 1.0, 0.0))");
    expect(s.vertexShader).not.toContain("vLookNormal");
  });
});

describe("setEntityLook", () => {
  it("recompiles every live entity material for the look's features", () => {
    const m = litMesh({});
    const before = m.version;
    setEntityLook(banded);
    expect(m.defines).toEqual({ SUN: "", BANDS: "" });
    expect(m.version).toBeGreaterThan(before);
    setEntityLook(today);
    expect(m.defines).toEqual({});
  });

  it("gives new materials the current look and forgets disposed ones", () => {
    setEntityLook(banded);
    const m = litSprite({});
    expect(m.defines).toEqual({ SUN: "", BANDS: "" });
    m.dispose();
    const v = m.version;
    setEntityLook(today);
    expect(m.version).toBe(v);
  });
});
