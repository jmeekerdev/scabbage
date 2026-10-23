import type { Cell } from "../../player/brush";
import type { Placed } from "../job";
import { cabin, tower } from "./buildings";
import { floor, stairs, wall } from "./parts";
import type { Part, Size, Template } from "./shape";

export type { Part, Size, Template } from "./shape";

/** The most blocks a template may be on any side. */
export const MAX_SIDE = 32;

export const TEMPLATES: ReadonlyMap<string, Template> = new Map([
  ["cabin", cabin],
  ["tower", tower],
  ["wall", wall],
  ["floor", floor],
  ["stairs", stairs],
]);

export const FACINGS = ["north", "east", "south", "west"] as const;
export type Facing = (typeof FACINGS)[number];

// The way each facing looks, as x and z steps; north is -z.
const FRONT: Record<Facing, [number, number]> = {
  north: [0, -1],
  east: [1, 0],
  south: [0, 1],
  west: [-1, 0],
};

type Names = ReadonlyMap<string, number>;

/**
 * A template's cells in the world: the middle of its front's bottom row at `at`, facing `facing`,
 * the rest behind it. Or why it can't be built.
 */
export function placeTemplate(
  name: string,
  asked: Partial<Size> | undefined,
  at: Cell,
  facing: Facing,
  material: string | undefined,
  names: Names,
): Placed[] | string {
  const t = TEMPLATES.get(name.toLowerCase());
  if (!t) return `There's no template called ${name}. Try ${[...TEMPLATES.keys()].join(", ")}.`;
  const size = { ...t.size, ...asked };
  const bad = sizeProblem(name, t, size);
  if (bad) return bad;
  const cells = t.cells(size);
  const blocks = partBlocks(t, new Set(cells.map((c) => c.part)), material, names);
  if (typeof blocks === "string") return blocks;
  return turnCells(
    cells.map((c) => ({ ...c, block: blocks[c.part] })),
    size.w,
    at,
    facing,
  );
}

/** Cells in a design's own space (x across its front, z back), turned and moved so its front's bottom middle is at `at`. */
export function turnCells(
  cells: readonly Placed[],
  width: number,
  at: Cell,
  facing: Facing,
): Placed[] {
  const [fx, fz] = FRONT[facing];
  const [rx, rz] = [-fz, fx]; // a quarter turn clockwise from the front, seen from above
  const mid = Math.floor(width / 2);
  return cells.map((c) => ({
    x: at.x + (c.x - mid) * rx - c.z * fx,
    y: at.y + c.y,
    z: at.z + (c.x - mid) * rz - c.z * fz,
    block: c.block,
  }));
}

function sizeProblem(name: string, t: Template, size: Size): string | undefined {
  const sides = [
    ["wide", size.w, t.min.w],
    ["deep", size.d, t.min.d],
    ["tall", size.h, t.min.h],
  ] as const;
  for (const [word, n, least] of sides) {
    if (!Number.isInteger(n) || n < least) return `A ${name} must be at least ${least} ${word}.`;
    if (n > MAX_SIDE) return `A ${name} can be at most ${MAX_SIDE} ${word}.`;
  }
  return t.check?.(size);
}

/** The block for each part used: the asked-for material, or the template's own. */
function partBlocks(
  t: Template,
  used: ReadonlySet<Part>,
  material: string | undefined,
  names: Names,
): Record<Part, number> | string {
  const main = (material ?? t.material).toLowerCase();
  const wanted = { main, floor: t.floor ?? main, roof: t.roof ?? main, window: "glass" };
  const out: Record<Part, number> = { main: 0, floor: 0, roof: 0, window: 0, air: 0 };
  for (const [part, block] of Object.entries(wanted) as [Part, string][]) {
    if (!used.has(part)) continue;
    const id = names.get(block);
    if (id === undefined) return `There's no block named ${block}.`;
    out[part] = id;
  }
  return out;
}
