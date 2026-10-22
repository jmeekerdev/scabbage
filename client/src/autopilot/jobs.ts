import type { Cell } from "../player/brush";
import type { BuildWorld } from "../player/building";
import type { Body } from "../player/physics";
import { Autopilot, type Outcome, type Plan, type Tick } from "./autopilot";
import { parseJob, tile, volume, type Job } from "./job";
import type { Vec } from "./route";
import { History } from "./undo";

type BlockAt = (x: number, y: number, z: number) => number | undefined;

export interface JobHost {
  note(text: string): void;
  blockAt: BlockAt;
}

const DOING: Record<string, string> = {
  goto: "Flying",
  clear: "Clearing",
  fill: "Filling",
  undo: "Undoing",
};

/** The /goto, /clear, /fill, /stop and /undo jobs: one at a time, each one undoable. */
export class Jobs {
  private readonly pilot: Autopilot;
  private history = new History();

  constructor(private readonly host: JobHost) {
    this.pilot = new Autopilot({
      progress: (p) => host.note(`${DOING[p.kind]}… ${p.fraction * 100}%`),
      done: (kind, outcome) => host.note(doneNote(kind, outcome)),
    });
  }

  get busy(): boolean {
    return this.pilot.busy;
  }

  run(
    kind: Job["kind"],
    arg: string,
    at: Cell,
    blockNamed: (name: string) => number | undefined,
    survival: boolean,
  ): void {
    if (survival)
      return this.host.note(`/${kind} works in creative for now; survival is coming later.`);
    const job = parseJob(kind, arg, at, blockNamed);
    if (typeof job === "string") return this.host.note(job);
    this.begin(planFor(job));
    this.host.note(startNote(job));
  }

  stop(): void {
    this.host.note(this.pilot.cancel() ? "Stopped." : "Nothing to stop.");
  }

  /** Cancels the job because the player moved or built themselves. */
  interrupt(): void {
    if (this.pilot.cancel()) this.host.note("Stopped: you took over.");
  }

  /** Puts back what the last job changed, as a job of its own that can be undone too. */
  undo(survival: boolean): void {
    if (survival)
      return this.host.note("/undo works in creative for now; survival is coming later.");
    this.pilot.cancel();
    const steps = this.history.undoSteps(this.host.blockAt);
    if (!steps.length) return this.host.note("Nothing to undo.");
    this.begin({ kind: "undo", steps });
    this.host.note(
      `Undoing ${count(
        steps.reduce((n, s) => n + s.cube.size ** 3, 0),
        "block",
      )}.`,
    );
  }

  /** Drives the autopilot one frame, noting what each edit replaces before it goes out. */
  tick(dt: number, body: Body, eye: Vec, w: BuildWorld): Tick | undefined {
    const tick = this.pilot.tick(dt, body, eye, w);
    for (const e of tick?.edits ?? []) this.history.record(e.cells, this.host.blockAt);
    return tick;
  }

  private begin(plan: Plan): void {
    this.history = new History();
    this.pilot.start(plan);
  }
}

/** The steps that carry out a job; goto flies to the middle of its block. */
export function planFor(job: Job): Plan {
  if (job.kind === "goto") {
    const { x, y, z } = job.target;
    return { kind: job.kind, steps: [], target: { x: x + 0.5, y, z: z + 0.5 } };
  }
  const block = job.kind === "fill" ? job.block : 0;
  const steps = tile(job.box, job.kind === "clear").map((cube) => ({ cube, block }));
  return { kind: job.kind, steps };
}

function doneNote(kind: string, { blocks, missed }: Outcome): string {
  if (kind === "goto") return missed ? "Stuck: I couldn't get any closer." : "Here.";
  const out = missed ? ` I couldn't get near ${count(missed, "spot")}.` : "";
  return `Done: ${count(blocks, "block")}.${out}`;
}

/** "1 block", "1,240 blocks". */
const count = (n: number, thing: string) =>
  `${n.toLocaleString("en-US")} ${thing}${n === 1 ? "" : "s"}`;

function startNote(job: Job): string {
  if (job.kind === "goto") return `Flying to ${job.target.x} ${job.target.y} ${job.target.z}.`;
  return `${DOING[job.kind]} ${count(volume(job.box), "block")}. /stop stops, /undo takes it back.`;
}
