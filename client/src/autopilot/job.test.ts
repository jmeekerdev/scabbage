import { describe, expect, it } from "vitest";
import { cubeCells } from "../player/brush";
import { changeSteps, MAX_JOB_BLOCKS, parseJob, tile } from "./job";

const at = { x: 10, y: 20, z: 30 };
const blockNamed = (name: string) => ({ stone: 1, planks: 5 })[name];

describe("parseJob", () => {
  it("reads goto coordinates, with ~ relative to the player", () => {
    expect(parseJob("goto", "1 2 3", at, blockNamed)).toEqual({
      kind: "goto",
      target: { x: 1, y: 2, z: 3 },
    });
    expect(parseJob("goto", "~ ~5 ~-2", at, blockNamed)).toEqual({
      kind: "goto",
      target: { x: 10, y: 25, z: 28 },
    });
  });

  it("reads a clear box in any corner order", () => {
    expect(parseJob("clear", "5 6 7 1 2 3", at, blockNamed)).toEqual({
      kind: "clear",
      box: { min: { x: 1, y: 2, z: 3 }, max: { x: 5, y: 6, z: 7 } },
    });
  });

  it("reads a fill box and its block", () => {
    expect(parseJob("fill", "0 0 0 ~ ~ ~ Planks", at, blockNamed)).toEqual({
      kind: "fill",
      box: { min: { x: 0, y: 0, z: 0 }, max: { x: 10, y: 20, z: 30 } },
      block: 5,
    });
  });

  it("refuses bad coordinates, unknown blocks and huge boxes", () => {
    expect(parseJob("goto", "1 2", at, blockNamed)).toMatch(/goto x y z/);
    expect(parseJob("goto", "1 two 3", at, blockNamed)).toMatch(/goto x y z/);
    expect(parseJob("fill", "0 0 0 1 1 1 lava", at, blockNamed)).toMatch(/no block named lava/i);
    expect(parseJob("fill", "0 0 0 1 1 1", at, blockNamed)).toMatch(/fill x1 y1 z1/);
    expect(parseJob("clear", "0 0 0 32 31 31", at, blockNamed)).toMatch(
      String(MAX_JOB_BLOCKS.toLocaleString("en-US")),
    );
    expect(parseJob("clear", "0 0 0 31 31 31", at, blockNamed)).not.toBeTypeOf("string");
  });
});

describe("tile", () => {
  const box = { min: { x: 0, y: 0, z: 0 }, max: { x: 6, y: 4, z: 3 } };
  const cells = (cubes: ReturnType<typeof tile>) =>
    cubes.flatMap(cubeCells).map((c) => `${c.x},${c.y},${c.z}`);

  it("covers the box exactly once, without spilling over", () => {
    const covered = cells(tile(box, false));
    expect(covered.length).toBe(7 * 5 * 4);
    expect(new Set(covered).size).toBe(covered.length);
  });

  it("uses the biggest brushes that fit", () => {
    const cubes = tile({ min: { x: 0, y: 0, z: 0 }, max: { x: 5, y: 2, z: 2 } }, false);
    expect(cubes.map((c) => c.size)).toEqual([3, 3]);
  });

  it("goes bottom-up for building and top-down for clearing", () => {
    const ys = (cubes: ReturnType<typeof tile>) => cubes.map((c) => c.y);
    const up = ys(tile(box, false));
    const down = ys(tile(box, true));
    expect(up).toEqual([...up].sort((a, b) => a - b));
    expect(down).toEqual([...down].sort((a, b) => b - a));
  });

  it("snakes along rows so each cube is next to the last", () => {
    const flat = tile({ min: { x: 0, y: 0, z: 0 }, max: { x: 2, y: 0, z: 1 } }, false);
    expect(flat.map((c) => [c.x, c.z])).toEqual([
      [0, 0],
      [1, 0],
      [2, 0],
      [2, 1],
      [1, 1],
      [0, 1],
    ]);
  });
});

describe("changeSteps", () => {
  const [AIR, STONE, PLANKS] = [0, 1, 5];
  const now = new Map([
    ["0,9,0", STONE],
    ["0,11,0", PLANKS],
    ["5,5,5", PLANKS],
  ]);
  const blockAt = (x: number, y: number, z: number) => now.get(`${x},${y},${z}`) ?? AIR;

  it("clears what's in the way top-down, then fills bottom-up, skipping cells already right", () => {
    const steps = changeSteps(
      [
        { x: 0, y: 10, z: 0, block: PLANKS },
        { x: 1, y: 10, z: 0, block: PLANKS },
        { x: 0, y: 9, z: 0, block: AIR },
        { x: 0, y: 11, z: 0, block: STONE },
        { x: 5, y: 5, z: 5, block: PLANKS },
      ],
      blockAt,
    );
    expect(steps.map((s) => [s.cube.x, s.cube.y, s.block])).toEqual([
      [0, 11, AIR],
      [0, 9, AIR],
      [0, 10, PLANKS],
      [1, 10, PLANKS],
      [0, 11, STONE],
    ]);
  });
});
