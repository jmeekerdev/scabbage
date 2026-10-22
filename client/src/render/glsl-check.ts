/// <reference types="node" />
// Test-only: compiles shaders the way three would, without a browser, using glslangValidator.
import { spawnSync } from "node:child_process";
import { ShaderChunk } from "three";

export interface ShaderSource {
  vertexShader: string;
  fragmentShader: string;
  defines?: Record<string, unknown>;
}

// What three's WebGLProgram puts before every non-raw shader (GLSL 3 conversion, precision, built-ins).
const PRECISION = `precision highp float;
precision highp int;
precision highp sampler2D;
precision highp samplerCube;
precision highp sampler3D;
precision highp sampler2DArray;
#define HIGH_PRECISION
#define average threeAverage // glslang knows an average() WebGL doesn't, which clashes with three's`;

const VERTEX_PREFIX = `#define attribute in
#define varying out
#define texture2D texture
${PRECISION}
uniform mat4 modelMatrix;
uniform mat4 modelViewMatrix;
uniform mat4 projectionMatrix;
uniform mat4 viewMatrix;
uniform mat3 normalMatrix;
uniform vec3 cameraPosition;
uniform bool isOrthographic;
attribute vec3 position;
attribute vec3 normal;
attribute vec2 uv;`;

const FRAGMENT_PREFIX = `#define varying in
layout(location = 0) out highp vec4 pc_fragColor;
#define gl_FragColor pc_fragColor
#define texture2D texture
#define textureCube texture
${PRECISION}
uniform mat4 viewMatrix;
uniform vec3 cameraPosition;
uniform bool isOrthographic;
${ShaderChunk.colorspace_pars_fragment}
vec4 linearToOutputTexel( vec4 value ) { return sRGBTransferOETF( value ); }
float luminance( const in vec3 rgb ) { return dot( rgb, vec3( 0.2126, 0.7152, 0.0722 ) ); }`;

const INCLUDE = /^[ \t]*#include +<([\w\d./]+)>/gm;

function resolveIncludes(s: string): string {
  return s.replace(INCLUDE, (_, name: string) => resolveIncludes(ShaderChunk[name as never]));
}

// Three fills in its light and clipping plane counts; nothing here uses either.
const COUNTS = /\b(NUM_[A-Z_]+|UNION_CLIPPING_PLANES)\b/g;

/** The full source three would hand WebGL; `flags` are the defines three adds itself, like USE_FOG. */
function fullSource(stage: "vertex" | "fragment", src: ShaderSource, flags: string[]): string {
  const defines = [
    ...flags.map((f) => `#define ${f}`),
    ...Object.entries(src.defines ?? {}).map(([k, v]) => `#define ${k} ${String(v)}`),
  ];
  const body = stage === "vertex" ? src.vertexShader : src.fragmentShader;
  const prefix = stage === "vertex" ? VERTEX_PREFIX : FRAGMENT_PREFIX;
  return ["#version 300 es", prefix, ...defines, resolveIncludes(body).replace(COUNTS, "0")].join(
    "\n",
  );
}

function compile(stage: "vertex" | "fragment", source: string): string[] {
  const run = spawnSync("glslangValidator", ["--stdin", "-S", stage.slice(0, 4)], {
    input: source,
    encoding: "utf8",
  });
  if (run.error)
    throw new Error(`glslangValidator not found; run tests in nix-shell (${run.error.message})`);
  const lines = source.split("\n");
  return run.stdout
    .split("\n")
    .filter((l) => l.startsWith("ERROR: 0:"))
    .filter((l) => !l.includes(": '' :")) // follow-on errors such as "compilation terminated"
    .map((l) => {
      const line = Number(l.split(":")[2]);
      return `${stage}: ${l.slice("ERROR: ".length).trim()} | ${lines[line - 1]?.trim() ?? ""}`;
    });
}

/** Every compile error in a material's shaders, as "stage: error | source line"; empty when both compile. */
export function glslErrors(src: ShaderSource, flags: string[] = []): string[] {
  return [
    ...compile("vertex", fullSource("vertex", src, flags)),
    ...compile("fragment", fullSource("fragment", src, flags)),
  ];
}
