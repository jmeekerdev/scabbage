import type { Cell } from "../player/brush";
import type { Step } from "./autopilot";
import { changeSteps, type Placed } from "./job";

type BlockAt = (x: number, y: number, z: number) => number | undefined;

/** What each cell a job changed held before, so the job can be taken back. */
export class History {
  private readonly before = new Map<string, Placed>();

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
    return changeSteps([...this.before.values()], blockAt);
  }
}
