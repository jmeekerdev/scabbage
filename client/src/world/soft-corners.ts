import { CHUNK_SIZE as S } from "./constants";
import { paddedIndex, POSITION_SCALE } from "./mesher";

/** How a block takes part in soft corners: open space, a natural block that softens, or a built one that pins. */
export const Smooth = { EMPTY: 0, NATURAL: 1, BUILT: 2 } as const;

const GRID = S + 1; // block corners along each side of a chunk

// The 8 blocks around a corner, as offsets from it to their centres, and the 12 edges joining them.
const AROUND = Array.from({ length: 8 }, (_, i) => [
  (i & 1) - 0.5,
  ((i >> 1) & 1) - 0.5,
  (i >> 2) - 0.5,
]);
const DUAL_EDGES = AROUND.flatMap((_, a) =>
  [1, 2, 4].filter((bit) => !(a & bit)).map((bit) => [a, a | bit]),
);

/**
 * Where each block corner of a padded chunk volume slides to, toward the shape of the natural blocks
 * around it (surface nets), and the slope normal it is lit by. Worked out once per corner, when asked.
 */
export class SoftCorners {
  private readonly shifts = new Int8Array(GRID ** 3 * 3);
  private readonly normals = new Int8Array(GRID ** 3 * 3);
  private readonly known = new Uint8Array(GRID ** 3);

  constructor(
    private readonly blocks: Uint16Array,
    private readonly kinds: Uint8Array,
    private readonly limit: number,
  ) {}

  /** Corner x,y,z's move in eighths of a block. */
  shift(x: number, y: number, z: number): number[] {
    const i = this.ensure(x, y, z) * 3;
    return [this.shifts[i], this.shifts[i + 1], this.shifts[i + 2]];
  }

  /** Corner x,y,z's slope normal scaled to ±127, or zeros where the face's own normal should stay. */
  normal(x: number, y: number, z: number): number[] {
    const i = this.ensure(x, y, z) * 3;
    return [this.normals[i], this.normals[i + 1], this.normals[i + 2]];
  }

  private ensure(x: number, y: number, z: number): number {
    const k = (y * GRID + z) * GRID + x;
    if (!this.known[k]) {
      this.known[k] = 1;
      this.work(k * 3, x, y, z);
    }
    return k;
  }

  private work(at: number, x: number, y: number, z: number): void {
    const kinds = AROUND.map(([dx, dy, dz]) => {
      const b = this.blocks[paddedIndex(x + dx - 0.5, y + dy - 0.5, z + dz - 0.5)];
      return this.kinds[b] ?? Smooth.BUILT;
    });
    const move = cornerMove(kinds, this.limit);
    if (!move) return;
    this.shifts.set(move.shift, at);
    this.normals.set(move.normal, at);
  }
}

/**
 * Where a corner slides, in eighths of a block, and its slope normal scaled to ±127, from the kinds
 * of the 8 blocks around it; undefined when a built block pins it.
 */
function cornerMove(
  kinds: readonly number[],
  limit: number,
): { shift: number[]; normal: number[] } | undefined {
  if (kinds.includes(Smooth.BUILT)) return undefined;
  const filled = kinds.map((k) => k === Smooth.NATURAL);
  const sum = [0, 0, 0];
  let crossings = 0;
  for (const [a, b] of DUAL_EDGES) {
    if (filled[a] === filled[b]) continue;
    for (let c = 0; c < 3; c++) sum[c] += (AROUND[a][c] + AROUND[b][c]) / 2;
    crossings++;
  }
  const g = [0, 0, 0];
  filled.forEach((f, n) => f && AROUND[n].forEach((o, c) => (g[c] += o)));
  const len = Math.hypot(...g);
  const shift = sum.map(
    (v) =>
      Math.round(
        Math.max(-limit, Math.min(limit, crossings ? v / crossings : 0)) * POSITION_SCALE,
      ) || 0,
  );
  const normal = g.map((v) => (len >= 1 ? Math.round((-v / len) * 127) || 0 : 0));
  return { shift, normal };
}

/** How far world corner x,y,z slides in blocks, matching the mesh; kindAt gives each block's Smooth kind. */
export function cornerShift(
  x: number,
  y: number,
  z: number,
  kindAt: (x: number, y: number, z: number) => number,
  limit: number,
): number[] {
  const kinds = AROUND.map(([dx, dy, dz]) => kindAt(x + dx - 0.5, y + dy - 0.5, z + dz - 0.5));
  return cornerMove(kinds, limit)?.shift.map((s) => s / POSITION_SCALE) ?? [0, 0, 0];
}
