import type { SolidAt } from "../player/physics";
import type { WorldBounds } from "./streamer";

/** Collision for the player; mirrors the server's World.Solid, plus unloaded chunks are walls. */
export function solidQuery(
  { radius, height }: WorldBounds,
  solid: readonly boolean[],
  blockAt: (x: number, y: number, z: number) => number | undefined,
): SolidAt {
  const inside = (x: number, z: number) =>
    radius < 0 || (x >= -radius && x < radius && z >= -radius && z < radius);
  return (x, y, z) => {
    if (y >= height) return false;
    if (y < 0 || !inside(x, z)) return true;
    const b = blockAt(x, y, z);
    return b === undefined || (solid[b] ?? false);
  };
}
