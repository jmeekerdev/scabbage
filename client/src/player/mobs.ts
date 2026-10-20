import { MobKind } from "../pb/scabbage/v1/net_pb";
import type { Vec3 } from "../world/raycast";
import type { Pose } from "./move-sync";
import { sampleAt, type Snapshot } from "./remote-players";

export const MOB_REACH = 4;
export const MOB_STEP_MS = 100; // how often the server moves mobs; mobs.Every on the server
const HURT_MS = 300;

/** Mob sizes; must match server/internal/mobs/ai.go. */
export const MOB_SIZE: Record<number, { halfWidth: number; height: number }> = {
  [MobKind.SHEEP]: { halfWidth: 0.45, height: 1.3 },
  [MobKind.SLIME]: { halfWidth: 0.4, height: 0.8 },
};

export interface MobInfo {
  id: number;
  kind: MobKind;
  pos?: { x: number; y: number; z: number };
  yaw: number;
  sheared: boolean;
  hurt: boolean;
}

export interface MobState {
  id: number;
  kind: MobKind;
  x: number;
  y: number;
  z: number;
  yaw: number;
  sheared: boolean;
  hurt: boolean; // flashing from a hit
}

interface Tracked {
  kind: MobKind;
  sheared: boolean;
  hurtUntil: number;
  snaps: Snapshot[];
}

/** Mobs, drawn two server steps in the past so their moves can be smoothly interpolated. */
export class Mobs {
  private readonly mobs = new Map<number, Tracked>();

  constructor(private readonly stepMs: number) {}

  change(mobs: readonly MobInfo[], gone: readonly number[], nowMs: number): void {
    for (const id of gone) this.mobs.delete(id);
    for (const m of mobs) {
      const pose: Pose = {
        x: m.pos?.x ?? 0,
        y: m.pos?.y ?? 0,
        z: m.pos?.z ?? 0,
        yaw: m.yaw,
        pitch: 0,
      };
      const t = this.mobs.get(m.id);
      if (!t) {
        this.mobs.set(m.id, {
          kind: m.kind,
          sheared: m.sheared,
          hurtUntil: 0,
          snaps: [{ atMs: nowMs, pose }],
        });
        continue;
      }
      this.moved(t, pose, nowMs);
      t.sheared = m.sheared;
      if (m.hurt) t.hurtUntil = nowMs + HURT_MS;
    }
  }

  clear(): void {
    this.mobs.clear();
  }

  sample(nowMs: number): MobState[] {
    const at = nowMs - 2 * this.stepMs;
    return [...this.mobs].map(([id, t]) => {
      const { x, y, z, yaw } = sampleAt(t.snaps, at);
      return { id, kind: t.kind, x, y, z, yaw, sheared: t.sheared, hurt: nowMs < t.hurtUntil };
    });
  }

  private moved(t: Tracked, pose: Pose, nowMs: number): void {
    const last = t.snaps.at(-1);
    // After a pause, start from where it stood one step ago rather than drifting from long ago.
    if (last && nowMs - last.atMs > 2 * this.stepMs)
      t.snaps.push({ atMs: nowMs - this.stepMs, pose: last.pose });
    t.snaps.push({ atMs: nowMs, pose });
  }
}

/** The nearest mob whose box the ray enters within reach. */
export function pickMob(
  eye: Vec3,
  dir: Vec3,
  mobs: readonly MobState[],
  reach: number,
): { id: number; distance: number } | undefined {
  let best: { id: number; distance: number } | undefined;
  for (const m of mobs) {
    const size = MOB_SIZE[m.kind];
    if (!size) continue;
    const w = size.halfWidth;
    const d = rayBox(eye, dir, [m.x - w, m.y, m.z - w], [m.x + w, m.y + size.height, m.z + w]);
    if (d !== undefined && d <= reach && (!best || d < best.distance))
      best = { id: m.id, distance: d };
  }
  return best;
}

/** Distance along a unit ray to where it enters the box, by the slab method. */
function rayBox(o: Vec3, dir: Vec3, lo: Vec3, hi: Vec3): number | undefined {
  let near = 0;
  let far = Infinity;
  for (let a = 0; a < 3; a++) {
    const d = dir[a] as number;
    const [l, h, p] = [lo[a] as number, hi[a] as number, o[a] as number];
    if (d === 0) {
      if (p < l || p > h) return undefined;
      continue;
    }
    const t1 = (l - p) / d;
    const t2 = (h - p) / d;
    near = Math.max(near, Math.min(t1, t2));
    far = Math.min(far, Math.max(t1, t2));
  }
  return near <= far ? near : undefined;
}
