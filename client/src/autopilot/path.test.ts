import { describe, expect, it } from "vitest";
import { flyPath } from "./path";
import type { Vec } from "./route";

/** Open air above y=0, plus any walls given as cell tests. */
const world =
  (...walls: ((x: number, y: number, z: number) => boolean)[]) =>
  (x: number, y: number, z: number) =>
    y >= 0 && y < 60 && !walls.some((w) => w(x, y, z));

const near = (goal: Vec, within: number) => (feet: Vec) =>
  Math.hypot(feet.x - goal.x, feet.y - goal.y, feet.z - goal.z) <= within;

describe("flyPath", () => {
  it("goes straight through open air", () => {
    const goal = { x: 30, y: 10, z: 0 };
    const path = flyPath({ x: 1.5, y: 9, z: 1.5 }, goal, near(goal, 3), world())!;
    expect(path.length).toBeLessThanOrEqual(2);
    expect(near(goal, 3)(path.at(-1)!)).toBe(true);
  });

  it("finds the gap in a wall", () => {
    const wall = (x: number, y: number, z: number) =>
      x >= 15 && x < 18 && !(z >= 30 && z < 36 && y < 12);
    const goal = { x: 30, y: 6, z: 0 };
    const path = flyPath({ x: 1.5, y: 6, z: 1.5 }, goal, near(goal, 3), world(wall))!;
    expect(path).toBeDefined();
    expect(path.some((p) => p.z >= 30 && p.z < 36)).toBe(true);
  });

  it("gets in under a roof from the side", () => {
    const roof = (x: number, y: number, z: number) =>
      y >= 20 && y < 24 && x > -30 && x < 30 && z > -30 && z < 30;
    const goal = { x: 0, y: 3, z: 0 };
    const path = flyPath({ x: 0, y: 40, z: 0 }, goal, near(goal, 4), world(roof))!;
    expect(path).toBeDefined();
    expect(near(goal, 4)(path.at(-1)!)).toBe(true);
  });

  it("gives up when there's no way, or the search runs too long", () => {
    const box = (x: number, y: number, z: number) =>
      Math.max(Math.abs(x - 30), Math.abs(y - 10), Math.abs(z)) <= 6 &&
      Math.max(Math.abs(x - 30), Math.abs(y - 10), Math.abs(z)) >= 4;
    const goal = { x: 30, y: 10, z: 0 };
    expect(flyPath({ x: 1.5, y: 9, z: 1.5 }, goal, near(goal, 2), world(box))).toBeUndefined();
    const far = { x: 3000, y: 10, z: 0 };
    expect(flyPath({ x: 1.5, y: 9, z: 1.5 }, far, near(far, 2), world(), 500)).toBeUndefined();
  });
});
