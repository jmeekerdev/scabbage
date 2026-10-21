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
varying vec2 vUv;
varying float vLayer;
varying float vShade;
varying float vGlow;
varying vec3 vInside;
#include <fog_pars_vertex>
void main() {
  vUv = uv / POSITION_SCALE;
  vGlow = light > 0.0 ? pow(LAMP_FADE, (MAX_LIGHT - light) / METRE) : 0.0; // each metre away dims it a little
  vLayer = layer;
  // Fixed per-face light: bright tops, darker sides and bottoms.
  int f = int(face + 0.5);
  vShade = f == 2 ? SHADE_TOP : f == 3 ? SHADE_BOTTOM : f < 2 ? SHADE_X : SHADE_Z;
  vShade *= 1.0 - AO + AO * ao / 3.0; // corners tucked against blocks go darker
  // A point just behind the face, so the fragment shader can tell which block it belongs to.
  vec3 normal = vec3(equal(ivec3(f / 2), ivec3(0, 1, 2))) * ((f & 1) == 1 ? -1.0 : 1.0);
  vInside = (modelMatrix * vec4(position / POSITION_SCALE, 1.0)).xyz - normal * 0.5;
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
varying vec3 vInside;
#include <fog_pars_fragment>

// A steady brightness per block, so neighbors of the same kind don't line up into a grid.
float tint(vec3 p) {
  uvec3 c = uvec3(ivec3(floor(p)));
  uint h = c.x * 73856093u ^ c.y * 19349663u ^ c.z * 83492791u;
  h = (h ^ (h >> 13u)) * 1274126177u;
  return 1.0 + TINT * (float(h & 255u) / 127.5 - 1.0);
}

void main() {
  vec4 color = texture(blocks, vec3(vUv, vLayer));
  #ifdef CUTOUT
  if (color.a < 0.5) discard;
  color.a = 1.0;
  #endif
  vec3 lit = max(vec3(daylight), LAMP * vGlow);
  gl_FragColor = vec4(color.rgb * vShade * tint(vInside) * lit, color.a);
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

/** Redraws a chunk material in another look; its shader recompiles once. */
export function applyLook(material: ShaderMaterial, look: Look): void {
  Object.assign(material.defines, lookDefines(look));
  material.needsUpdate = true;
}

/** The look's numbers as shader defines, which cost nothing per frame. */
export function lookDefines(look: Look): Record<string, string> {
  const n = (v: number) => v.toFixed(2);
  const { faces, lamp } = look.light;
  return {
    TINT: n(look.surface.tint),
    SHADE_TOP: n(faces.top),
    SHADE_BOTTOM: n(faces.bottom),
    SHADE_X: n(faces.x),
    SHADE_Z: n(faces.z),
    AO: n(look.light.ao),
    LAMP: `vec3(${n(lamp.r)}, ${n(lamp.g)}, ${n(lamp.b)})`,
    LAMP_FADE: n(look.light.lampFade),
  };
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
  const defines = {
    POSITION_SCALE: POSITION_SCALE.toFixed(1),
    MAX_LIGHT: MAX_LIGHT.toFixed(1),
    METRE: METRE.toFixed(1),
    ...lookDefines(look),
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
