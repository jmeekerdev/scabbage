import type { Cell } from "../../player/brush";
import type { Placed } from "../job";
import { turnCells, type Facing } from "./templates";

/** A saved or proposed plan, as the server sends it: palette keys in layers (bottom first) of rows (front first). */
export interface Plan {
  name: string;
  about?: string;
  palette: Record<string, string>;
  layers: string[][];
}

const KEEP = ".";
const AIR = "_";

/**
 * A plan's cells in the world: the middle of its front's bottom row at `at`, facing `facing`,
 * with `material` in place of its most-used block. Or why it can't be built.
 */
export function placePlan(
  plan: Plan,
  at: Cell,
  facing: Facing,
  material: string | undefined,
  names: ReadonlyMap<string, number>,
): Placed[] | string {
  const blocks = new Map<string, number>([[AIR, 0]]);
  for (const [key, name] of Object.entries(plan.palette)) {
    const id = names.get(name.toLowerCase());
    if (id === undefined) return `There's no block named ${name}.`;
    blocks.set(key, id);
  }
  const cells = localCells(plan, blocks);
  if (material !== undefined) {
    const id = names.get(material.toLowerCase());
    if (id === undefined) return `There's no block named ${material}.`;
    swapMostUsed(cells, id);
  }
  const width = [...(plan.layers[0]?.[0] ?? "")].length;
  return turnCells(cells, width, at, facing);
}

/** The plan's cells in its own space, rows read left to right as seen from the front. */
function localCells(plan: Plan, blocks: ReadonlyMap<string, number>): Placed[] {
  const out: Placed[] = [];
  plan.layers.forEach((layer, y) =>
    layer.forEach((row, z) => {
      const keys = [...row];
      keys.forEach((key, i) => {
        const block = blocks.get(key);
        if (key !== KEEP && block !== undefined) out.push({ x: keys.length - 1 - i, y, z, block });
      });
    }),
  );
  return out;
}

function swapMostUsed(cells: Placed[], to: number): void {
  const uses = new Map<number, number>();
  for (const c of cells) if (c.block !== 0) uses.set(c.block, (uses.get(c.block) ?? 0) + 1);
  const [most] = [...uses].sort((a, b) => b[1] - a[1])[0] ?? [];
  for (const c of cells) if (c.block === most) c.block = to;
}
