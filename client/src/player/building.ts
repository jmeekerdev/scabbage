import { lookDirection, raycast, type RayHit, type Vec3 } from "../world/raycast";
import { HALF_WIDTH, HEIGHT, type Body } from "./physics";

/** How far the eye reaches; server/internal/edit allows a little more. */
export const REACH = 5;
const AIR = 0;

/** What the building rules need to know about the world; mirrors server/internal/edit. */
export interface BuildWorld {
  blockAt(x: number, y: number, z: number): number | undefined;
  solid: readonly boolean[];
  unbreakable: ReadonlySet<number>; // also never placeable
  inside(x: number, y: number, z: number): boolean;
}

export interface Edit {
  x: number;
  y: number;
  z: number;
  block: number;
}

/** Building rules for a world as the server describes it. */
export function buildWorldFor(
  info: {
    radius: number;
    height: number;
    blocks: readonly { id: number; name: string; solid: boolean }[];
  },
  blockAt: BuildWorld["blockAt"],
): BuildWorld {
  const { radius: r, height } = info;
  const solid: boolean[] = [];
  for (const b of info.blocks) solid[b.id] = b.solid;
  return {
    blockAt,
    solid,
    unbreakable: new Set(info.blocks.filter((b) => b.name === "bedrock").map((b) => b.id)),
    inside: (x, y, z) => y >= 0 && y < height && (r < 0 || (x >= -r && x < r && z >= -r && z < r)),
  };
}

/** The loaded solid block under the crosshair, if one is within reach. */
export function aim(eye: Vec3, yaw: number, pitch: number, w: BuildWorld): RayHit | undefined {
  return raycast(eye, lookDirection(yaw, pitch), REACH, (x, y, z) =>
    isSolid(w, w.blockAt(x, y, z)),
  );
}

export function breakEdit(hit: RayHit, w: BuildWorld): Edit | undefined {
  const current = w.blockAt(hit.x, hit.y, hit.z);
  if (current === undefined || w.unbreakable.has(current)) return undefined;
  return { x: hit.x, y: hit.y, z: hit.z, block: AIR };
}

/** Places `block` against the face that was hit, if that spot is free and clear of the body. */
export function placeEdit(hit: RayHit, block: number, body: Body, w: BuildWorld): Edit | undefined {
  const [x, y, z] = [hit.x + hit.nx, hit.y + hit.ny, hit.z + hit.nz];
  const current = w.blockAt(x, y, z);
  if (block === AIR || w.unbreakable.has(block) || !w.inside(x, y, z)) return undefined;
  if (current === undefined || isSolid(w, current)) return undefined;
  if (isSolid(w, block) && overlaps(body, x, y, z)) return undefined;
  return { x, y, z, block };
}

const isSolid = (w: BuildWorld, b: number | undefined) => b !== undefined && (w.solid[b] ?? false);

function overlaps(b: Body, x: number, y: number, z: number): boolean {
  return (
    b.x - HALF_WIDTH < x + 1 &&
    b.x + HALF_WIDTH > x &&
    b.y < y + 1 &&
    b.y + HEIGHT > y &&
    b.z - HALF_WIDTH < z + 1 &&
    b.z + HALF_WIDTH > z
  );
}
