import type { Cell } from "../player/brush";
import { order, tileCells } from "./job";
import type { Step } from "./autopilot";

type BlockAt = (x: number, y: number, z: number) => number | undefined;

/** What each cell a job changed held before, so the job can be taken back. */
export class History {
  private readonly before = new Map<string, Cell & { block: number }>();

  get size(): number {
    return this.before.size;
  }

  /** Notes what `cells` hold now, unless this job already changed them. */
  record(cells: readonly Cell[], blockAt: BlockAt): void {
    for (const c of cells) {
      const key = `${c.x},${c.y},${c.z}`;
      if (!this.before.has(key)) this.before.set(key, { ...c, block: blockAt(c.x, c.y, c.z) ?? 0 });
    }
  }

  /** Steps that put every changed cell back: clear what's there top-down, then refill bottom-up. */
  undoSteps(blockAt: BlockAt): Step[] {
    const changed = [...this.before.values()].filter((c) => blockAt(c.x, c.y, c.z) !== c.block);
    const clears = tileCells(
      changed.filter((c) => blockAt(c.x, c.y, c.z) !== 0),
      true,
    ).map((cube) => ({ cube, block: 0 }));
    const olds = new Set(changed.map((c) => c.block).filter((b) => b !== 0));
    const fills = [...olds].flatMap((block) =>
      tileCells(
        changed.filter((c) => c.block === block),
        false,
      ).map((cube) => ({ ...cube, block })),
    );
    return [...clears, ...order(fills, false).map(({ block, ...cube }) => ({ cube, block }))];
  }
}
