import type { BoxesAt } from "../player/physics";
import { FULL_BLOCK, type Box } from "./shapes";
import type { WorldBounds } from "./streamer";

/** Collision for the player; mirrors the server's World.Boxes, plus unloaded chunks are walls. */
export function solidQuery(
  { radius, height }: WorldBounds,
  boxesById: readonly (readonly Box[])[],
  blockAt: (x: number, y: number, z: number) => number | undefined,
): BoxesAt {
  const inside = (x: number, z: number) =>
    radius < 0 || (x >= -radius && x < radius && z >= -radius && z < radius);
  return (x, y, z) => {
    if (y >= height) return [];
    if (y < 0 || !inside(x, z)) return FULL_BLOCK;
    const b = blockAt(x, y, z);
    return b === undefined ? FULL_BLOCK : (boxesById[b] ?? []);
  };
}
