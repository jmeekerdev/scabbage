import type { Cell } from "../player/brush";
import { boxBetween, MAX_PLACE, tooBig, type Job, type Placed } from "./job";
import type { Jobs, Started } from "./jobs";

export interface AgentHost {
  jobs: Jobs;
  blocks(): ReadonlyMap<string, number>; // what fill and place may use, by name
  survival(): boolean;
}

type Names = ReadonlyMap<string, number>;

const MAX_SUGGESTIONS = 8;

/** The answer sent back to the server: a result, or `error` saying what went wrong. */
export type AgentAnswer = Record<string, unknown> & { error?: string };

type Args = Record<string, unknown>;

/** Runs an agent's tool call in this tab: starts its job, or stops or undoes one. */
export function runAgentCommand(
  id: number,
  tool: string,
  json: string,
  host: AgentHost,
): AgentAnswer {
  let args: Args;
  try {
    args = JSON.parse(json) as Args;
  } catch {
    return { error: "The arguments weren't JSON." };
  }
  if (tool === "stop") return { stopped: host.jobs.stop() };
  if (tool === "undo") return answer(host.jobs.undo(host.survival(), id), id);
  const job = toJob(tool, args, host.blocks());
  if (typeof job === "string") return { error: job };
  return answer(host.jobs.start(job, host.survival(), id), id);
}

function answer(r: Started, id: number): AgentAnswer {
  return "error" in r ? r : { job: id, ...r };
}

/** Reads a tool's arguments into a job, or says what is wrong with them. */
function toJob(tool: string, args: Args, names: Names): Job | string {
  if (tool === "goto") {
    const target = cellOf(args);
    return target ? { kind: "goto", target } : "goto needs x, y and z as whole numbers.";
  }
  if (tool === "place") return placeJob(args.blocks, names);
  if (tool !== "clear" && tool !== "fill") return `This game can't run ${tool}.`;
  const [from, to] = [cellOf(args.from), cellOf(args.to)];
  if (!from || !to) return `${tool} needs from and to, each with x, y and z as whole numbers.`;
  const box = boxBetween(from, to);
  const big = tooBig(box);
  if (big) return big;
  if (tool === "clear" || args.block === "air") return { kind: "clear", box };
  const block = blockOf(args.block, names);
  return typeof block === "string" ? block : { kind: "fill", box, block };
}

function placeJob(blocks: unknown, names: Names): Job | string {
  if (!Array.isArray(blocks)) return "place needs blocks: a list of {x, y, z, block}.";
  if (blocks.length > MAX_PLACE)
    return `place takes at most ${MAX_PLACE.toLocaleString("en-US")} blocks; split it into several calls.`;
  const cells: Placed[] = [];
  const badCells: number[] = [];
  const unknown = new Map<string, number[]>(); // cells naming each unknown block
  (blocks as Args[]).forEach((b, i) => {
    const cell = cellOf(b);
    const block = b.block === "air" ? 0 : names.get(String(b.block).toLowerCase());
    if (!cell) badCells.push(i);
    if (block === undefined)
      unknown.set(String(b.block), [...(unknown.get(String(b.block)) ?? []), i]);
    if (cell && block !== undefined) cells.push({ ...cell, block });
  });
  if (!badCells.length && !unknown.size) return { kind: "place", cells };
  return placeProblems(badCells, unknown, names);
}

/** Says which cells of a place batch are wrong and how, so an agent can fix just those. */
function placeProblems(badCells: number[], unknown: Map<string, number[]>, names: Names): string {
  const lines = ["Nothing was placed; fix these and send the batch again."];
  if (badCells.length)
    lines.push(`${cellList(badCells, "needs", "need")} x, y and z as whole numbers.`);
  for (const [name, at] of unknown) {
    const close = closeNames(name.toLowerCase(), [...names.keys()]);
    const hint = close.length ? `; close names: ${close.join(", ")}` : "";
    lines.push(`${cellList(at, "names", "name")} "${name}", which isn't a block${hint}.`);
  }
  lines.push("The blocks tool lists every name.");
  return lines.join(" ");
}

const MAX_LISTED = 5;

/** "Cell 3 needs", "Cells 4, 5, 6, 7, 8 and 1 more need". */
function cellList(at: number[], one: string, many: string): string {
  if (at.length === 1) return `Cell ${at[0]} ${one}`;
  const shown = at.slice(0, MAX_LISTED).join(", ");
  const more = at.length > MAX_LISTED ? ` and ${at.length - MAX_LISTED} more` : "";
  return `Cells ${shown}${more} ${many}`;
}

function cellOf(v: unknown): Cell | undefined {
  const { x, y, z } = (v ?? {}) as Args;
  return [x, y, z].every(Number.isInteger)
    ? { x: x as number, y: y as number, z: z as number }
    : undefined;
}

function blockOf(name: unknown, names: Names): number | string {
  const wanted = String(name).toLowerCase();
  const block = typeof name === "string" ? names.get(wanted) : undefined;
  if (block !== undefined) return block;
  const close = closeNames(wanted, [...names.keys()]);
  const hint = close.length ? ` Close names: ${close.join(", ")}.` : "";
  return `There's no block named ${String(name)}.${hint} The blocks tool lists every name.`;
}

/** Names like `wanted`: the longest shared start first, then the most shared words, then shortest. */
function closeNames(wanted: string, names: readonly string[]): string[] {
  const words = wanted.split(/[_\s-]+/).filter((w) => w.length >= 3);
  const rank = (n: string) => ({
    n,
    start: sharedStart(wanted, n),
    words: words.filter((w) => n.includes(w)).length,
  });
  return names
    .map(rank)
    .filter((r) => r.words > 0 || r.start >= 4 || wanted.includes(r.n))
    .sort((a, b) => b.start - a.start || b.words - a.words || a.n.length - b.n.length)
    .slice(0, MAX_SUGGESTIONS)
    .map((r) => r.n);
}

function sharedStart(a: string, b: string): number {
  let i = 0;
  while (i < a.length && a[i] === b[i]) i++;
  return i;
}
