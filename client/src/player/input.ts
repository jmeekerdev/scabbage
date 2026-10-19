import type { MoveInput } from "./physics";

/** Building actions due this frame. */
export interface BuildActions {
  breaking: boolean;
  placing: boolean;
  picking: boolean;
}

/** What one input device asks for this frame. */
export interface Intent {
  move: MoveInput;
  build: BuildActions;
}

export const NO_INPUT: MoveInput = { forward: 0, right: 0, jump: false, descend: false };
export const NO_BUILD: BuildActions = { breaking: false, placing: false, picking: false };

/** Merges devices used together, e.g. a pad and a keyboard, capping movement at full speed. */
export function combineIntents(intents: readonly Intent[]): Intent {
  const sum = (f: (i: Intent) => number) =>
    Math.max(
      -1,
      Math.min(
        1,
        intents.reduce((t, i) => t + f(i), 0),
      ),
    );
  const any = (f: (i: Intent) => boolean) => intents.some(f);
  return {
    move: {
      forward: sum((i) => i.move.forward),
      right: sum((i) => i.move.right),
      jump: any((i) => i.move.jump),
      descend: any((i) => i.move.descend),
    },
    build: {
      breaking: any((i) => i.build.breaking),
      placing: any((i) => i.build.placing),
      picking: any((i) => i.build.picking),
    },
  };
}

/** A stick's tilt with a radial dead zone, rescaled so the zone's edge is 0 and full tilt is 1. */
export function stickAxes(x: number, y: number, deadZone: number): [number, number] {
  const len = Math.hypot(x, y);
  if (len <= deadZone) return [0, 0];
  const scale = Math.min(1, (len - deadZone) / (1 - deadZone)) / len;
  return [x * scale, y * scale];
}

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

/** Fires once on press, then every intervalMs while held, e.g. building with a held button. */
export class HoldRepeat {
  private nextMs?: number;

  constructor(private readonly intervalMs: number) {}

  press(nowMs: number): void {
    this.nextMs = nowMs;
  }

  release(): void {
    this.nextMs = undefined;
  }

  due(nowMs: number): boolean {
    if (this.nextMs === undefined || nowMs < this.nextMs) return false;
    this.nextMs = Math.max(this.nextMs + this.intervalMs, nowMs - this.intervalMs / 2);
    return true;
  }
}

const MAX_PITCH = Math.PI / 2 - 0.01;

/** View direction in radians; yaw 0 faces -z, positive pitch looks up. */
export class Look {
  yaw = 0;
  pitch = 0;

  constructor(private readonly radiansPerPixel: number) {}

  /** Turns by a pointer movement in pixels. */
  turn(dx: number, dy: number): void {
    this.turnBy(dx * this.radiansPerPixel, dy * this.radiansPerPixel);
  }

  /** Turns right by `right` and down by `down` radians. */
  turnBy(right: number, down: number): void {
    const yaw = this.yaw - right;
    this.yaw = Math.atan2(Math.sin(yaw), Math.cos(yaw));
    this.pitch = Math.min(MAX_PITCH, Math.max(-MAX_PITCH, this.pitch - down));
  }
}
