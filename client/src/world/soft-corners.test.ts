import { describe, expect, it } from "vitest";
import { PADDED_VOLUME, paddedIndex } from "./mesher";
import { cornerShift, Smooth, SoftCorners } from "./soft-corners";

const GROUND = 1;
const BRICK = 2;
const WATER = 3;
const kinds = Uint8Array.from([Smooth.EMPTY, Smooth.NATURAL, Smooth.BUILT, Smooth.EMPTY]);

function corners(blocks: [number, number, number, number][], limit = 0.5): SoftCorners {
  const v = new Uint16Array(PADDED_VOLUME);
  for (const [x, y, z, b] of blocks) v[paddedIndex(x, y, z)] = b;
  return new SoftCorners(v, kinds, limit);
}

const ground = (y: number, b = GROUND): [number, number, number, number][] =>
  [4, 5, 6, 7].flatMap((x) =>
    [4, 5, 6, 7].map((z): [number, number, number, number] => [x, y, z, b]),
  );

describe("SoftCorners", () => {
  it("leaves corners on flat ground in place, facing up", () => {
    const c = corners(ground(3));
    expect(c.shift(6, 4, 6)).toEqual([0, 0, 0]);
    expect(c.normal(6, 4, 6)).toEqual([0, 127, 0]);
  });

  it("slides a lone block's corner toward its middle, in eighths", () => {
    const c = corners([[5, 5, 5, GROUND]]);
    expect(c.shift(6, 6, 6)).toEqual([-3, -3, -3]);
    expect(c.shift(5, 5, 5)).toEqual([3, 3, 3]);
  });

  it("tilts the normal away from the ground around a step", () => {
    const c = corners([...ground(3), [5, 4, 5, GROUND], [5, 4, 6, GROUND]]);
    const [x, y] = c.normal(6, 5, 6);
    expect(x).toBeGreaterThan(0);
    expect(y).toBeGreaterThan(0);
  });

  it("keeps corners still where they touch a built block", () => {
    const c = corners([
      [5, 5, 5, GROUND],
      [6, 5, 5, BRICK],
    ]);
    expect(c.shift(6, 6, 6)).toEqual([0, 0, 0]);
    expect(c.normal(6, 6, 6)).toEqual([0, 0, 0]);
  });

  it("treats water as open, so shores soften", () => {
    expect(
      corners([
        [5, 5, 5, GROUND],
        [6, 5, 5, WATER],
      ]).shift(6, 6, 6),
    ).toEqual([-3, -3, -3]);
  });

  it("moves at most the look's limit", () => {
    expect(corners([[5, 5, 5, GROUND]], 0.25).shift(6, 6, 6)).toEqual([-2, -2, -2]);
  });
});

describe("cornerShift", () => {
  const lone = (x: number, y: number, z: number) => (x === 5 && y === 5 && z === 5 ? 1 : 0);

  it("moves a world corner as far as the mesher does, in blocks", () => {
    expect(cornerShift(6, 6, 6, lone, 0.5)).toEqual([-3 / 8, -3 / 8, -3 / 8]);
    expect(cornerShift(5, 5, 5, lone, 0.5)).toEqual([3 / 8, 3 / 8, 3 / 8]);
  });

  it("leaves corners next to built or unknown blocks in place", () => {
    expect(cornerShift(6, 6, 6, (x, y, z) => (lone(x, y, z) ? 1 : 2), 0.5)).toEqual([0, 0, 0]);
  });
});
