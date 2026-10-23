import type { Box } from "../world/shapes";
import { METER } from "../world/constants";

// Body size must match server/internal/motion; speeds must stay under its limits.
export const HALF_WIDTH = 0.25 * METER;
export const HEIGHT = 1.6 * METER;
export const EYE_HEIGHT = 1.45 * METER;
export const WALK_SPEED = 4.5 * METER;
export const FLY_SPEED = 11 * METER;
const FLY_VERTICAL_SPEED = 8 * METER;
const GRAVITY = 28 * METER;
const JUMP_SPEED = 8.4 * METER; // about 1.26 m high
const TERMINAL_SPEED = 50 * METER;
const EPS = 1e-6;
const STEP_HEIGHT = 0.5 * METER; // server/internal/motion allows the same
const CLIMB_PACE = 0.5; // walking speed while climbing stairs
const STAIR_RUN = 1.5; // step-ups at most this many blocks apart are stairs, not a gentle slope
const STAIR_AHEAD = 1; // stairs pace lasts while another step is this near ahead

/** What players collide with in the block at integer x,y,z. */
export type BoxesAt = (x: number, y: number, z: number) => readonly Box[];

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
  vx: number; // momentum from a launch pad, carried through the air until landing
  vz: number;
  onGround: boolean;
  flying: boolean;
  sinceStep: number; // blocks walked since the last step-up
  onStairs: boolean; // stepped up twice within STAIR_RUN
}

export function newBody(x: number, y: number, z: number): Body {
  return {
    x,
    y,
    z,
    vy: 0,
    vx: 0,
    vz: 0,
    onGround: false,
    flying: false,
    sinceStep: Infinity,
    onStairs: false,
  };
}

/** Advances the body by dt seconds, facing yaw (0 looks toward -z). */
export function stepBody(
  b: Body,
  input: MoveInput,
  yaw: number,
  dt: number,
  boxesAt: BoxesAt,
): void {
  b.onStairs &&= stepAhead(b, horizontalVelocity(input, yaw, 1), boxesAt);
  const speed = b.flying ? FLY_SPEED : WALK_SPEED * (b.onStairs ? CLIMB_PACE : 1);
  const [vx, vz] = horizontalVelocity(input, yaw, speed);
  b.vy = b.flying ? flyVertical(input) : fallVertical(b, input, dt);

  const falling = b.vy < 0;
  const hitY = moveAxis(b, 1, b.vy * dt, boxesAt);
  b.onGround = falling && hitY;
  if (hitY) b.vy = 0;
  if (b.onGround) b.flying = false;
  if (b.onGround || b.flying) [b.vx, b.vz] = [0, 0];
  const [x, z] = [b.x, b.z];
  walkAxis(b, 0, (vx + b.vx) * dt, boxesAt);
  walkAxis(b, 2, (vz + b.vz) * dt, boxesAt);
  b.sinceStep += Math.hypot(b.x - x, b.z - z);
  b.onStairs &&= !b.flying && b.sinceStep <= STAIR_RUN;
}

/** Whether something rises in the body's way within STAIR_AHEAD along dx,dz; true when standing still. */
function stepAhead(b: Body, [dx, dz]: [number, number], boxesAt: BoxesAt): boolean {
  const len = Math.hypot(dx, dz);
  if (len === 0) return true;
  const probe = { ...b };
  const hitX = moveAxis(probe, 0, (dx / len) * STAIR_AHEAD, boxesAt);
  return moveAxis(probe, 2, (dz / len) * STAIR_AHEAD, boxesAt) || hitX;
}

/** Throws the body off the ground with velocity v, in blocks per second. */
export function launch(b: Body, v: { x: number; y: number; z: number }): void {
  [b.vx, b.vy, b.vz, b.onGround] = [v.x, v.y, v.z, false];
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

/** Walks a horizontal axis, lifting a grounded body onto slabs and stairs in the way. */
function walkAxis(b: Body, axis: Axis, delta: number, boxesAt: BoxesAt): void {
  const start = b[KEYS[axis]];
  if (!moveAxis(b, axis, delta, boxesAt) || !b.onGround) return;
  const lifted = { ...b };
  moveAxis(lifted, 1, STEP_HEIGHT, boxesAt);
  moveAxis(lifted, axis, delta - (b[KEYS[axis]] - start), boxesAt);
  moveAxis(lifted, 1, -STEP_HEIGHT, boxesAt);
  if (Math.abs(lifted[KEYS[axis]] - b[KEYS[axis]]) <= EPS) return;
  if (lifted.y > b.y + EPS) [lifted.onStairs, lifted.sinceStep] = [b.sinceStep <= STAIR_RUN, 0];
  Object.assign(b, lifted);
}

/** Moves along one axis until the first collision box; returns whether one was hit. */
function moveAxis(b: Body, axis: Axis, delta: number, boxesAt: BoxesAt): boolean {
  if (delta === 0) return false;
  const [lo, hi] = bounds(b);
  const dir = delta > 0 ? 1 : -1;
  const face = dir > 0 ? hi[axis] : lo[axis];
  let reached = face + delta;
  let hit = false;
  const cells = (a: number) =>
    a === axis
      ? [Math.floor(Math.min(face, reached)), Math.floor(Math.max(face, reached))]
      : [Math.floor(lo[a] + EPS), Math.ceil(hi[a] - EPS) - 1];
  const [[x0, x1], [y0, y1], [z0, z1]] = [cells(0), cells(1), cells(2)];
  for (let y = y0; y <= y1; y++) {
    for (let z = z0; z <= z1; z++) {
      for (let x = x0; x <= x1; x++) {
        for (const box of boxesAt(x, y, z)) {
          const cell = [x, y, z];
          if (!crosses(box, cell, lo, hi, axis)) continue;
          const near = cell[axis] + box[dir > 0 ? axis : axis + 3];
          const far = cell[axis] + box[dir > 0 ? axis + 3 : axis];
          const stop = stopAt(near, far, face, dir);
          if (stop === undefined || dir * (stop - reached) > 0) continue;
          reached = stop;
          hit = true;
        }
      }
    }
  }
  b[KEYS[axis]] += dir > 0 ? Math.max(0, reached - face) : Math.min(0, reached - face);
  return hit;
}

/** Where the leading face stops for a box, or undefined if the box is behind it. */
function stopAt(near: number, far: number, face: number, dir: number): number | undefined {
  if (dir * (near - face) >= -EPS) return near;
  return dir * (far - face) > EPS ? face : undefined; // already inside: stay put, never get pushed out
}

function bounds(b: Body): [number[], number[]] {
  return [
    [b.x - HALF_WIDTH, b.y, b.z - HALF_WIDTH],
    [b.x + HALF_WIDTH, b.y + HEIGHT, b.z + HALF_WIDTH],
  ];
}

/** Whether a box in cell overlaps the body's cross-section across the moving axis. */
function crosses(box: Box, cell: number[], lo: number[], hi: number[], axis: Axis): boolean {
  for (let a = 0; a < 3; a++) {
    if (a === axis) continue;
    if (cell[a] + box[a] >= hi[a] - EPS || cell[a] + box[a + 3] <= lo[a] + EPS) return false;
  }
  return true;
}
