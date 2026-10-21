import { describe, expect, it } from "vitest";
import {
  collisionBoxes,
  FULL_BLOCK,
  pistonFacing,
  SHAPE_BOXES,
  Shape,
  sideView,
  stairsFacing,
} from "./shapes";

describe("collisionBoxes", () => {
  it("gives slabs their lower half and non-solid blocks nothing", () => {
    expect(collisionBoxes(Shape.SLAB, true)).toEqual([[0, 0, 0, 1, 0.5, 1]]);
    expect(collisionBoxes(Shape.TORCH, false)).toEqual([]);
    expect(collisionBoxes(99, true)).toBe(FULL_BLOCK);
    expect(collisionBoxes(Shape.FLAT, true)).toEqual([[0, 0, 0, 1, 0.125, 1]]);
  });

  it("puts the tall half of north stairs toward -z", () => {
    expect(collisionBoxes(Shape.STAIRS_NORTH, true)[1]).toEqual([0, 0.5, 0, 1, 1, 0.5]);
  });
});

describe("stairsFacing", () => {
  it("rises away from the player", () => {
    expect(stairsFacing(40, 0)).toBe(40); // looking north
    expect(stairsFacing(40, Math.PI / 2)).toBe(43); // looking west
    expect(stairsFacing(40, Math.PI)).toBe(42);
    expect(stairsFacing(40, -Math.PI / 2)).toBe(41); // looking east
    expect(stairsFacing(40, 2 * Math.PI + 0.3)).toBe(40);
  });
});

describe("pistonFacing", () => {
  it("pushes where the player looks, each facing followed by its powered state", () => {
    expect(pistonFacing(60, 0, 0)).toBe(60); // north
    expect(pistonFacing(60, -Math.PI / 2, 0)).toBe(62); // east
    expect(pistonFacing(60, Math.PI, 0.5)).toBe(64); // south
    expect(pistonFacing(60, Math.PI / 2, 0)).toBe(66); // west
    expect(pistonFacing(60, 0, 1.2)).toBe(68); // up
    expect(pistonFacing(60, 2, -1.2)).toBe(70); // down
  });
});

describe("plant shapes", () => {
  it("stand on the floor inside their block", () => {
    for (const shape of [Shape.TUFT, Shape.FLOWER]) {
      const boxes = SHAPE_BOXES[shape];
      expect(boxes.length).toBeGreaterThan(1);
      expect(boxes.some((b) => b[1] === 0)).toBe(true);
      for (const b of boxes) expect(b.every((v) => v >= 0 && v <= 8)).toBe(true);
    }
  });
});

describe("sideView", () => {
  const shown = (shape: number) => {
    const view = sideView(shape);
    let n = 0;
    for (let y = 0; y < 16; y++) for (let x = 0; x < 16; x++) n += view(x, y) ? 1 : 0;
    return n;
  };

  it("cuts icons to the shape seen from the side, rows top-down", () => {
    expect(shown(Shape.CUBE)).toBe(256);
    expect(sideView(Shape.SLAB)(3, 12)).toBe(true);
    expect(sideView(Shape.SLAB)(3, 3)).toBe(false);
    expect(sideView(Shape.STAIRS_NORTH)(12, 3)).toBe(true);
    expect(sideView(Shape.STAIRS_NORTH)(3, 3)).toBe(false);
    expect(shown(Shape.TUFT)).toBeLessThan(128);
    expect(shown(Shape.FLOWER)).toBeLessThan(128);
  });

  it("shows flat blocks whole, since a sliver would be unreadable", () => {
    expect(shown(Shape.FLAT)).toBe(256);
  });
});
