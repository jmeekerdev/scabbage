import { cubeCells, MAX_BRUSH, type Cell, type Cube } from "../player/brush";
import type { Step } from "./autopilot";

/** The most blocks one job may change: a 32-block cube. */
export const MAX_JOB_BLOCKS = 32 ** 3;
/** The most cells one place job may list. */
export const MAX_PLACE = 4096;

/** The blocks from `min` to `max`, both included. */
export interface Box {
  min: Cell;
  max: Cell;
}

/** A cell and the block it should hold; 0 is air. */
export type Placed = Cell & { block: number };

type BlockAt = (x: number, y: number, z: number) => number | undefined;

export type Job =
  | { kind: "goto"; target: Cell }
  | { kind: "clear"; box: Box }
  | { kind: "fill"; box: Box; block: number }
  | { kind: "place"; cells: Placed[] }
  | { kind: "build"; template: string; cells: Placed[] };

/** The jobs chat commands start with a box or spot; place comes only from agents. */
export type CommandKind = Exclude<Job["kind"], "place" | "build">;

const USAGE = {
  goto: "Use /goto x y z, where ~ means where you are, e.g. /goto ~ ~10 ~.",
  clear: "Use /clear x1 y1 z1 x2 y2 z2 to break every block in that box.",
  fill: "Use /fill x1 y1 z1 x2 y2 z2 block to fill that box's empty spots.",
};

/** Reads a job command's arguments, or says what is wrong with them. */
export function parseJob(
  kind: CommandKind,
  arg: string,
  at: Cell,
  blockNamed: (name: string) => number | undefined,
): Job | string {
  const words = arg.trim().split(/\s+/).filter(Boolean);
  const wanted = kind === "goto" ? 3 : kind === "clear" ? 6 : 7;
  const coords = words.slice(0, kind === "goto" ? 3 : 6).map((w, i) => coordinate(w, at, i));
  if (words.length !== wanted || coords.some((c) => c === undefined)) return USAGE[kind];
  const [x1, y1, z1, x2, y2, z2] = coords as number[];
  if (kind === "goto") return { kind, target: { x: x1, y: y1, z: z1 } };
  const box = boxBetween({ x: x1, y: y1, z: z1 }, { x: x2, y: y2, z: z2 });
  const big = tooBig(box);
  if (big) return big;
  if (kind === "clear") return { kind, box };
  const name = words[6]!.toLowerCase();
  const block = blockNamed(name);
  return block === undefined ? `There's no block named ${name}.` : { kind, box, block };
}

/** A whole number, or ~ / ~n for the player's own coordinate on that axis plus n. */
export function coordinate(word: string, at: Cell, index: number): number | undefined {
  const base = word.startsWith("~") ? [at.x, at.y, at.z][index % 3]! : 0;
  const rest = word.startsWith("~") ? word.slice(1) || "0" : word;
  return /^-?\d+$/.test(rest) ? base + Number(rest) : undefined;
}

/** Why a box is too big for one job, or undefined if it fits. */
export function tooBig(box: Box): string | undefined {
  if (volume(box) <= MAX_JOB_BLOCKS) return undefined;
  return `That box has ${volume(box).toLocaleString("en-US")} blocks; one job can change at most ${MAX_JOB_BLOCKS.toLocaleString("en-US")}.`;
}

export function boxBetween(a: Cell, b: Cell): Box {
  const min = { x: Math.min(a.x, b.x), y: Math.min(a.y, b.y), z: Math.min(a.z, b.z) };
  return { min, max: { x: Math.max(a.x, b.x), y: Math.max(a.y, b.y), z: Math.max(a.z, b.z) } };
}

export function volume({ min, max }: Box): number {
  return (max.x - min.x + 1) * (max.y - min.y + 1) * (max.z - min.z + 1);
}

/** Splits a box into the biggest brush cubes that fit inside it, in the order to work them. */
export function tile(box: Box, topDown: boolean): Cube[] {
  const { min, max } = box;
  const cells: Cell[] = [];
  for (let y = min.y; y <= max.y; y++)
    for (let z = min.z; z <= max.z; z++)
      for (let x = min.x; x <= max.x; x++) cells.push({ x, y, z });
  return tileCells(cells, topDown);
}

/** Covers exactly `cells` with the biggest brush cubes that fit, in the order to work them. */
export function tileCells(cells: readonly Cell[], topDown: boolean): Cube[] {
  const key = (c: Cell) => `${c.x},${c.y},${c.z}`;
  const left = new Set(cells.map(key));
  const cubes: Cube[] = [];
  const scan = [...cells].sort((a, b) => a.y - b.y || a.z - b.z || a.x - b.x); // lowest corners first
  for (const c of scan) {
    if (!left.has(key(c))) continue;
    let size = MAX_BRUSH;
    while (size > 1 && !cubeCells({ x: c.x, y: c.y, z: c.z, size }).every((d) => left.has(key(d))))
      size--;
    const cube = { x: c.x, y: c.y, z: c.z, size };
    for (const d of cubeCells(cube)) left.delete(key(d));
    cubes.push(cube);
  }
  return order(cubes, topDown);
}

/**
 * Steps that make each cell hold its block: clear what's in the way top-down, then fill bottom-up,
 * one block at a time. Cells already right are left alone.
 */
export function changeSteps(cells: readonly Placed[], blockAt: BlockAt): Step[] {
  const changed = cells.filter((c) => blockAt(c.x, c.y, c.z) !== c.block);
  const clears = tileCells(
    changed.filter((c) => blockAt(c.x, c.y, c.z) !== 0),
    true,
  ).map((cube) => ({ cube, block: 0 }));
  const blocks = new Set(changed.map((c) => c.block).filter((b) => b !== 0));
  const fills = [...blocks].flatMap((block) =>
    tileCells(
      changed.filter((c) => c.block === block),
      false,
    ).map((cube) => ({ ...cube, block })),
  );
  return [...clears, ...order(fills, false).map(({ block, ...cube }) => ({ cube, block }))];
}

/** Layer by layer, snaking along rows so each cube is next to the last. */
export function order<T extends Cell>(cubes: readonly T[], topDown: boolean): T[] {
  const zs = [...new Set(cubes.map((c) => c.z))].sort((a, b) => a - b);
  const row = new Map(zs.map((z, i) => [z, i]));
  return [...cubes].sort((a, b) => {
    if (a.y !== b.y) return topDown ? b.y - a.y : a.y - b.y;
    if (a.z !== b.z) return a.z - b.z;
    return row.get(a.z)! % 2 ? b.x - a.x : a.x - b.x;
  });
}
