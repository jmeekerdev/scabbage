import { ShaderLib, Texture, type Material, type ShaderMaterial } from "three";
import { describe, expect, it } from "vitest";
import { LOOKS } from "../looks/registry";
import type { Look } from "../looks/look";
import { AfterPass } from "./after-pass";
import { opaqueMaterial, translucentMaterial } from "./chunk-material";
import { litMesh, litSprite, setEntityLook } from "./entity-material";
import { FarLand } from "./far-land";
import { glslErrors, type ShaderSource } from "./glsl-check";

const FOG = ["USE_FOG"];
const MAPPED = ["USE_FOG", "USE_MAP", "MAP_UV uv", "USE_ALPHATEST"];

const own = (o: object) => (o as { material: ShaderMaterial }).material;

/** A built-in material's shader after its onBeforeCompile hook, as three would build it. */
function hooked(m: Material, lib: keyof typeof ShaderLib): ShaderSource {
  const shader = { ...ShaderLib[lib], uniforms: {} };
  m.onBeforeCompile(shader as never, undefined as never);
  return { ...shader, defines: m.defines };
}

/** Each material that draws the world in a look, with the defines three adds for it. */
function programs(look: Look): [string, ShaderSource, string[]][] {
  setEntityLook(look);
  const after = (overhead: boolean) => {
    const pass = new AfterPass(0);
    pass.setLook(look, overhead);
    return own(pass);
  };
  return [
    ["opaque blocks", opaqueMaterial(new Texture(), look), FOG],
    ["translucent blocks", translucentMaterial(new Texture(), look), FOG],
    ["far land", own(new FarLand(() => {}, look)), FOG],
    ["afterwards", after(false), []],
    ["afterwards overhead", after(true), []],
    ["figure", hooked(litMesh({}), "basic"), FOG],
    ["skinned figure", hooked(litMesh({ alphaTest: 0.5 }), "basic"), MAPPED],
    ["drop", hooked(litSprite({}), "sprite"), MAPPED],
  ];
}

describe("every look's shaders", () => {
  for (const [id, look] of LOOKS)
    it(`compile in ${id}`, () => {
      const errors = programs(look).flatMap(([name, src, flags]) =>
        glslErrors(src, flags).map((e) => `${name}: ${e}`),
      );
      expect(errors).toEqual([]);
    });
});
