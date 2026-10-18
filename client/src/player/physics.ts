// Body size must match server/internal/motion; speeds must stay under its limits.
export const HALF_WIDTH = 0.3;
export const HEIGHT = 1.8;
export const EYE_HEIGHT = 1.62;
export const WALK_SPEED = 4.5;
export const FLY_SPEED = 11;
const FLY_VERTICAL_SPEED = 8;
const GRAVITY = 28;
const JUMP_SPEED = 8.4;
const TERMINAL_SPEED = 50;
const EPS = 1e-6;

/** Whether the block at integer x,y,z blocks movement. */
export type SolidAt = (x: number, y: number, z: number) => boolean;

/** What the player wants to do this step; forward and right are -1..1. */
export interface MoveInput {
  forward: number;
  right: number;
  jump: boolean;
  descend: boolean;
}

/** The player's physical state; x,y,z is the center of the feet. */
export interface Body {
  x: number;
  y: number;
  z: number;
  vy: number;
  onGround: boolean;
  flying: boolean;
}

export function newBody(x: number, y: number, z: number): Body {
  return { x, y, z, vy: 0, onGround: false, flying: false };
}

/** Advances the body by dt seconds, facing yaw (0 looks toward -z). */
export function stepBody(b: Body, input: MoveInput, yaw: number, dt: number, solid: SolidAt): void {
  const speed = b.flying ? FLY_SPEED : WALK_SPEED;
  const [vx, vz] = horizontalVelocity(input, yaw, speed);
  b.vy = b.flying ? flyVertical(input) : fallVertical(b, input, dt);

  const falling = b.vy < 0;
  const hitY = moveAxis(b, 1, b.vy * dt, solid);
  b.onGround = falling && hitY;
  if (hitY) b.vy = 0;
  if (b.onGround) b.flying = false;
  moveAxis(b, 0, vx * dt, solid);
  moveAxis(b, 2, vz * dt, solid);
}

function horizontalVelocity(input: MoveInput, yaw: number, speed: number): [number, number] {
  const len = Math.hypot(input.forward, input.right);
  if (len === 0) return [0, 0];
  const scale = speed / Math.max(1, len);
  const [f, r] = [input.forward * scale, input.right * scale];
  const [sin, cos] = [Math.sin(yaw), Math.cos(yaw)];
  return [-sin * f + cos * r, -cos * f - sin * r];
}

function flyVertical(input: MoveInput): number {
  return (Number(input.jump) - Number(input.descend)) * FLY_VERTICAL_SPEED;
}

function fallVertical(b: Body, input: MoveInput, dt: number): number {
  if (input.jump && b.onGround) return JUMP_SPEED;
  return Math.max(b.vy - GRAVITY * dt, -TERMINAL_SPEED);
}

type Axis = 0 | 1 | 2;
const KEYS = ["x", "y", "z"] as const;

/** Moves along one axis until the first solid block; returns whether one was hit. */
function moveAxis(b: Body, axis: Axis, delta: number, solid: SolidAt): boolean {
  if (delta === 0) return false;
  const [lo, hi] = bounds(b);
  const face = delta > 0 ? hi[axis] : lo[axis];
  const target = face + delta;
  let reached = target;
  let hit = false;
  // Cells the leading face passes through, nearest first, including the one it starts in.
  const first = delta > 0 ? Math.floor(face) : Math.ceil(face) - 1;
  const last = delta > 0 ? Math.ceil(target) - 1 : Math.floor(target);
  const dir = delta > 0 ? 1 : -1;
  for (let c = first; dir * (last - c) >= 0; c += dir) {
    if (sliceSolid(lo, hi, axis, c, solid)) {
      // Never back up: a body already overlapping a solid cell just stays put.
      reached = delta > 0 ? Math.max(face, c) : Math.min(face, c + 1);
      hit = true;
      break;
    }
  }
  b[KEYS[axis]] += reached - face;
  return hit;
}

function bounds(b: Body): [number[], number[]] {
  return [
    [b.x - HALF_WIDTH, b.y, b.z - HALF_WIDTH],
    [b.x + HALF_WIDTH, b.y + HEIGHT, b.z + HALF_WIDTH],
  ];
}

/** Whether any block in layer c along axis, within the body's cross-section, is solid. */
function sliceSolid(lo: number[], hi: number[], axis: Axis, c: number, solid: SolidAt): boolean {
  const range = (a: number) =>
    a === axis ? [c, c] : [Math.floor(lo[a] + EPS), Math.ceil(hi[a] - EPS) - 1];
  const [x0, x1] = range(0);
  const [y0, y1] = range(1);
  const [z0, z1] = range(2);
  for (let y = y0; y <= y1; y++) {
    for (let z = z0; z <= z1; z++) {
      for (let x = x0; x <= x1; x++) if (solid(x, y, z)) return true;
    }
  }
  return false;
}
