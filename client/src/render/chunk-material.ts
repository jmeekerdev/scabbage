import {
  DataArrayTexture,
  FrontSide,
  LinearMipmapNearestFilter,
  NearestFilter,
  RepeatWrapping,
  ShaderMaterial,
  SRGBColorSpace,
  UniformsLib,
  UniformsUtils,
  type Texture,
} from "three";
import type { Look } from "../looks/look";
import { LOOK_LIGHT_GLSL, LOOK_VEIL_GLSL, lookFeatures, lookUniforms } from "./look-shader";
import type { Atlas } from "./atlas";
import { TEXTURE_SIZE } from "./textures";
import { MAX_LIGHT } from "../world/light";
import { METRE } from "../world/constants";
import { POSITION_SCALE } from "../world/mesher";

/** Every block texture as one array texture; repeat wrapping lets merged quads tile per block. */
export function blockTextures(atlas: Atlas): DataArrayTexture {
  const tex = new DataArrayTexture(atlas.pixels, TEXTURE_SIZE, TEXTURE_SIZE, atlas.depth);
  tex.colorSpace = SRGBColorSpace;
  tex.magFilter = NearestFilter;
  tex.minFilter = LinearMipmapNearestFilter;
  tex.wrapS = tex.wrapT = RepeatWrapping;
  tex.generateMipmaps = true;
  tex.needsUpdate = true;
  return tex;
}

const vertexShader = /* glsl */ `
attribute float face;
attribute float layer;
attribute float ao;
attribute float light;
uniform float lampFade;
varying vec2 vUv;
varying float vLayer;
varying vec3 vNormal;
varying float vAo;
varying float vGlow;
varying vec3 vInside;
varying vec3 vWorld;
#ifdef EDGES
attribute vec4 edge;
varying vec4 vEdge;
varying vec3 vSideU;
varying vec3 vSideV;
#endif
#ifdef SOFT
attribute vec3 shift;
#endif
#include <fog_pars_vertex>

vec3 axis(int a) {
  return vec3(equal(ivec3(a), ivec3(0, 1, 2)));
}

void main() {
  vUv = uv / POSITION_SCALE;
  vGlow = light > 0.0 ? pow(lampFade, (MAX_LIGHT - light) / METRE) : 0.0; // each metre away dims it a little
  vLayer = layer;
  vAo = ao / 3.0;
  int f = int(face + 0.5);
  vec3 faceNormal = axis(f / 2) * ((f & 1) == 1 ? -1.0 : 1.0);
  vNormal = faceNormal;
  vec3 moved = position;
  #ifdef SOFT
  moved += shift;
  if (dot(normal, normal) > 0.25) vNormal = normalize(normal);
  #endif
  #ifdef EDGES
  vEdge = edge;
  vSideU = axis((f / 2 + 1) % 3);
  vSideV = axis((f / 2 + 2) % 3);
  #endif
  // A point just behind the unmoved face, so the fragment shader can tell which block it belongs to.
  vInside = (modelMatrix * vec4(position / POSITION_SCALE, 1.0)).xyz - faceNormal * 0.5;
  vWorld = (modelMatrix * vec4(moved / POSITION_SCALE, 1.0)).xyz;
  vec4 mvPosition = modelViewMatrix * vec4(moved / POSITION_SCALE, 1.0);
  gl_Position = projectionMatrix * mvPosition;
  #include <fog_vertex>
}`;

const fragmentShader = /* glsl */ `
precision highp sampler2DArray;
uniform sampler2DArray blocks;
#ifdef FLAT
uniform sampler2D layerColors;
#endif
uniform float daylight;
uniform float tint;
uniform vec3 ao;
uniform vec3 lamp;
uniform float waterOpacity;
uniform float wash;
uniform float fibre;
uniform float pool;
uniform vec3 stitch;
varying vec2 vUv;
varying float vLayer;
varying vec3 vNormal;
varying float vAo;
varying float vGlow;
varying vec3 vInside;
varying vec3 vWorld;
#ifdef EDGES
varying vec4 vEdge;
varying vec3 vSideU;
varying vec3 vSideV;
#endif
#include <fog_pars_fragment>
${LOOK_LIGHT_GLSL}
${LOOK_VEIL_GLSL}

// The normal tipped outward near open sides (vEdge holds distance + 1 to each, 0 if closed), so shapes look rounded.
vec3 rounded(vec3 n) {
  #ifdef EDGES
  vec4 near = step(0.5, vEdge) * max(1.0 - (vEdge - 1.0) / EDGE_ROUND, 0.0);
  return normalize(n + (near.y - near.x) * vSideU + (near.w - near.z) * vSideV);
  #else
  return n;
  #endif
}

// A steady brightness per block, so neighbors of the same kind don't line up into a grid.
float blockTint(vec3 p) {
  uvec3 c = uvec3(ivec3(floor(p)));
  uint h = c.x * 73856093u ^ c.y * 19349663u ^ c.z * 83492791u;
  h = (h ^ (h >> 13u)) * 1274126177u;
  return 1.0 + tint * (float(h & 255u) / 127.5 - 1.0);
}

#if defined(WASH) || defined(FIBRE)
float hash3(vec3 p) {
  p = fract(p * 0.1031);
  p += dot(p, p.zyx + 31.32);
  return fract((p.x + p.y) * p.z);
}
float noise3(vec3 p) {
  vec3 i = floor(p), f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  vec2 o = vec2(0.0, 1.0);
  return mix(
    mix(mix(hash3(i), hash3(i + o.yxx), f.x), mix(hash3(i + o.xyx), hash3(i + o.yyx), f.x), f.y),
    mix(mix(hash3(i + o.xxy), hash3(i + o.yxy), f.x), mix(hash3(i + o.xyy), hash3(i + o.yyy), f.x), f.y),
    f.z);
}
#endif

#if defined(POOL) || defined(STITCH)
// Blocks to the nearest open side, or far away when every side is closed.
float openEdge() {
  vec4 d = mix(vec4(9.0), vEdge - 1.0, step(0.5, vEdge));
  return min(min(d.x, d.y), min(d.z, d.w));
}

// 0–1, how much a running stitch a little in from each open side covers this point.
float stitches() {
  vec2 along = vec2(dot(vWorld, vSideV), dot(vWorld, vSideU));
  float st = 0.0;
  for (int k = 0; k < 4; k++) {
    if (vEdge[k] < 0.5) continue;
    float a = fract((k < 2 ? along.x : along.y) * 2.0);
    float line = 1.0 - smoothstep(0.035, 0.065, abs(vEdge[k] - 1.18));
    st = max(st, line * smoothstep(0.1, 0.18, a) * (1.0 - smoothstep(0.62, 0.7, a)));
  }
  return st;
}
#endif

// Touches fixed to the world that a look may add: washes, fibres, pooled edges and stitches.
vec3 styled(vec3 c) {
  #ifdef WASH
  float hue = noise3(vWorld * 0.06 + 3.0), depth = noise3(vWorld * 0.21);
  vec3 w = mix(vec3(1.04, 1.0, 0.94), vec3(0.92, 0.97, 1.06), hue) * (0.86 + 0.28 * depth);
  c *= mix(vec3(1.0), w, wash);
  #endif
  #ifdef FIBRE
  float near = 1.0 - smoothstep(6.0, 20.0, distance(vWorld, cameraPosition)); // fades before it shimmers
  float fib = noise3(vWorld * vec3(9.0, 26.0, 9.0)) * 0.5 + hash3(floor(vWorld * 30.0)) * 0.5;
  c *= 1.0 + fibre * (fib - 0.5) * 0.24 * near;
  #endif
  #ifdef POOL
  c *= 1.0 - pool * (1.0 - smoothstep(0.0, 0.22, openEdge()));
  #endif
  #ifdef STITCH
  c = mix(c, stitch, stitches() * 0.8 * (1.0 - smoothstep(10.0, 30.0, distance(vWorld, cameraPosition))));
  #endif
  return c;
}

vec4 picture() {
  return texture(blocks, vec3(vUv, vLayer));
}

// The block's picture, or in untextured looks its flat colour; stickers keep their picture.
vec4 surface() {
  #ifdef FLAT
  vec4 flatColor = texelFetch(layerColors, ivec2(int(vLayer + 0.5), 0), 0);
  if (flatColor.a > 0.75) return picture();
  return vec4(flatColor.rgb, flatColor.a > 0.25 ? picture().a : 1.0);
  #else
  return picture();
  #endif
}

void main() {
  vec4 color = surface();
  #ifdef CUTOUT
  if (color.a < 0.5) discard;
  color.a = 1.0;
  #else
  color.a *= waterOpacity;
  #endif
  color.rgb = styled(color.rgb);
  vec3 n = rounded(normalize(vNormal));
  vec3 shade = lookLight(n) * mix(ao, vec3(1.0), vAo) * blockTint(vInside);
  vec3 lit = max(vec3(daylight), lamp * vGlow);
  vec3 toEye = normalize(cameraPosition - vWorld);
  gl_FragColor = vec4(color.rgb * shade * lit + lookShine(n, toEye) * daylight, color.a);
  #include <colorspace_fragment>
  #include <fog_fragment>
  gl_FragColor = veiled(gl_FragColor);
}`;

/** Shared by every chunk material; the sky sets it each frame. */
export const daylight = { value: 1 };

/** Opaque blocks with see-through pixels cut out (leaves, glass). */
export function opaqueMaterial(blocks: Texture, look: Look): ShaderMaterial {
  return chunkMaterial(blocks, look, { defines: { CUTOUT: "" } });
}

/** Blended blocks such as water, drawn after everything opaque. */
export function translucentMaterial(blocks: Texture, look: Look): ShaderMaterial {
  return chunkMaterial(blocks, look, { transparent: true, depthWrite: false });
}

function chunkMaterial(
  blocks: Texture,
  look: Look,
  extra: ConstructorParameters<typeof ShaderMaterial>[0],
): ShaderMaterial {
  // UniformsUtils.merge would clone the texture, so it is attached afterwards.
  const uniforms = UniformsUtils.merge([UniformsLib.fog]);
  uniforms.blocks = { value: blocks };
  uniforms.daylight = daylight;
  Object.assign(uniforms, lookUniforms);
  const defines = {
    POSITION_SCALE: POSITION_SCALE.toFixed(1),
    MAX_LIGHT: MAX_LIGHT.toFixed(1),
    METRE: METRE.toFixed(1),
    EDGE_ROUND: "0.42", // blocks from an open side where rounding starts
    ...lookFeatures(look),
    ...extra?.defines,
  };
  return new ShaderMaterial({
    uniforms,
    vertexShader,
    fragmentShader,
    fog: true,
    side: FrontSide,
    ...extra,
    defines,
  });
}
