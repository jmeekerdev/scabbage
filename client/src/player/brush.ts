import { METER } from "../world/constants";
import type { RayHit } from "../world/raycast";

/** The widest brush: a cube as big as one block was before blocks shrank. */
export const MAX_BRUSH = METER;

export interface Cell {
  x: number;
  y: number;
  z: number;
}

/** A cube of blocks `size` a side whose lowest corner is x,y,z. */
export interface Cube extends Cell {
  size: number;
}

/** The brush size after `size`, starting over at 1 after the widest. */
export function nextBrush(size: number): number {
  return size >= MAX_BRUSH ? 1 : size + 1;
}

/**
 * The cube a brush covers at a hit: digging reaches into the face and building grows out of it.
 * Across the face x and z snap to the brush's grid, so walls line up, and y centers on the hit.
 */
export function brushCube(hit: RayHit, size: number, placing: boolean): Cube {
  const normal = [hit.nx, hit.ny, hit.nz];
  const [x, y, z] = [hit.x, hit.y, hit.z].map((c, axis) => {
    const n = normal[axis]!;
    if (n !== 0) return placing ? (n > 0 ? c + 1 : c - size) : n > 0 ? c - size + 1 : c;
    return axis === 1 ? c - Math.floor((size - 1) / 2) : Math.floor(c / size) * size;
  });
  return { x: x!, y: y!, z: z!, size };
}

/** Every block of a cube. */
export function cubeCells(c: Cube): Cell[] {
  const out: Cell[] = [];
  for (let x = c.x; x < c.x + c.size; x++)
    for (let y = c.y; y < c.y + c.size; y++)
      for (let z = c.z; z < c.z + c.size; z++) out.push({ x, y, z });
  return out;
}
