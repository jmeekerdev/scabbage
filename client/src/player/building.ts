import { lookDirection, raycast, type RayHit, type Vec3 } from "../world/raycast";
import { isStairs, pistonFacing, Shape, stairsFacing } from "../world/shapes";
import { HALF_WIDTH, HEIGHT, type Body } from "./physics";
import { METRE } from "../world/constants";

/** How far the eye reaches; server/internal/edit allows a little more. */
export const REACH = 5 * METRE;
const AIR = 0;

/** What the building rules need to know about the world; mirrors server/internal/edit. */
export interface BuildWorld {
  blockAt(x: number, y: number, z: number): number | undefined;
  solid: readonly boolean[];
  targetable: readonly boolean[]; // solid, flat and usable blocks and torches
  shape: readonly number[];
  usable: ReadonlySet<number>; // pressed by hand, like switches, instead of built against
  sixWay: ReadonlySet<number>; // placed facing any of six ways, like pistons
  item: readonly number[]; // the block players hold for each one
  unbreakable: ReadonlySet<number>; // also never placeable
  mineMs: readonly number[]; // how long survival players dig each block
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
    blocks: readonly {
      id: number;
      name: string;
      solid: boolean;
      shape: number;
      mineMs: number;
      usable?: boolean;
      item?: number;
    }[];
  },
  blockAt: BuildWorld["blockAt"],
): BuildWorld {
  const { radius: r, height } = info;
  const [solid, targetable]: [boolean[], boolean[]] = [[], []];
  const [shape, mineMs, item]: [number[], number[], number[]] = [[], [], []];
  for (const b of info.blocks) {
    solid[b.id] = b.solid;
    targetable[b.id] = b.solid || !!b.usable || b.shape === Shape.TORCH || b.shape === Shape.FLAT;
    shape[b.id] = b.shape;
    mineMs[b.id] = b.mineMs;
    item[b.id] = b.item || b.id;
  }
  return {
    blockAt,
    solid,
    targetable,
    shape,
    mineMs,
    item,
    usable: new Set(info.blocks.filter((b) => b.usable).map((b) => b.id)),
    sixWay: new Set(info.blocks.filter((b) => b.name === "piston").map((b) => b.id)),
    unbreakable: new Set(info.blocks.filter((b) => b.name === "bedrock").map((b) => b.id)),
    inside: (x, y, z) => y >= 0 && y < height && (r < 0 || (x >= -r && x < r && z >= -r && z < r)),
  };
}

/** The loaded block under the crosshair that can be broken, if one is within reach. */
export function aim(eye: Vec3, yaw: number, pitch: number, w: BuildWorld): RayHit | undefined {
  return raycast(eye, lookDirection(yaw, pitch), REACH, (x, y, z) =>
    isTargetable(w, w.blockAt(x, y, z)),
  );
}

export function breakEdit(hit: RayHit, w: BuildWorld): Edit | undefined {
  const current = w.blockAt(hit.x, hit.y, hit.z);
  if (current === undefined || w.unbreakable.has(current)) return undefined;
  return { x: hit.x, y: hit.y, z: hit.z, block: AIR };
}

/** Whether placing at `target` presses it instead; sneaking builds against it, e.g. to stack doors. */
export function usesBlock(target: number | undefined, sneaking: boolean, w: BuildWorld): boolean {
  return !sneaking && target !== undefined && w.usable.has(target);
}

/**
 * Places `block` against the face that was hit, if that spot is free and clear of the body.
 * Stairs turn to rise away from the player and pistons to push where they look.
 */
export function placeEdit(
  hit: RayHit,
  block: number,
  body: Body,
  look: { yaw: number; pitch: number },
  w: BuildWorld,
): Edit | undefined {
  const [x, y, z] = [hit.x + hit.nx, hit.y + hit.ny, hit.z + hit.nz];
  const current = w.blockAt(x, y, z);
  if (block === AIR || w.unbreakable.has(block) || !w.inside(x, y, z)) return undefined;
  if (current === undefined || isTargetable(w, current)) return undefined;
  if (w.solid[block] && overlaps(body, x, y, z)) return undefined;
  return { x, y, z, block: turned(block, look, w) };
}

function turned(block: number, look: { yaw: number; pitch: number }, w: BuildWorld): number {
  if (w.sixWay.has(block)) return pistonFacing(block, look.yaw, look.pitch);
  return isStairs(w.shape[block] ?? 0) ? stairsFacing(block, look.yaw) : block;
}

const isTargetable = (w: BuildWorld, b: number | undefined) =>
  b !== undefined && (w.targetable[b] ?? false);

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

/** Tells a fresh press from a held button's repeats, so holding a switch doesn't flick it. */
export class PressFilter {
  private lastMs = -Infinity;

  constructor(private readonly repeatMs: number) {}

  fresh(nowMs: number): boolean {
    const fresh = nowMs - this.lastMs > this.repeatMs * 1.5;
    this.lastMs = nowMs;
    return fresh;
  }
}

/** Stops a held place button that used a block, like a chest, from then building against it. */
export class UseHold {
  private untilMs = -Infinity;

  constructor(private readonly repeatMs: number) {}

  used(nowMs: number): void {
    this.untilMs = nowMs + this.repeatMs * 1.5;
  }

  /** Whether this place press continues a use; each repeat keeps the hold going. */
  blocks(nowMs: number): boolean {
    if (nowMs >= this.untilMs) return false;
    this.used(nowMs);
    return true;
  }
}
