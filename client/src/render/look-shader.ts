import { Vector3, Vector4, type ShaderMaterial, type Texture } from "three";
import type { Look, Rgb } from "../looks/look";

const rgb = (c: Rgb) => new Vector3(c.r, c.g, c.b);

/** The look's numbers, shared by every material that draws the world; changing them never recompiles. */
export const lookUniforms = {
  tint: { value: 0 },
  faces: { value: new Vector4() }, // x, top, z, bottom
  ao: { value: new Vector3() },
  lamp: { value: new Vector3() },
  lampFade: { value: 0 },
  sunDir: { value: new Vector3(0, 1, 0) },
  sunLit: { value: new Vector3() },
  sunShade: { value: new Vector3() },
  sunWrap: { value: 0 },
  sunBands: { value: 0 },
  shine: { value: 0 },
  waterOpacity: { value: 1 },
  layerColors: { value: null as Texture | null },
};

export function setLookUniforms(look: Look): void {
  const u = lookUniforms;
  const { faces, sun } = look.light;
  u.tint.value = look.surface.tint;
  u.faces.value.set(faces.x, faces.top, faces.z, faces.bottom);
  u.ao.value.copy(rgb(look.light.ao));
  u.lamp.value.copy(rgb(look.light.lamp));
  u.lampFade.value = look.light.lampFade;
  u.waterOpacity.value = look.water.opacity;
  if (!sun) return;
  u.sunDir.value.set(sun.dir.x, sun.dir.y, sun.dir.z).normalize();
  u.sunLit.value.copy(rgb(sun.lit));
  u.sunShade.value.copy(rgb(sun.shade));
  u.sunWrap.value = sun.wrap;
  u.sunBands.value = sun.bands;
  u.shine.value = sun.shine;
}

/** The shader features a look switches on, as defines; features left off aren't compiled at all. */
export function lookFeatures(look: Look): Record<string, string> {
  const sun = look.light.sun;
  const on = {
    FLAT: !look.surface.textured,
    SUN: !!sun,
    BANDS: !!sun && sun.bands >= 2,
    SHINE: !!sun && sun.shine > 0,
  };
  return Object.fromEntries(Object.entries(on).flatMap(([k, v]) => (v ? [[k, ""]] : [])));
}

const FEATURES = ["FLAT", "SUN", "BANDS", "SHINE"];

/** Switches a material to a look's features, recompiling it only if they changed. */
export function applyLook(material: ShaderMaterial, look: Look): void {
  const next = lookFeatures(look);
  const changed = FEATURES.some((k) => k in next !== k in material.defines);
  if (!changed) return;
  for (const k of FEATURES) delete material.defines[k];
  Object.assign(material.defines, next);
  material.needsUpdate = true;
}

/** lookLight(n): the light on a surface facing n; lookShine(n, toEye): the sun's highlight to add. */
export const LOOK_LIGHT_GLSL = /* glsl */ `
uniform vec4 faces;
#ifdef SUN
uniform vec3 sunDir;
uniform vec3 sunLit;
uniform vec3 sunShade;
uniform float sunWrap;
uniform float sunBands;
uniform float shine;
#endif

vec3 lookLight(vec3 n) {
  vec3 n2 = n * n;
  vec3 light = vec3(dot(n2, vec3(faces.x, n.y > 0.0 ? faces.y : faces.w, faces.z)));
  #ifdef SUN
  float t = clamp((dot(n, sunDir) + sunWrap) / (1.0 + sunWrap), 0.0, 1.0);
  #ifdef BANDS
  t = floor(t * (sunBands - 1.0) + 0.5) / (sunBands - 1.0);
  #endif
  light *= mix(sunShade, sunLit, t);
  #endif
  return light;
}

vec3 lookShine(vec3 n, vec3 toEye) {
  #ifdef SHINE
  return vec3(pow(max(dot(reflect(-sunDir, n), toEye), 0.0), 8.0) * shine);
  #else
  return vec3(0.0);
  #endif
}`;
