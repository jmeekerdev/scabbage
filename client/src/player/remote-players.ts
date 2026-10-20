import { rayBox, type Vec3 } from "../world/raycast";
import type { Pose } from "./move-sync";
import { HALF_WIDTH, HEIGHT } from "./physics";

export interface RemoteInfo {
  entity: number;
  name: string;
  color: number;
  skin: Uint8Array; // empty for the default look
}

export interface RemoteState extends RemoteInfo {
  pose: Pose;
}

export interface Snapshot {
  atMs: number;
  pose: Pose;
}

/** Other players, drawn two ticks in the past so their moves can be smoothly interpolated. */
export class RemotePlayers {
  private readonly players = new Map<number, { info: RemoteInfo; snaps: Snapshot[] }>();
  private readonly delayMs: number;

  constructor(private readonly tickMs: number) {
    this.delayMs = 2 * tickMs;
  }

  join(info: RemoteInfo, pose: Pose, nowMs: number): void {
    this.players.set(info.entity, { info, snaps: [{ atMs: nowMs, pose }] });
  }

  move(entity: number, pose: Pose, nowMs: number): void {
    const snaps = this.players.get(entity)?.snaps;
    const last = snaps?.at(-1);
    if (!snaps || !last) return;
    // After a pause, start from where they stood one tick ago rather than drifting from long ago.
    if (nowMs - last.atMs > 2 * this.tickMs)
      snaps.push({ atMs: nowMs - this.tickMs, pose: last.pose });
    snaps.push({ atMs: nowMs, pose });
  }

  setSkin(entity: number, skin: Uint8Array): void {
    const p = this.players.get(entity);
    if (p) p.info = { ...p.info, skin };
  }

  leave(entity: number): void {
    this.players.delete(entity);
  }

  /** Every player's interpolated pose at `nowMs`. */
  sample(nowMs: number): RemoteState[] {
    const at = nowMs - this.delayMs;
    return [...this.players.values()].map(({ info, snaps }) => ({
      ...info,
      pose: sampleAt(snaps, at),
    }));
  }
}

/** The pose at time `at`, dropping snapshots that are no longer needed. */
export function sampleAt(snaps: Snapshot[], at: number): Pose {
  while (snaps.length > 1 && (snaps[1] as Snapshot).atMs <= at) snaps.shift();
  const [a, b] = snaps as [Snapshot, Snapshot | undefined];
  if (!b || at <= a.atMs) return a.pose;
  return lerpPose(a.pose, b.pose, (at - a.atMs) / (b.atMs - a.atMs));
}

function lerpPose(a: Pose, b: Pose, t: number): Pose {
  const lerp = (p: number, q: number) => p + (q - p) * t;
  return {
    x: lerp(a.x, b.x),
    y: lerp(a.y, b.y),
    z: lerp(a.z, b.z),
    yaw: a.yaw + shortestTurn(a.yaw, b.yaw) * t,
    pitch: lerp(a.pitch, b.pitch),
  };
}

function shortestTurn(from: number, to: number): number {
  const d = (to - from) % (2 * Math.PI);
  return d > Math.PI ? d - 2 * Math.PI : d < -Math.PI ? d + 2 * Math.PI : d;
}

/** The nearest other player whose body the ray enters within reach. */
export function pickPlayer(
  eye: Vec3,
  dir: Vec3,
  players: readonly RemoteState[],
  reach: number,
): { entity: number; distance: number } | undefined {
  let best: { entity: number; distance: number } | undefined;
  for (const { entity, pose: p } of players) {
    const lo: Vec3 = [p.x - HALF_WIDTH, p.y, p.z - HALF_WIDTH];
    const d = rayBox(eye, dir, lo, [p.x + HALF_WIDTH, p.y + HEIGHT, p.z + HALF_WIDTH]);
    if (d !== undefined && d <= reach && (!best || d < best.distance))
      best = { entity, distance: d };
  }
  return best;
}
