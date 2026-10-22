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
  const jobs = new Jobs({ note: (t) => notes.push(t), blockAt });
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
  return { jobs, notes, blockAt, finish, at, named };
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
});
