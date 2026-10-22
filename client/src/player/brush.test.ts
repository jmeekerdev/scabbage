import { describe, expect, it } from "vitest";
import { brushCube, cubeCells, MAX_BRUSH, nextBrush } from "./brush";

const onTop = { x: 4, y: 9, z: -2, nx: 0, ny: 1, nz: 0, distance: 2 };
const onEastSide = { x: 4, y: 9, z: -2, nx: 1, ny: 0, nz: 0, distance: 2 };

describe("brushCube", () => {
  it("is the hit block itself at size 1", () => {
    expect(brushCube(onTop, 1, false)).toEqual({ x: 4, y: 9, z: -2, size: 1 });
    expect(brushCube(onTop, 1, true)).toEqual({ x: 4, y: 10, z: -2, size: 1 });
  });

  it("digs into the face and builds out from it", () => {
    expect(brushCube(onTop, 3, false)).toMatchObject({ y: 7 });
    expect(brushCube(onTop, 3, true)).toMatchObject({ y: 10 });
    expect(brushCube(onEastSide, 3, false)).toMatchObject({ x: 2 });
    expect(brushCube(onEastSide, 3, true)).toMatchObject({ x: 5 });
    expect(brushCube({ ...onTop, ny: -1 }, 3, true)).toMatchObject({ y: 6 });
  });

  it("snaps across the face to the brush's grid in x and z, and centers on the hit in y", () => {
    expect(brushCube(onTop, 3, false)).toMatchObject({ x: 3, z: -3 });
    expect(brushCube(onEastSide, 3, true)).toMatchObject({ y: 8, z: -3 });
  });
});

describe("cubeCells", () => {
  it("lists every block of the cube", () => {
    const cells = cubeCells({ x: 1, y: 2, z: 3, size: 2 });
    expect(cells).toHaveLength(8);
    expect(cells).toContainEqual({ x: 2, y: 3, z: 4 });
  });
});

describe("nextBrush", () => {
  it("grows up to an old block, then starts over", () => {
    expect(nextBrush(1)).toBe(2);
    expect(nextBrush(MAX_BRUSH)).toBe(1);
  });
});
