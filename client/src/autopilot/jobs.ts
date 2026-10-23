import type { Cell } from "../player/brush";
import type { BuildWorld } from "../player/building";
import type { Body } from "../player/physics";
import { Autopilot, BRUSHES_PER_S, type Outcome, type Plan, type Tick } from "./autopilot";
import { changeSteps, parseJob, tile, volume, type CommandKind, type Job } from "./job";
import type { Vec } from "./route";
import { previewOf, type Preview } from "./preview";
import { History } from "./undo";

type BlockAt = (x: number, y: number, z: number) => number | undefined;

export type JobState = "running" | "awaiting" | "done" | "stopped";

/** Jobs changing more blocks than this wait for the player to say /yes. */
export const CONFIRM_OVER = 500;

const TAKE_BACK = "/stop stops, /undo takes it back.";

/**
 * A job started, with what it says it's doing and, for building, the brushes it takes and the
 * seconds they take on the spot; or why it couldn't start.
 */
export type Started =
  { started: string; brushes?: number; seconds?: number; awaiting?: true } | { error: string };

export interface JobHost {
  note(text: string): void;
  blockAt: BlockAt;
  /** The protected zone's name if the player may not build in this column. */
  refusedAt?(x: number, z: number): string | undefined;
  /** Whether a block is there to break, rather than air, water or plants; air only by default. */
  filled?(block: number): boolean;
  /** Shows the changes a job waiting for /yes would make, or nothing. */
  preview?(p: Preview | undefined): void;
  /** Hears how a job an agent started is going; `agent` is the agent's id for it. */
  agentJob?(agent: number, state: JobState, percent: number, note: string): void;
}

const DOING: Record<string, string> = {
  goto: "Flying",
  build: "Building",
  clear: "Clearing",
  fill: "Filling",
  place: "Placing",
  undo: "Undoing",
};

interface Held {
  plan: Plan;
  agent: number;
  note: string;
}

/**
 * The /goto, /clear, /fill, /build, /stop and /undo jobs, and agents' jobs: one at a time, each
 * undoable. Big jobs, and agents' jobs when asked, wait for /yes.
 */
export class Jobs {
  private readonly pilot: Autopilot;
  private history = new History();
  private agent = 0; // the agent's id for the running job, 0 for the player's own
  private held?: Held; // a job waiting for /yes

  constructor(private readonly host: JobHost) {
    this.pilot = new Autopilot({
      progress: (p) => {
        host.note(`${DOING[p.kind]}… ${p.fraction * 100}%`);
        this.report(this.agent, "running", p.fraction * 100, "");
      },
      done: (kind, outcome) => this.ended("done", doneNote(kind, outcome)),
    });
  }

  get busy(): boolean {
    return this.pilot.busy;
  }

  /** Whether a job is waiting for /yes. */
  get waiting(): boolean {
    return this.held !== undefined;
  }

  /** Runs a chat command's job. */
  run(
    kind: CommandKind,
    arg: string,
    at: Cell,
    blockNamed: (name: string) => number | undefined,
    survival: boolean,
  ): void {
    const job = parseJob(kind, arg, at, blockNamed);
    if (typeof job === "string") return this.host.note(job);
    this.start(job, survival);
  }

  /** Starts a job, for an agent if `agent` is its id, saying so in chat; `confirm` waits for /yes. */
  start(job: Job, survival: boolean, agent = 0, confirm = false): Started {
    if (survival) return this.refuse(agent, `/${job.kind} works`);
    const protectedZone = this.protectedColumn(job);
    if (protectedZone) {
      this.host.note(protectedZone);
      return { error: protectedZone };
    }
    const plan = planFor(job, this.host.blockAt);
    const preview = previewOf(job, this.host.blockAt, this.host.filled ?? ((b) => b !== 0));
    const changes = preview.add.length + preview.remove.length;
    if (job.kind !== "goto" && (confirm || changes > CONFIRM_OVER))
      return this.hold({ plan, agent, note: startNote(job) }, preview);
    this.begin(plan, agent);
    return this.started(
      `${startNote(job)} ${job.kind === "goto" ? "" : TAKE_BACK}`.trim(),
      agent,
      plan,
    );
  }

  /** Starts the job waiting for /yes. */
  yes(): void {
    const held = this.held;
    if (!held) return this.host.note("Nothing is waiting for /yes.");
    this.held = undefined;
    this.host.preview?.(undefined);
    this.begin(held.plan, held.agent);
    this.started(`${held.note} ${TAKE_BACK}`, held.agent, held.plan);
  }

  /** Drops the job waiting for /yes. */
  no(): void {
    if (!this.held) return this.host.note("Nothing is waiting for /no.");
    this.drop("The player said no.");
    this.host.note("Cancelled.");
  }

  /** Stops the job, saying whether there was one. */
  stop(): string {
    const note = this.busy || this.held ? "Stopped." : "Nothing to stop.";
    this.cancel(note);
    this.drop(note);
    this.host.note(note);
    return note;
  }

  /** Cancels the job because the player moved or built themselves. */
  interrupt(): void {
    if (this.busy) {
      this.cancel("Stopped: you took over.");
      this.host.note("Stopped: you took over.");
    }
  }

  /** Puts back what the last job changed, as a job of its own that can be undone too. */
  undo(survival: boolean, agent = 0): Started {
    if (survival) return this.refuse(agent, "/undo works");
    this.cancel("Stopped: undoing.");
    this.drop("Stopped: undoing.");
    const steps = this.history.undoSteps(this.host.blockAt);
    if (!steps.length) {
      this.host.note("Nothing to undo.");
      return { error: "Nothing to undo." };
    }
    const plan = { kind: "undo", steps };
    this.begin(plan, agent);
    const blocks = steps.reduce((n, s) => n + s.cube.size ** 3, 0);
    return this.started(`Undoing ${count(blocks, "block")}.`, agent, plan);
  }

  /** Drives the autopilot one frame, noting what each edit replaces before it goes out. */
  tick(dt: number, body: Body, eye: Vec, w: BuildWorld): Tick | undefined {
    const tick = this.pilot.tick(dt, body, eye, w);
    for (const e of tick?.edits ?? []) this.history.record(e.cells, this.host.blockAt);
    return tick;
  }

  private begin(plan: Plan, agent: number): void {
    this.cancel("Stopped: another job started.");
    this.drop("Stopped: another job started.");
    this.history = new History();
    this.pilot.start(plan);
    this.agent = agent;
    this.report(agent, "running", 0, "");
  }

  /** Keeps a job until the player says /yes or /no, showing what it would change. */
  private hold(held: Held, preview: Preview): Started {
    this.cancel("Stopped: another job started.");
    this.drop("Stopped: another job started.");
    this.held = held;
    this.host.preview?.(preview);
    const note = `${held.note} Build it? /yes builds, /no cancels.`;
    this.report(held.agent, "awaiting", 0, "Waiting for the player to say /yes.");
    return { ...this.started(note, held.agent, held.plan), awaiting: true };
  }

  /** Forgets the job waiting for /yes, telling its agent why. */
  private drop(note: string): void {
    const held = this.held;
    if (!held) return;
    this.held = undefined;
    this.host.preview?.(undefined);
    this.report(held.agent, "stopped", 0, note);
  }

  private started(note: string, agent: number, plan: Plan): Started {
    this.host.note(agent ? `🤖 ${note}` : note);
    const brushes = plan.steps.length;
    if (!brushes) return { started: note };
    return { started: note, brushes, seconds: Math.ceil(brushes / BRUSHES_PER_S) };
  }

  /** Why the job may not change some of its columns, or undefined if it may change them all. */
  private protectedColumn(job: Job): string | undefined {
    const refusedAt = this.host.refusedAt;
    if (job.kind === "goto" || !refusedAt) return undefined;
    const columns = "cells" in job ? job.cells : boxColumns(job.box.min, job.box.max);
    for (const { x, z } of columns) {
      const zone = refusedAt(x, z);
      if (zone)
        return `Only builders of ${zone} can change blocks there (at ${x}, ${z}). Ask an admin to add you.`;
    }
    return undefined;
  }

  private refuse(agent: number, what: string): Started {
    const error = `${agent ? "Agents work" : what} in creative for now; survival is coming later.`;
    this.host.note(error);
    return { error };
  }

  private cancel(note: string): void {
    if (this.pilot.cancel()) this.ended("stopped", note);
  }

  private ended(state: JobState, note: string): void {
    if (state === "done") this.host.note(note);
    this.report(this.agent, state, state === "done" ? 100 : 0, note);
    this.agent = 0;
  }

  private report(agent: number, state: JobState, percent: number, note: string): void {
    if (agent) this.host.agentJob?.(agent, state, percent, note);
  }
}

function* boxColumns(min: Cell, max: Cell): Generator<{ x: number; z: number }> {
  for (let x = min.x; x <= max.x; x++) for (let z = min.z; z <= max.z; z++) yield { x, z };
}

/** The steps that carry out a job; goto flies to the middle of its block. */
export function planFor(job: Job, blockAt: BlockAt): Plan {
  if (job.kind === "goto") {
    const { x, y, z } = job.target;
    return { kind: job.kind, steps: [], target: { x: x + 0.5, y, z: z + 0.5 } };
  }
  if (job.kind === "place" || job.kind === "build")
    return { kind: job.kind, steps: changeSteps(job.cells, blockAt) };
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
  if (job.kind === "build") return `Building a ${job.template}.`;
  const blocks = job.kind === "place" ? job.cells.length : volume(job.box);
  return `${DOING[job.kind]} ${count(blocks, "block")}.`;
}
