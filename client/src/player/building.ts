import { lookDirection, raycast, type RayHit, type Vec3 } from "../world/raycast";
import { isStairs, padFacing, pistonFacing, Shape, stairsFacing } from "../world/shapes";
import { HALF_WIDTH, HEIGHT, type Body } from "./physics";
import { METER } from "../world/constants";
import { brushCube, cubeCells, type Cell, type Cube } from "./brush";

/** How far the eye reaches; server/internal/edit allows a little more. */
export const REACH = 5 * METER;
const AIR = 0;
// Blocks placed facing any of six ways, like pistons.
const SIX_WAY = [
  "piston",
  "sticky_piston",
  "diode",
  "gate_and",
  "delay_1",
  "relay",
  "capacitor_1",
  "block_detector",
];

/** What the building rules need to know about the world; mirrors server/internal/edit. */
export interface BuildWorld {
  blockAt(x: number, y: number, z: number): number | undefined;
  solid: readonly boolean[];
  targetable: readonly boolean[]; // solid, flat and usable blocks and torches
  shape: readonly number[];
  usable: ReadonlySet<number>; // pressed by hand, like switches, instead of built against
  liquid: ReadonlySet<number>; // aimed at and broken from outside, built into from anywhere
  sixWay: ReadonlySet<number>; // placed facing any of six ways, like pistons
  padWay: ReadonlySet<number>; // placed facing along the ground or up, like launch pads
  item: readonly number[]; // the block players hold for each one
  unbreakable: ReadonlySet<number>; // also never placeable
  mineMs: readonly number[]; // how long survival players dig each block
  inside(x: number, y: number, z: number): boolean;
}

/** A brush of `block`, and the blocks of it we expect the server to change. */
export interface Edit extends Cube {
  block: number;
  cells: Cell[];
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
    liquid: new Set(info.blocks.filter((b) => b.name === "water").map((b) => b.id)),
    sixWay: new Set(info.blocks.filter((b) => SIX_WAY.includes(b.name)).map((b) => b.id)),
    padWay: new Set(info.blocks.filter((b) => b.name === "launch_pad").map((b) => b.id)),
    unbreakable: new Set(info.blocks.filter((b) => b.name === "bedrock").map((b) => b.id)),
    inside: (x, y, z) => y >= 0 && y < height && (r < 0 || (x >= -r && x < r && z >= -r && z < r)),
  };
}

/** The loaded block under the crosshair that can be broken, if one is within reach. */
export function aim(eye: Vec3, yaw: number, pitch: number, w: BuildWorld): RayHit | undefined {
  const [ex, ey, ez] = eye.map(Math.floor) as Vec3;
  const underwater = isLiquid(w, w.blockAt(ex, ey, ez));
  return raycast(eye, lookDirection(yaw, pitch), REACH, (x, y, z) => {
    const b = w.blockAt(x, y, z);
    return isTargetable(w, b) || (!underwater && isLiquid(w, b));
  });
}

/** Breaks a brush of `size` into the face that was hit, leaving unbreakable blocks. */
export function breakEdit(hit: RayHit, w: BuildWorld, size = 1): Edit | undefined {
  const cube = brushCube(hit, size, false);
  // Liquid only breaks when aimed at, so digging a lake bed doesn't leave air pockets.
  const drains = isLiquid(w, w.blockAt(hit.x, hit.y, hit.z));
  const cells = cubeCells(cube).filter(({ x, y, z }) => {
    const b = w.blockAt(x, y, z);
    return (isTargetable(w, b) || (drains && isLiquid(w, b))) && !w.unbreakable.has(b!);
  });
  return cells.length ? { ...cube, block: AIR, cells } : undefined;
}

/** How long survival players dig an edit: the slowest block's time for each block of its width. */
export function digMs(edit: Edit, w: BuildWorld): number {
  const slowest = Math.max(
    0,
    ...edit.cells.map(({ x, y, z }) => w.mineMs[w.blockAt(x, y, z) ?? 0] ?? 0),
  );
  return slowest * edit.size;
}

/** Whether placing at `target` presses it instead; sneaking builds against it, e.g. to stack doors. */
export function usesBlock(target: number | undefined, sneaking: boolean, w: BuildWorld): boolean {
  return !sneaking && target !== undefined && w.usable.has(target);
}

/**
 * Places a brush of `block` against the face that was hit, into the free spots clear of the body.
 * Stairs turn to rise away from the player and pistons to push where they look.
 */
export function placeEdit(
  hit: RayHit,
  block: number,
  body: Body,
  look: { yaw: number; pitch: number },
  w: BuildWorld,
  size = 1,
): Edit | undefined {
  if (block === AIR || w.unbreakable.has(block)) return undefined;
  const cube = brushCube(hit, size, true);
  const cells = cubeCells(cube).filter(({ x, y, z }) => {
    const current = w.blockAt(x, y, z);
    if (!w.inside(x, y, z) || current === undefined || isTargetable(w, current)) return false;
    return !(w.solid[block] && overlaps(body, x, y, z));
  });
  return cells.length ? { ...cube, block: turned(block, look, w), cells } : undefined;
}

function turned(block: number, look: { yaw: number; pitch: number }, w: BuildWorld): number {
  if (w.sixWay.has(block)) return pistonFacing(block, look.yaw, look.pitch);
  if (w.padWay.has(block)) return padFacing(block, look.yaw, look.pitch);
  return isStairs(w.shape[block] ?? 0) ? stairsFacing(block, look.yaw) : block;
}

const isTargetable = (w: BuildWorld, b: number | undefined) =>
  b !== undefined && (w.targetable[b] ?? false);

const isLiquid = (w: BuildWorld, b: number | undefined) => b !== undefined && w.liquid.has(b);

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
