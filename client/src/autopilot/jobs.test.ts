import { describe, expect, it } from "vitest";
import { buildWorldFor } from "../player/building";
import { EYE_HEIGHT, newBody } from "../player/physics";
import { Jobs } from "./jobs";

const [AIR, STONE, PLANKS] = [0, 1, 5];

function setup() {
  const set = new Map<string, number>();
  const blockAt = (x: number, y: number, z: number) =>
    set.get(`${x},${y},${z}`) ?? (y < 10 ? STONE : AIR);
  const blocks = [
    { id: AIR, name: "air", solid: false, shape: 0, mineMs: 0 },
    { id: STONE, name: "stone", solid: true, shape: 0, mineMs: 0 },
    { id: PLANKS, name: "planks", solid: true, shape: 0, mineMs: 0 },
  ];
  const w = buildWorldFor({ radius: -1, height: 64, blocks }, blockAt);
  const notes: string[] = [];
  const reports: string[] = [];
  const jobs = new Jobs({
    note: (t) => notes.push(t),
    blockAt,
    agentJob: (id, state, percent, note) => reports.push(`${id} ${state} ${percent} ${note}`),
  });
  const body = newBody(1.5, 12, 1.5);
  const eye = { x: body.x, y: body.y + EYE_HEIGHT, z: body.z };
  /** Runs the autopilot until it is done, applying its edits to the world. */
  const finish = () => {
    for (let i = 0; i < 100 && jobs.busy; i++)
      for (const e of jobs.tick(0.125, body, eye, w)?.edits ?? [])
        for (const c of e.cells) set.set(`${c.x},${c.y},${c.z}`, e.block);
  };
  const at = { x: 1, y: 12, z: 1 };
  const named = (n: string) => ({ planks: PLANKS })[n];
  return { jobs, notes, reports, blockAt, finish, at, named };
}

describe("Jobs", () => {
  it("fills a box and says when it's done", () => {
    const { jobs, notes, blockAt, finish, at, named } = setup();
    jobs.run("fill", "0 10 0 2 10 2 planks", at, named, false);
    expect(jobs.busy).toBe(true);
    finish();
    expect(blockAt(2, 10, 2)).toBe(PLANKS);
    expect(notes.at(-1)).toBe("Done: 9 blocks.");
  });

  it("undoes the last job, and undoing again redoes it", () => {
    const { jobs, blockAt, finish, at, named } = setup();
    jobs.run("clear", "0 8 0 2 9 2", at, named, false);
    finish();
    expect(blockAt(1, 8, 1)).toBe(AIR);
    jobs.undo(false);
    finish();
    expect(blockAt(1, 8, 1)).toBe(STONE);
    jobs.undo(false);
    finish();
    expect(blockAt(1, 8, 1)).toBe(AIR);
  });

  it("says what is wrong instead of starting", () => {
    const { jobs, notes, at, named } = setup();
    jobs.run("fill", "0 0 0 1 1 1 lava", at, named, false);
    jobs.run("clear", "0 0 0 1 1 1", at, named, true);
    jobs.undo(false);
    expect(jobs.busy).toBe(false);
    expect(notes).toEqual([
      "There's no block named lava.",
      "/clear works in creative for now; survival is coming later.",
      "Nothing to undo.",
    ]);
  });

  it("stops when asked or when the player takes over", () => {
    const { jobs, notes, at, named } = setup();
    jobs.stop();
    jobs.run("goto", "~ ~10 ~", at, named, false);
    jobs.interrupt();
    jobs.interrupt();
    expect(jobs.busy).toBe(false);
    expect(notes).toEqual(["Nothing to stop.", "Flying to 1 22 1.", "Stopped: you took over."]);
  });

  it("runs an agent's job, marked so the player knows, and reports how it went", () => {
    const { jobs, notes, reports, blockAt, finish } = setup();
    const box = { min: { x: 0, y: 10, z: 0 }, max: { x: 3, y: 10, z: 0 } };
    const r = jobs.start({ kind: "fill", box, block: PLANKS }, false, 7);
    expect(r).toEqual({
      started: "Filling 4 blocks. /stop stops, /undo takes it back.",
      brushes: 4,
      seconds: 1,
    });
    finish();
    expect(blockAt(3, 10, 0)).toBe(PLANKS);
    expect(notes[0]).toBe("🤖 Filling 4 blocks. /stop stops, /undo takes it back.");
    expect(reports[0]).toBe("7 running 0 ");
    expect(reports.at(-1)).toBe("7 done 100 Done: 4 blocks.");
  });

  it("tells the agent when its job is stopped or replaced", () => {
    const { jobs, reports, at, named } = setup();
    jobs.start({ kind: "goto", target: { x: 50, y: 20, z: 0 } }, false, 1);
    jobs.interrupt();
    jobs.start({ kind: "goto", target: { x: 50, y: 20, z: 0 } }, false, 2);
    jobs.run("goto", "~ ~10 ~", at, named, false);
    expect(reports.filter((r) => r.includes("stopped"))).toEqual([
      "1 stopped 0 Stopped: you took over.",
      "2 stopped 0 Stopped: another job started.",
    ]);
  });

  it("places listed cells, clearing what's in their way, and refuses in survival", () => {
    const { jobs, blockAt, finish } = setup();
    const cells = [
      { x: 0, y: 9, z: 0, block: PLANKS },
      { x: 0, y: 11, z: 0, block: STONE },
      { x: 1, y: 9, z: 0, block: AIR },
    ];
    expect(jobs.start({ kind: "place", cells }, true, 3)).toEqual({
      error: "Agents work in creative for now; survival is coming later.",
    });
    expect(jobs.start({ kind: "place", cells }, false, 3)).toMatchObject({
      started: "Placing 3 blocks. /stop stops, /undo takes it back.",
    });
    finish();
    expect([blockAt(0, 9, 0), blockAt(0, 11, 0), blockAt(1, 9, 0)]).toEqual([PLANKS, STONE, AIR]);
  });

  it("refuses jobs that reach into a zone the player can't build in", () => {
    const notes: string[] = [];
    const jobs = new Jobs({
      note: (t) => notes.push(t),
      blockAt: () => AIR,
      refusedAt: (x) => (x >= 100 ? "Castle" : undefined),
    });
    const box = { min: { x: 90, y: 10, z: 0 }, max: { x: 101, y: 10, z: 0 } };
    const cells = [{ x: 105, y: 10, z: 3, block: PLANKS }];
    const refusal = {
      error:
        "Only builders of Castle can change blocks there (at 100, 0). Ask an admin to add you.",
    };
    expect(jobs.start({ kind: "fill", box, block: PLANKS }, false, 1)).toEqual(refusal);
    expect(jobs.start({ kind: "place", cells }, false, 1)).toEqual({
      error:
        "Only builders of Castle can change blocks there (at 105, 3). Ask an admin to add you.",
    });
    expect(jobs.start({ kind: "goto", target: { x: 105, y: 10, z: 0 } }, false, 1)).toHaveProperty(
      "started",
    );
    expect(notes[0]).toBe(refusal.error);
  });
});
