import { Look } from "./input";
import type { Pose } from "./move-sync";
import { EYE_HEIGHT, newBody, stepBody, type MoveInput, type BoxesAt } from "./physics";

const STEP = 1 / 60;
const STEP_SLOP = 1e-9; // summed frame times drift just below whole steps
const MAX_FRAME = 0.25; // a stalled tab doesn't fast-forward
const LOOK_RADIANS_PER_PIXEL = 0.0025;

/** The local player: physics at a fixed rate so it behaves the same at any frame rate. */
export class Player {
  readonly body = newBody(0, 0, 0);
  readonly look = new Look(LOOK_RADIANS_PER_PIXEL);
  private prev = { x: 0, y: 0, z: 0 };
  private spare = 0;
  private canFly = true;

  place(x: number, y: number, z: number): void {
    Object.assign(this.body, { x, y, z, vy: 0 });
    this.prev = { x, y, z };
    this.spare = 0;
  }

  toggleFly(): void {
    if (!this.canFly) return;
    this.body.flying = !this.body.flying;
    this.body.vy = 0;
  }

  /** Survival players walk; turning flying off drops anyone in the air. */
  allowFlying(on: boolean): void {
    this.canFly = on;
    if (!on) this.body.flying = false;
  }

  update(dt: number, input: MoveInput, solid: BoxesAt): void {
    this.spare += Math.min(dt, MAX_FRAME);
    while (this.spare >= STEP - STEP_SLOP) {
      this.spare -= STEP;
      const { x, y, z } = this.body;
      this.prev = { x, y, z };
      stepBody(this.body, input, this.look.yaw, STEP, solid);
    }
  }

  /** Eye position, blended between the last two steps by the time left over. */
  eye(): { x: number; y: number; z: number } {
    const t = Math.max(0, this.spare) / STEP;
    const mix = (a: number, b: number) => a + (b - a) * t;
    const { prev: p, body: b } = this;
    return { x: mix(p.x, b.x), y: mix(p.y, b.y) + EYE_HEIGHT, z: mix(p.z, b.z) };
  }

  pose(): Pose {
    const { x, y, z } = this.body;
    return { x, y, z, yaw: this.look.yaw, pitch: this.look.pitch };
  }
}
