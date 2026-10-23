import type { Cell } from "../player/brush";
import type { Job } from "./job";

type BlockAt = (x: number, y: number, z: number) => number | undefined;

/** The cells a job will build in and the ones it will break. */
export interface Preview {
  add: Cell[];
  remove: Cell[];
}

/** What a job would change, judged by what is loaded now; `filled` says a block is there to break. */
export function previewOf(job: Job, blockAt: BlockAt, filled: (block: number) => boolean): Preview {
  const out: Preview = { add: [], remove: [] };
  const has = (c: Cell) => {
    const b = blockAt(c.x, c.y, c.z);
    return b === undefined ? undefined : filled(b);
  };
  if (job.kind === "goto") return out;
  if (job.kind === "clear" || job.kind === "fill") {
    for (const c of boxCells(job.box.min, job.box.max)) {
      const there = has(c);
      if (job.kind === "clear" && there) out.remove.push(c);
      if (job.kind === "fill" && there === false) out.add.push(c);
    }
    return out;
  }
  for (const c of job.cells) {
    if (blockAt(c.x, c.y, c.z) === c.block) continue;
    const cell = { x: c.x, y: c.y, z: c.z };
    if (c.block !== 0) out.add.push(cell);
    else if (has(cell)) out.remove.push(cell);
  }
  return out;
}

function* boxCells(min: Cell, max: Cell): Generator<Cell> {
  for (let y = min.y; y <= max.y; y++)
    for (let z = min.z; z <= max.z; z++) for (let x = min.x; x <= max.x; x++) yield { x, y, z };
}
