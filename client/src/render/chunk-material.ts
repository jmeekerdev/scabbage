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
import type { Atlas } from "./atlas";
import { TEXTURE_SIZE } from "./textures";
import { MAX_LIGHT } from "../world/light";
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
varying vec2 vUv;
varying float vLayer;
varying float vShade;
varying float vGlow;
#include <fog_pars_vertex>
void main() {
  vUv = uv / POSITION_SCALE;
  vGlow = light > 0.0 ? pow(0.82, MAX_LIGHT - light) : 0.0; // each block away dims it a little
  vLayer = layer;
  // Fixed per-face light: bright tops, darker sides and bottoms.
  int f = int(face + 0.5);
  vShade = f == 2 ? 1.0 : f == 3 ? 0.5 : f < 2 ? 0.8 : 0.65;
  vShade *= 0.5 + ao / 6.0; // corners tucked against blocks go darker
  vec4 mvPosition = modelViewMatrix * vec4(position / POSITION_SCALE, 1.0);
  gl_Position = projectionMatrix * mvPosition;
  #include <fog_vertex>
}`;

const fragmentShader = /* glsl */ `
precision highp sampler2DArray;
uniform sampler2DArray blocks;
uniform float daylight;
varying vec2 vUv;
varying float vLayer;
varying float vShade;
varying float vGlow;
#include <fog_pars_fragment>
void main() {
  vec4 color = texture(blocks, vec3(vUv, vLayer));
  #ifdef CUTOUT
  if (color.a < 0.5) discard;
  color.a = 1.0;
  #endif
  vec3 lit = max(vec3(daylight), vec3(1.0, 0.86, 0.66) * vGlow); // lamps glow warm
  gl_FragColor = vec4(color.rgb * vShade * lit, color.a);
  #include <colorspace_fragment>
  #include <fog_fragment>
}`;

/** Shared by every chunk material; the sky sets it each frame. */
export const daylight = { value: 1 };

/** Opaque blocks with see-through pixels cut out (leaves, glass). */
export function opaqueMaterial(blocks: Texture): ShaderMaterial {
  return chunkMaterial(blocks, { defines: { CUTOUT: "" } });
}

/** Blended blocks such as water, drawn after everything opaque. */
export function translucentMaterial(blocks: Texture): ShaderMaterial {
  return chunkMaterial(blocks, { transparent: true, depthWrite: false });
}

function chunkMaterial(
  blocks: Texture,
  extra: ConstructorParameters<typeof ShaderMaterial>[0],
): ShaderMaterial {
  // UniformsUtils.merge would clone the texture, so it is attached afterwards.
  const uniforms = UniformsUtils.merge([UniformsLib.fog]);
  uniforms.blocks = { value: blocks };
  uniforms.daylight = daylight;
  const defines = {
    POSITION_SCALE: POSITION_SCALE.toFixed(1),
    MAX_LIGHT: MAX_LIGHT.toFixed(1),
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
