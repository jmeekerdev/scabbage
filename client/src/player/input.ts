import type { MoveInput } from "./physics";

const axis = (keys: ReadonlySet<string>, plus: string[], minus: string[]) =>
  Number(plus.some((k) => keys.has(k))) - Number(minus.some((k) => keys.has(k)));

/** Maps held KeyboardEvent.code values to movement. */
export function moveInputFromKeys(keys: ReadonlySet<string>): MoveInput {
  return {
    forward: axis(keys, ["KeyW", "ArrowUp"], ["KeyS", "ArrowDown"]),
    right: axis(keys, ["KeyD", "ArrowRight"], ["KeyA", "ArrowLeft"]),
    jump: keys.has("Space"),
    descend: keys.has("ShiftLeft") || keys.has("ShiftRight"),
  };
}

/** Detects two presses within windowMs, e.g. double-tapping jump to fly. */
export class DoubleTap {
  private last = -Infinity;

  constructor(private readonly windowMs: number) {}

  press(nowMs: number): boolean {
    const hit = nowMs - this.last <= this.windowMs;
    this.last = hit ? -Infinity : nowMs;
    return hit;
  }
}

const MAX_PITCH = Math.PI / 2 - 0.01;

/** View direction in radians; yaw 0 faces -z, positive pitch looks up. */
export class Look {
  yaw = 0;
  pitch = 0;

  constructor(private readonly radiansPerPixel: number) {}

  turn(dx: number, dy: number): void {
    const yaw = this.yaw - dx * this.radiansPerPixel;
    this.yaw = Math.atan2(Math.sin(yaw), Math.cos(yaw));
    const pitch = this.pitch - dy * this.radiansPerPixel;
    this.pitch = Math.min(MAX_PITCH, Math.max(-MAX_PITCH, pitch));
  }
}
