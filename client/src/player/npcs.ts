import { AVATAR_COLORS } from "../join/rules";
import type { Vec3 } from "../world/raycast";
import { pickPlayer, type RemoteState } from "./remote-players";

const NOTICE = 8; // blocks: closer than this, characters turn to face you
const NO_SKIN = new Uint8Array(0);

type Spot = { name: string; x: number; y: number; z: number; color: number };
type Listed = { id: number; name: string; pos?: { x: number; y: number; z: number } };

/** Characters standing on script blocks, as the server lists them. */
export class Npcs {
  private readonly spots = new Map<number, Spot>();

  change(npcs: readonly Listed[], gone: readonly number[]): void {
    for (const id of gone) this.spots.delete(id);
    for (const { id, name, pos } of npcs) {
      const x = (pos?.x ?? 0) + 0.5;
      const z = (pos?.z ?? 0) + 0.5;
      this.spots.set(id, { name, x, y: pos?.y ?? 0, z, color: colorFor(name) });
    }
  }

  clear(): void {
    this.spots.clear();
  }

  /** Each character as a figure to draw, turned toward `visitor` when they are close. */
  figures(visitor: { x: number; z: number }): RemoteState[] {
    return [...this.spots].map(([id, s]) => {
      const dx = visitor.x - s.x;
      const dz = visitor.z - s.z;
      const yaw = Math.hypot(dx, dz) < NOTICE ? Math.atan2(-dx, -dz) : 0;
      return {
        entity: id,
        name: s.name,
        color: s.color,
        skin: NO_SKIN,
        pose: { x: s.x, y: s.y, z: s.z, yaw, pitch: 0 },
      };
    });
  }

  /** The nearest character along the ray within reach. */
  pick(eye: Vec3, dir: Vec3, reach: number): { id: number; distance: number } | undefined {
    const hit = pickPlayer(eye, dir, this.figures({ x: Infinity, z: Infinity }), reach);
    return hit && { id: hit.entity, distance: hit.distance };
  }
}

/** A steady shirt color for a name, so a character looks the same to everyone. */
function colorFor(name: string): number {
  let h = 0;
  for (const ch of name) h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0;
  return AVATAR_COLORS[h % AVATAR_COLORS.length];
}
