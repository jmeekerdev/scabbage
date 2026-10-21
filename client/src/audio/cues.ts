import { METRE } from "../world/constants";

const STRIDE = 1.5 * METRE; // blocks between footsteps
const MAX_STEP = 1 * METRE; // blocks per frame; more than this is a teleport
const NEAR = 4 * METRE; // blocks within which sounds play at full volume
const FAR = 24 * METRE; // blocks beyond which sounds are silent

interface Walker {
  x: number;
  z: number;
  onGround: boolean;
  flying: boolean;
}

/** Decides when the local player's feet make a sound. */
export class Footsteps {
  private last?: { x: number; z: number };
  private walked = STRIDE;

  /** True when a footstep should play this frame. */
  update(w: Walker): boolean {
    const prev = this.last ?? w;
    this.last = { x: w.x, z: w.z };
    const dist = Math.hypot(w.x - prev.x, w.z - prev.z);
    if (!w.onGround || w.flying || dist === 0 || dist > MAX_STEP) {
      this.walked = STRIDE; // the next step on the ground sounds at once
      return false;
    }
    this.walked += dist;
    if (this.walked < STRIDE) return false;
    this.walked = 0;
    return true;
  }
}

/** Volume for a sound `dist` blocks away, fading linearly from NEAR to FAR. */
export function distanceGain(dist: number): number {
  return Math.min(1, Math.max(0, (FAR - dist) / (FAR - NEAR)));
}
