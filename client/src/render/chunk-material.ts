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
import { LOOK_LIGHT_GLSL, lookFeatures, lookUniforms } from "./look-shader";
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
#include <fog_pars_vertex>
void main() {
  vUv = uv / POSITION_SCALE;
  vGlow = light > 0.0 ? pow(lampFade, (MAX_LIGHT - light) / METRE) : 0.0; // each metre away dims it a little
  vLayer = layer;
  vAo = ao / 3.0;
  int f = int(face + 0.5);
  vNormal = vec3(equal(ivec3(f / 2), ivec3(0, 1, 2))) * ((f & 1) == 1 ? -1.0 : 1.0);
  // A point just behind the face, so the fragment shader can tell which block it belongs to.
  vInside = (modelMatrix * vec4(position / POSITION_SCALE, 1.0)).xyz - vNormal * 0.5;
  vec4 mvPosition = modelViewMatrix * vec4(position / POSITION_SCALE, 1.0);
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
varying vec2 vUv;
varying float vLayer;
varying vec3 vNormal;
varying float vAo;
varying float vGlow;
varying vec3 vInside;
#include <fog_pars_fragment>
${LOOK_LIGHT_GLSL}

// A steady brightness per block, so neighbors of the same kind don't line up into a grid.
float blockTint(vec3 p) {
  uvec3 c = uvec3(ivec3(floor(p)));
  uint h = c.x * 73856093u ^ c.y * 19349663u ^ c.z * 83492791u;
  h = (h ^ (h >> 13u)) * 1274126177u;
  return 1.0 + tint * (float(h & 255u) / 127.5 - 1.0);
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
  vec3 shade = lookLight(vNormal) * mix(ao, vec3(1.0), vAo) * blockTint(vInside);
  vec3 lit = max(vec3(daylight), lamp * vGlow);
  vec3 toEye = normalize(cameraPosition - vInside - vNormal * 0.5);
  gl_FragColor = vec4(color.rgb * shade * lit + lookShine(vNormal, toEye) * daylight, color.a);
  #include <colorspace_fragment>
  #include <fog_fragment>
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
