import {
  BufferAttribute,
  BufferGeometry,
  LineSegments,
  Points,
  ShaderMaterial,
  Vector3,
  type Object3D,
  type Scene,
} from "three";
import type { Preset } from "../settings/settings";
import { METRE } from "../world/constants";
import { snowsAt, type WeatherState } from "./weather";

/** How many drops each preset can show at most, in a storm. */
export const DROPS: Record<Preset, number> = { low: 500, medium: 1200, high: 2000 };

const BOX = new Vector3(16 * METRE, 12 * METRE, 16 * METRE); // the space drops fall through
const RAIN = { fall: 8 * METRE, length: 0.4 * METRE, slant: 1.5 * METRE }; // per second; length
const SNOW = { fall: 1 * METRE, size: 60 }; // per second; flake size in pixels one block away

const vertexShader = /* glsl */ `
uniform vec3 center;
uniform vec3 box;
uniform vec3 tailOffset;
uniform float sway;
uniform float size;
attribute float tail;
void main() {
  vec3 head = (modelMatrix * vec4(position, 1.0)).xyz;
  vec3 at = center + mod(head - center + 0.5 * box, box) - 0.5 * box;
  at.x += sin(sway + position.y) * 0.5;
  at += tail * tailOffset;
  vec4 seen = viewMatrix * vec4(at, 1.0);
  gl_Position = projectionMatrix * seen;
  gl_PointSize = size / max(1.0, -seen.z);
}`;

const fragmentShader = /* glsl */ `
uniform vec3 color;
uniform float opacity;
void main() {
#ifdef FLAKES
  if (length(gl_PointCoord - 0.5) > 0.5) discard;
#endif
  gl_FragColor = vec4(color, opacity);
}`;

/** Rain or snow falling around the camera; moving the mesh each frame tells FrameState to draw. */
export class Precipitation {
  private readonly rain: LineSegments<BufferGeometry, ShaderMaterial>;
  private readonly snow: Points<BufferGeometry, ShaderMaterial>;
  private drops: number;

  constructor(scene: Scene, preset: Preset) {
    this.drops = DROPS[preset];
    const seeds = scatter(DROPS.high);
    this.rain = new LineSegments(dropGeometry(seeds, 2), material(0.35, false));
    this.snow = new Points(dropGeometry(seeds, 1), material(0.85, true));
    this.snow.material.uniforms.size.value = SNOW.size;
    for (const o of [this.rain, this.snow]) hide(o, scene);
  }

  setPreset(preset: Preset): void {
    this.drops = DROPS[preset];
  }

  /** Shows the weather at the eye, unless it is sheltered by something overhead. */
  update(
    nowMs: number,
    eye: { x: number; y: number; z: number },
    w: WeatherState,
    sheltered: boolean,
  ): void {
    const falling = w.wet > 0 && !sheltered;
    const snowing = snowsAt(eye.y);
    this.rain.visible = falling && !snowing;
    this.snow.visible = falling && snowing;
    const shown = snowing ? this.snow : this.rain;
    if (!shown.visible) return;
    const count = Math.round(this.drops * w.wet * (0.5 + 0.5 * w.storm));
    shown.geometry.setDrawRange(0, count * (snowing ? 1 : 2));
    const s = nowMs / 1000;
    const u = shown.material.uniforms;
    u.center.value.set(eye.x, eye.y, eye.z);
    if (snowing) {
      u.sway.value = s % (2 * Math.PI);
      shown.position.set(0, -(s * SNOW.fall) % BOX.y, 0);
    } else {
      const slant = RAIN.slant * w.storm;
      u.tailOffset.value.set((slant * RAIN.length) / RAIN.fall, RAIN.length, 0);
      shown.position.set((-s * slant) % BOX.x, -(s * RAIN.fall) % BOX.y, 0);
    }
  }
}

/** Random spots in the box, the same each time so drops don't jump when settings change. */
function scatter(n: number): Float32Array {
  const out = new Float32Array(n * 3);
  let seed = 1;
  const next = () => ((seed = (seed * 16807) % 2147483647) - 1) / 2147483646;
  for (let i = 0; i < n; i++) out.set([next() * BOX.x, next() * BOX.y, next() * BOX.z], i * 3);
  return out;
}

/** Each seed as a drop of `ends` vertices, the second one its tail. */
function dropGeometry(seeds: Float32Array, ends: number): BufferGeometry {
  const n = seeds.length / 3;
  const position = new Float32Array(n * ends * 3);
  const tail = new Float32Array(n * ends);
  for (let i = 0; i < n; i++)
    for (let e = 0; e < ends; e++) {
      position.set(seeds.subarray(i * 3, i * 3 + 3), (i * ends + e) * 3);
      tail[i * ends + e] = e;
    }
  const g = new BufferGeometry();
  g.setAttribute("position", new BufferAttribute(position, 3));
  g.setAttribute("tail", new BufferAttribute(tail, 1));
  return g;
}

function material(opacity: number, flakes: boolean): ShaderMaterial {
  return new ShaderMaterial({
    vertexShader,
    fragmentShader,
    defines: flakes ? { FLAKES: 1 } : {},
    uniforms: {
      center: { value: new Vector3() },
      box: { value: BOX },
      tailOffset: { value: new Vector3() },
      sway: { value: 0 },
      size: { value: 1 },
      color: { value: new Vector3(0.85, 0.9, 1) },
      opacity: { value: opacity },
    },
    transparent: true,
    depthWrite: false,
  });
}

function hide(o: Object3D, scene: Scene): void {
  o.name = "precipitation";
  o.frustumCulled = false; // the shader places drops around the camera wherever the mesh is
  o.visible = false;
  scene.add(o);
}

/** Whether anything but air is over the eye, below the height drops fall from. */
export function sheltered(
  blockAt: (x: number, y: number, z: number) => number | undefined,
  eye: { x: number; y: number; z: number },
): boolean {
  const [x, z] = [Math.floor(eye.x), Math.floor(eye.z)];
  for (let y = Math.floor(eye.y) + 1; y < eye.y + BOX.y; y++) {
    const b = blockAt(x, y, z);
    if (b !== undefined && b !== 0) return true;
  }
  return false;
}
