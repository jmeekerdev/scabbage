import type { Change } from "./frame-state";

export const IDLE_AFTER_MS = 1000; // how long a still view runs at full speed
export const CALM_STEP_MS = 1000 / 30; // a still view where things move, like mobs or a flashing lamp
export const IDLE_STEP_MS = 100; // nothing changing: just watch for changes

/** Decides how fast the frame loop runs: full speed only while the player's own view changes. */
export class Pacer {
  private lastView = -Infinity;
  private lastObjects = -Infinity;
  private nowMs = 0;

  /** Records what a frame found changed. */
  frame(nowMs: number, change: Change): void {
    if (this.lastView === -Infinity || change === "view") this.lastView = nowMs;
    if (change === "objects") this.lastObjects = nowMs;
    this.nowMs = nowMs;
  }

  /** Back to full speed, e.g. on player input. */
  wake(nowMs: number): void {
    this.lastView = this.nowMs = nowMs;
  }

  /** Milliseconds between frames on a timer, or undefined for every display frame. */
  step(): number | undefined {
    if (this.nowMs - this.lastView < IDLE_AFTER_MS) return undefined;
    return this.nowMs - this.lastObjects < IDLE_AFTER_MS ? CALM_STEP_MS : IDLE_STEP_MS;
  }
}
