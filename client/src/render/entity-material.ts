import {
  type Material,
  MeshBasicMaterial,
  SpriteMaterial,
  type WebGLProgramParametersWithUniforms,
} from "three";
import type { Look } from "../looks/look";
import { daylight } from "./chunk-material";
import { LOOK_LIGHT_GLSL, LOOK_VEIL_GLSL, lookFeatures, lookUniforms } from "./look-shader";

// The light features figures share with blocks; edges, soft corners and surface touches belong to blocks.
const LIGHT_FEATURES = ["SUN", "BANDS"];

/** The shader features a look switches on for players, mobs and drops. */
export function entityFeatures(look: Look): Record<string, string> {
  const all = lookFeatures(look);
  return Object.fromEntries(LIGHT_FEATURES.filter((k) => k in all).map((k) => [k, ""]));
}

let features: Record<string, string> = {};
const live = new Set<Material>();

/** Relights every figure for a look, recompiling only when its light features change. */
export function setEntityLook(look: Look): void {
  const next = entityFeatures(look);
  if (JSON.stringify(next) === JSON.stringify(features)) return;
  features = next;
  for (const m of live) {
    m.defines = { ...features };
    m.needsUpdate = true;
  }
}

/** Hooks a material into the look's light and veil, and keeps it following look switches until disposed. */
function follow<T extends Material>(m: T, normal?: string): T {
  m.defines = { ...features };
  m.onBeforeCompile = (shader) => {
    lit(shader, normal ?? "vec3(0.0, 1.0, 0.0)");
    if (normal) passNormal(shader);
  };
  live.add(m);
  m.addEventListener("dispose", () => live.delete(m));
  return m;
}

/** Passes each mesh vertex's facing in the world on to the light. */
function passNormal(shader: WebGLProgramParametersWithUniforms): void {
  shader.vertexShader = shader.vertexShader
    .replace("#include <common>", `#include <common>\n${NORMAL_VARYING}`)
    .replace(
      "#include <fog_vertex>",
      "vLookNormal = mat3(modelMatrix) * normal;\n#include <fog_vertex>",
    );
  shader.fragmentShader = shader.fragmentShader.replace(
    "#include <common>",
    `#include <common>\n${NORMAL_VARYING}`,
  );
}

const NORMAL_VARYING = "varying vec3 vLookNormal;";

function lit(shader: WebGLProgramParametersWithUniforms, normal: string): void {
  Object.assign(shader.uniforms, lookUniforms, { daylight });
  shader.fragmentShader = shader.fragmentShader
    .replace("#include <common>", `#include <common>\nuniform float daylight;\n${LOOK_LIGHT_GLSL}`)
    .replace(
      "#include <fog_pars_fragment>",
      `#include <fog_pars_fragment>\n#ifdef USE_FOG\n${LOOK_VEIL_GLSL}\n#endif`,
    )
    .replace(
      "#include <opaque_fragment>",
      `outgoingLight *= lookLight(${normal}) * daylight;\n#include <opaque_fragment>`,
    )
    .replace(
      "#include <fog_fragment>",
      "#include <fog_fragment>\n#ifdef USE_FOG\ngl_FragColor = veiled(gl_FragColor);\n#endif",
    );
}

/** A mesh material lit like the blocks: the look's face light and sun, dimmed by the time of day. */
export function litMesh(
  params: ConstructorParameters<typeof MeshBasicMaterial>[0],
): MeshBasicMaterial {
  return follow(new MeshBasicMaterial(params), "normalize(vLookNormal)");
}

/** A sprite material lit like the top of a block, since a sprite has no facing of its own. */
export function litSprite(params: ConstructorParameters<typeof SpriteMaterial>[0]): SpriteMaterial {
  return follow(new SpriteMaterial(params));
}
