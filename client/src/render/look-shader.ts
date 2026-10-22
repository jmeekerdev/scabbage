import {
  Color,
  SRGBColorSpace,
  Vector2,
  Vector3,
  Vector4,
  type ShaderMaterial,
  type Texture,
} from "three";
import type { Look, Rgb } from "../looks/look";
import { METRE } from "../world/constants";
import type { MeshOptions } from "../world/mesher";

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
  veil: { value: 0 },
  detail: { value: 0 },
  wash: { value: 0 },
  fibre: { value: 0 },
  pool: { value: 0 },
  stitch: { value: new Vector3() },
  heightRange: { value: new Vector2() }, // in blocks
  heightLift: { value: 0 },
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
  u.detail.value = look.surface.detail ?? 0;
  u.wash.value = look.surface.wash ?? 0;
  u.fibre.value = look.surface.fibre ?? 0;
  u.pool.value = look.shape.pool ?? 0;
  const st = look.shape.stitch ?? { r: 1, g: 1, b: 1 };
  const linear = new Color().setRGB(st.r, st.g, st.b, SRGBColorSpace);
  u.stitch.value.set(linear.r, linear.g, linear.b);
  const height = look.light.height;
  if (height) u.heightRange.value.set(height.low * METRE, height.high * METRE);
  u.heightLift.value = height?.lift ?? 0;
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
    DETAIL: !look.surface.textured && (look.surface.detail ?? 0) > 0,
    SUN: !!sun,
    BANDS: !!sun && sun.bands >= 2,
    SHINE: !!sun && sun.shine > 0,
    HEIGHT: (look.light.height?.lift ?? 0) > 0,
    EDGES: look.shape.edges,
    SOFT: look.shape.softCorners > 0,
    WASH: (look.surface.wash ?? 0) > 0,
    FIBRE: (look.surface.fibre ?? 0) > 0,
    POOL: look.shape.edges && (look.shape.pool ?? 0) > 0,
    STITCH: look.shape.edges && !!look.shape.stitch,
  };
  return Object.fromEntries(Object.entries(on).flatMap(([k, v]) => (v ? [[k, ""]] : [])));
}

const FEATURES = [
  "FLAT",
  "DETAIL",
  "SUN",
  "BANDS",
  "SHINE",
  "HEIGHT",
  "EDGES",
  "SOFT",
  "WASH",
  "FIBRE",
  "POOL",
  "STITCH",
];

/** The extra mesh data a look's shape needs; corners move at most half a block. */
export function meshOptions(look: Look): MeshOptions {
  return { edges: look.shape.edges, softCorners: Math.min(0.5, look.shape.softCorners) };
}

/** Switches a material to a look's features, recompiling it only if they changed. */
export function applyLook(material: ShaderMaterial, look: Look): void {
  const next = lookFeatures(look);
  const changed = FEATURES.some((k) => k in next !== k in material.defines);
  if (!changed) return;
  for (const k of FEATURES) delete material.defines[k];
  Object.assign(material.defines, next);
  material.needsUpdate = true;
}

/** veiled(c): the colour hidden in fog by the veil, while chunks rebuild for a new look. Needs the fog uniforms. */
export const LOOK_VEIL_GLSL = /* glsl */ `
uniform float veil;
vec4 veiled(vec4 c) {
  return vec4(mix(c.rgb, fogColor, veil), c.a);
}`;

/** lookLight(n): the light on a surface facing n; lookShine(n, toEye): the sun's highlight to add;
 * lookRise(y): the light's change with height. */
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
#ifdef HEIGHT
uniform vec2 heightRange;
uniform float heightLift;
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

float lookRise(float y) {
  #ifdef HEIGHT
  float t = clamp((y - heightRange.x) / (heightRange.y - heightRange.x), 0.0, 1.0);
  return 1.0 + heightLift * (2.0 * t - 1.0);
  #else
  return 1.0;
  #endif
}

vec3 lookShine(vec3 n, vec3 toEye) {
  #ifdef SHINE
  return vec3(pow(max(dot(reflect(-sunDir, n), toEye), 0.0), 8.0) * shine);
  #else
  return vec3(0.0);
  #endif
}`;
