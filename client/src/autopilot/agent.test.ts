import { describe, expect, it } from "vitest";
import { runAgentCommand } from "./agent";
import { MAX_PLACE } from "./job";
import { Jobs } from "./jobs";

function setup(survival = false) {
  const jobs = new Jobs({ note: () => undefined, blockAt: () => 0 });
  const names = new Map([
    ["planks", 5],
    ["glass", 9],
    ["planks_stairs", 20],
    ["planks_stairs_east", 21],
    ["oak_door", 30],
  ]);
  const host = { jobs, blocks: () => names, survival: () => survival };
  const run = (tool: string, args: unknown) => runAgentCommand(9, tool, JSON.stringify(args), host);
  return { jobs, run };
}

const box = { from: { x: 0, y: 10, z: 0 }, to: { x: 2, y: 11, z: 2 } };

describe("runAgentCommand", () => {
  it("starts jobs and answers with their id", () => {
    const { jobs, run } = setup();
    expect(run("goto", { x: 1, y: 2, z: 3 })).toEqual({ job: 9, started: "Flying to 1 2 3." });
    expect(jobs.busy).toBe(true);
    expect(run("fill", { ...box, block: "planks" })).toMatchObject({ job: 9 });
    expect(run("clear", box)).toMatchObject({
      started: expect.stringContaining("18 blocks"),
      brushes: 11, // a 2-block cube and ten single blocks
    });
    expect(run("place", { blocks: [{ x: 0, y: 0, z: 0, block: "air" }] })).toMatchObject({
      job: 9,
    });
    expect(run("stop", {})).toEqual({ stopped: "Stopped." });
    expect(jobs.busy).toBe(false);
  });

  it("explains bad arguments instead of starting", () => {
    const { jobs, run } = setup();
    const errors = [
      run("goto", { x: 1.5, y: 2, z: 3 }),
      run("goto", { x: 1, y: 2 }),
      run("clear", { from: { x: 0, y: 0, z: 0 } }),
      run("clear", { from: { x: 0, y: 0, z: 0 }, to: { x: 99, y: 99, z: 99 } }),
      run("fill", { ...box, block: "lava" }),
      run("place", { blocks: [{ x: 0, y: 0, z: 0, block: "lava" }] }),
      run("place", { blocks: Array(MAX_PLACE + 1).fill({ x: 0, y: 0, z: 0, block: "air" }) }),
      run("place", { blocks: "lots" }),
      run("fly", {}),
      runAgentCommand(1, "goto", "{nope", {
        jobs,
        blocks: () => new Map(),
        survival: () => false,
      }),
    ].map((r) => r.error);
    expect(errors).toEqual([
      "goto needs x, y and z as whole numbers.",
      "goto needs x, y and z as whole numbers.",
      "clear needs from and to, each with x, y and z as whole numbers.",
      expect.stringContaining("at most 32,768"),
      "There's no block named lava. The blocks tool lists every name.",
      'Nothing was placed; fix these and send the batch again. Cell 0 names "lava", which isn\'t a block. The blocks tool lists every name.',
      "place takes at most 4,096 blocks; split it into several calls.",
      "place needs blocks: a list of {x, y, z, block}.",
      "This game can't run fly.",
      "The arguments weren't JSON.",
    ]);
    expect(jobs.busy).toBe(false);
  });

  it("clears when told to fill with air, and suggests names close to unknown ones", () => {
    const { run } = setup();
    expect(run("fill", { ...box, block: "air" }).started).toMatch(/^Clearing 18 blocks/);
    expect(run("fill", { ...box, block: "door" }).error).toBe(
      "There's no block named door. Close names: oak_door. The blocks tool lists every name.",
    );
    expect(run("fill", { ...box, block: "wooden_stairs" }).error).toBe(
      "There's no block named wooden_stairs. Close names: planks_stairs, planks_stairs_east. " +
        "The blocks tool lists every name.",
    );
    expect(run("fill", { ...box, block: "Glass" }).error).toBeUndefined();
    expect(run("fill", { ...box, block: "planks_stairs_eas" }).error).toMatch(
      /Close names: planks_stairs_east, planks_stairs, planks\./,
    );
  });

  it("names the cells a place batch got wrong, and places none of them", () => {
    const { jobs, run } = setup();
    const ok = { x: 0, y: 0, z: 0, block: "planks" };
    const blocks = [ok, { ...ok, block: "wooden_stairs" }, ok, { ...ok, x: 1.5 }];
    blocks.push(...Array(6).fill({ ...ok, block: "lava" }));
    expect(run("place", { blocks }).error).toBe(
      "Nothing was placed; fix these and send the batch again. " +
        "Cell 3 needs x, y and z as whole numbers. " +
        'Cell 1 names "wooden_stairs", which isn\'t a block; close names: planks_stairs, planks_stairs_east. ' +
        'Cells 4, 5, 6, 7, 8 and 1 more name "lava", which isn\'t a block. ' +
        "The blocks tool lists every name.",
    );
    expect(jobs.busy).toBe(false);
  });

  it("refuses jobs in survival", () => {
    const { run } = setup(true);
    expect(run("goto", { x: 1, y: 2, z: 3 })).toEqual({
      error: "Agents work in creative for now; survival is coming later.",
    });
    expect(run("undo", {}).error).toContain("creative");
  });
});
