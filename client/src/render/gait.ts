import { METRE } from "../world/constants";

const STRIDE = 2.5 / METRE; // radians of swing cycle per block walked
const MAX_SWING = 0.7; // radians at full walking speed
const WALK_SPEED = 4.5 * METRE; // blocks/s that gives the full swing
const TELEPORT_SPEED = 30 * METRE; // blocks/s; faster than this is a jump cut, not a walk
const AIRBORNE_SPEED = 1 * METRE; // vertical blocks/s that counts as jumping or falling
const SMOOTH_MS = 120;

interface Point {
  x: number;
  y: number;
  z: number;
}

export interface Stance {
  /** Limb angle in radians; legs and arms swing opposite ways by this much. */
  swing: number;
  airborne: boolean;
}

/** Turns a figure's movement into walking limb swings that ease in and out. */
export class Gait {
  private last?: Point;
  private phase = 0;
  private amplitude = 0;

  update(p: Point, dtMs: number): Stance {
    const prev = this.last ?? p;
    this.last = { ...p };
    const dt = Math.max(dtMs, 1) / 1000;
    let dist = Math.hypot(p.x - prev.x, p.z - prev.z);
    if (dist / dt > TELEPORT_SPEED) dist = 0;
    this.phase = (this.phase + dist * STRIDE) % (2 * Math.PI);
    const target = Math.min(1, dist / dt / WALK_SPEED) * MAX_SWING;
    this.amplitude += (target - this.amplitude) * Math.min(1, dtMs / SMOOTH_MS);
    return {
      swing: Math.sin(this.phase) * this.amplitude,
      airborne: Math.abs(p.y - prev.y) / dt > AIRBORNE_SPEED,
    };
  }
}
