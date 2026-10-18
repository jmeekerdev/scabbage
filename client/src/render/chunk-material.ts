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
import { paintTextures, TEXTURE_NAMES, TEXTURE_SIZE } from "./textures";

/** Every block texture as one array texture; repeat wrapping lets merged quads tile per block. */
export function blockTextures(): DataArrayTexture {
  const tex = new DataArrayTexture(
    paintTextures(),
    TEXTURE_SIZE,
    TEXTURE_SIZE,
    TEXTURE_NAMES.length,
  );
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
varying vec2 vUv;
varying float vLayer;
varying float vShade;
#include <fog_pars_vertex>
void main() {
  vUv = uv;
  vLayer = layer;
  // Fixed per-face light: bright tops, darker sides and bottoms.
  int f = int(face + 0.5);
  vShade = f == 2 ? 1.0 : f == 3 ? 0.5 : f < 2 ? 0.8 : 0.65;
  vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
  gl_Position = projectionMatrix * mvPosition;
  #include <fog_vertex>
}`;

const fragmentShader = /* glsl */ `
precision highp sampler2DArray;
uniform sampler2DArray blocks;
varying vec2 vUv;
varying float vLayer;
varying float vShade;
#include <fog_pars_fragment>
void main() {
  vec4 color = texture(blocks, vec3(vUv, vLayer));
  #ifdef CUTOUT
  if (color.a < 0.5) discard;
  color.a = 1.0;
  #endif
  gl_FragColor = vec4(color.rgb * vShade, color.a);
  #include <colorspace_fragment>
  #include <fog_fragment>
}`;

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
  return new ShaderMaterial({
    uniforms,
    vertexShader,
    fragmentShader,
    fog: true,
    side: FrontSide,
    ...extra,
  });
}
