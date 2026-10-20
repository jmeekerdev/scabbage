export interface Pose {
  x: number;
  y: number;
  z: number;
  yaw: number;
  pitch: number;
  flying?: boolean;
}

const same = (a: Pose, b: Pose) =>
  a.x === b.x &&
  a.y === b.y &&
  a.z === b.z &&
  a.yaw === b.yaw &&
  a.pitch === b.pitch &&
  a.flying === b.flying;

/** Decides when to report the player's pose to the server: on change, at most once per interval. */
export class MoveSync {
  teleport = 0;
  private lastMs = -Infinity;
  private sent?: Pose;

  constructor(private readonly intervalMs: number) {}

  /** The pose to send now, if any. */
  due(nowMs: number, pose: Pose): Pose | undefined {
    if (nowMs - this.lastMs < this.intervalMs) return undefined;
    if (this.sent && same(this.sent, pose)) return undefined;
    this.lastMs = nowMs;
    this.sent = { ...pose };
    return this.sent;
  }

  /** The pose to send right now regardless of the interval, if it changed. */
  flush(pose: Pose): Pose | undefined {
    if (this.sent && same(this.sent, pose)) return undefined;
    this.sent = { ...pose };
    return this.sent;
  }

  /** Records a server teleport; the next pose is sent even if unchanged, to acknowledge it. */
  teleported(id: number): void {
    this.teleport = id;
    this.sent = undefined;
    this.lastMs = -Infinity;
  }
}
