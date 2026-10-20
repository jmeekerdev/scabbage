export type Vec3 = [number, number, number];

/** The block a ray hit and the normal of the face it entered through. */
export interface RayHit {
  x: number;
  y: number;
  z: number;
  nx: number;
  ny: number;
  nz: number;
  distance: number;
}

/** Unit vector for a view; yaw 0 faces -z, positive pitch looks up. */
export function lookDirection(yaw: number, pitch: number): Vec3 {
  const c = Math.cos(pitch);
  return [-Math.sin(yaw) * c, Math.sin(pitch), -Math.cos(yaw) * c];
}

/** Steps cell by cell along a unit ray (Amanatides–Woo) to the first block `hits` accepts. */
export function raycast(
  origin: Vec3,
  dir: Vec3,
  maxDistance: number,
  hits: (x: number, y: number, z: number) => boolean,
): RayHit | undefined {
  const cell = origin.map(Math.floor) as Vec3;
  const step = dir.map(Math.sign);
  // Distance along the ray to the next boundary on each axis, and between boundaries.
  const next = dir.map((d, a) =>
    d === 0 ? Infinity : ((d > 0 ? cell[a] + 1 : cell[a]) - origin[a]) / d,
  );
  const delta = dir.map((d) => (d === 0 ? Infinity : Math.abs(1 / d)));
  for (;;) {
    const a = next[0] < next[1] ? (next[0] < next[2] ? 0 : 2) : next[1] < next[2] ? 1 : 2;
    const distance = next[a];
    if (distance > maxDistance) return undefined;
    cell[a] += step[a];
    next[a] += delta[a];
    if (hits(cell[0], cell[1], cell[2])) {
      const n = [0, 0, 0];
      n[a] = -step[a];
      return { x: cell[0], y: cell[1], z: cell[2], nx: n[0], ny: n[1], nz: n[2], distance };
    }
  }
}

/** Distance along a unit ray to where it enters the box, by the slab method. */
export function rayBox(o: Vec3, dir: Vec3, lo: Vec3, hi: Vec3): number | undefined {
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
