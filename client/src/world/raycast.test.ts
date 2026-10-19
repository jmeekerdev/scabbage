import { describe, expect, it } from "vitest";
import { lookDirection, raycast } from "./raycast";

const wallAtX3 = (x: number) => x === 3;

describe("raycast", () => {
  it("hits the first targetable block and reports the face it entered", () => {
    const hit = raycast([0.5, 0.5, 0.5], [1, 0, 0], 5, wallAtX3);
    expect(hit).toEqual({ x: 3, y: 0, z: 0, nx: -1, ny: 0, nz: 0, distance: 2.5 });
  });

  it("misses blocks beyond the reach", () => {
    expect(raycast([0.5, 0.5, 0.5], [1, 0, 0], 2, wallAtX3)).toBeUndefined();
  });

  it("walks diagonals through every cell it crosses", () => {
    const floor = (_x: number, y: number) => y === -1;
    const hit = raycast([0.5, 1.5, 0.5], [0.6, -0.8, 0], 5, floor);
    expect(hit).toMatchObject({ y: -1, ny: 1 });
    expect(hit?.x).toBe(1);
  });

  it("works for negative directions", () => {
    const hit = raycast([0.5, 0.5, 0.5], [0, 0, -1], 5, (_x, _y, z) => z === -2);
    expect(hit).toMatchObject({ z: -2, nz: 1, distance: 1.5 });
  });

  it("ignores the block the ray starts in", () => {
    expect(raycast([0.5, 0.5, 0.5], [1, 0, 0], 0.4, () => true)).toBeUndefined();
  });
});

describe("lookDirection", () => {
  it("faces -z at yaw 0 and up at positive pitch", () => {
    const [x, y, z] = lookDirection(0, 0);
    expect([x, y, z].map((v) => Math.round(v * 1e9) / 1e9)).toEqual([-0, 0, -1]);
    expect(lookDirection(0, Math.PI / 4)[1]).toBeCloseTo(Math.SQRT1_2);
  });
});
