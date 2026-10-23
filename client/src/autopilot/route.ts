import type { Body, MoveInput } from "../player/physics";

const ARRIVED = 0.5; // blocks from the target that count as there
const SLOW_WITHIN = 3; // blocks; closer than this the autopilot eases off
const SINK_WITHIN = 3; // blocks across; farther out it keeps its height so it won't dive into walls
const STUCK_MOVE = 0.25; // blocks it must move each STUCK_S while trying to
const STUCK_S = 0.5;
const ESCAPE_S = 0.25; // about two meters at flying speed
// Ways out when stuck, tried in turn: over, around either side, then back out from under.
const ESCAPES: MoveInput[] = [
  { forward: 0, right: 0, jump: true, descend: false },
  { forward: 0, right: 1, jump: false, descend: false },
  { forward: 0, right: -1, jump: false, descend: false },
  { forward: -1, right: 0, jump: false, descend: false },
];

/** A point in blocks. */
export interface Vec {
  x: number;
  y: number;
  z: number;
}

/** One step of flying toward a point: what to press and which way to face. */
export interface SteerStep {
  move: MoveInput;
  yaw: number;
  stuck: boolean; // it just found itself stuck, and is trying a way out
}

/** Flies a body toward points like a player would, getting over or around whatever blocks the way. */
export class Steering {
  private anchor?: Vec;
  private stillS = 0;
  private escapeS = 0;
  private escapes = 0;

  /** One step toward `target`; `keepHeight` waits to sink until over it, unless the way is known clear. */
  steer(dt: number, body: Body, target: Vec, keepHeight = true): SteerStep {
    const [dx, dy, dz] = [target.x - body.x, target.y - body.y, target.z - body.z];
    const across = Math.hypot(dx, dz);
    const yaw = across > 1e-6 ? Math.atan2(-dx, -dz) : 0;
    const stuck = this.stuck(dt, body);
    if (stuck) this.escapeS = ESCAPE_S;
    if (this.escapeS > 0) {
      this.escapeS -= dt;
      const move = ESCAPES[(this.escapes - 1) % ESCAPES.length]!;
      return { yaw, stuck, move };
    }
    const move = {
      forward: Math.min(1, across / SLOW_WITHIN),
      right: 0,
      jump: dy > ARRIVED,
      descend: dy < -ARRIVED && (!keepHeight || across < SINK_WITHIN),
    };
    return { yaw, stuck, move };
  }

  arrived(body: Body, target: Vec): boolean {
    return Math.hypot(target.x - body.x, target.y - body.y, target.z - body.z) <= ARRIVED;
  }

  /** Forgets progress, e.g. after hovering to build, so a pause doesn't look like being stuck. */
  reset(): void {
    this.anchor = undefined;
    this.stillS = 0;
    this.escapes = 0;
  }

  private stuck(dt: number, body: Body): boolean {
    const a = this.anchor;
    if (!a || Math.hypot(body.x - a.x, body.y - a.y, body.z - a.z) > STUCK_MOVE) {
      this.anchor = { x: body.x, y: body.y, z: body.z };
      this.stillS = 0;
      return false;
    }
    this.stillS += dt;
    if (this.stillS < STUCK_S) return false;
    this.stillS = 0;
    this.escapes++;
    return true;
  }
}
