import { describe, expect, it } from "vitest";
import type { Box } from "./shapes";
import { wireBoxes, Wiring } from "./wire";

const { WIRE, JOINS, HOLDS } = Wiring;

/** The wiring around a wire, from offsets written "dx,dy,dz"; the floor holds it unless given. */
function around(blocks: Record<string, number>, floor = true) {
  const all: Record<string, number> = floor ? { "0,-1,0": HOLDS, ...blocks } : blocks;
  return (dx: number, dy: number, dz: number) => all[`${dx},${dy},${dz}`] ?? 0;
}

const sorted = (boxes: Box[]) => boxes.map((b) => b.join(",")).sort();
const same = (got: Box[], want: Box[]) => expect(sorted(got)).toEqual(sorted(want));

const CENTRE: Box = [3, 0, 3, 5, 1, 5];
const EAST: Box = [5, 0, 3, 8, 1, 5];
const WEST: Box = [0, 0, 3, 3, 1, 5];
const SOUTH: Box = [3, 0, 5, 5, 1, 8];
const NORTH: Box = [3, 0, 0, 5, 1, 3];

describe("wireBoxes on the floor", () => {
  it("draws a cross when nothing is joined", () => {
    same(wireBoxes(around({})), [CENTRE, EAST, WEST, SOUTH, NORTH]);
  });

  it("runs straight through when joined on one side", () => {
    same(wireBoxes(around({ "1,0,0": WIRE })), [CENTRE, EAST, WEST]);
  });

  it("turns corners toward wire and devices", () => {
    same(wireBoxes(around({ "1,0,0": WIRE, "0,0,1": JOINS | HOLDS })), [CENTRE, EAST, SOUTH]);
  });

  it("climbs the face of a step to wire on top", () => {
    const step = around({ "1,0,0": HOLDS, "1,1,0": WIRE, "-1,0,0": WIRE });
    same(wireBoxes(step), [CENTRE, EAST, WEST, [7, 1, 3, 8, 8, 5]]);
  });

  it("doesn't climb when a block over it is in the way", () => {
    const blocked = around({ "1,0,0": HOLDS, "1,1,0": WIRE, "0,1,0": HOLDS, "-1,0,0": WIRE });
    same(wireBoxes(blocked), [CENTRE, EAST, WEST]);
  });

  it("reaches over the edge toward wire a step down", () => {
    same(wireBoxes(around({ "0,-1,-1": WIRE })), [CENTRE, NORTH, SOUTH]);
  });

  it("rises up the wall beside it toward wire overhead", () => {
    const rising = around({ "0,1,0": WIRE, "0,0,-1": HOLDS, "1,0,0": WIRE });
    same(wireBoxes(rising), [CENTRE, EAST, WEST, [3, 1, 0, 5, 8, 1]]);
  });
});

describe("wireBoxes off the floor", () => {
  it("runs up a wall between wire above and below", () => {
    const wall = around({ "0,1,0": WIRE, "0,-1,0": WIRE, "-1,0,0": HOLDS }, false);
    same(wireBoxes(wall), [
      [0, 3, 3, 1, 5, 5],
      [0, 5, 3, 1, 8, 5],
      [0, 0, 3, 1, 3, 5],
    ]);
  });

  it("runs along a wall toward wire beside it", () => {
    const wall = around({ "0,0,1": HOLDS, "1,0,0": WIRE }, false);
    same(wireBoxes(wall), [
      [3, 3, 7, 5, 5, 8],
      [5, 3, 7, 8, 5, 8],
      [0, 3, 7, 3, 5, 8],
    ]);
  });

  it("hangs as a post with nothing to lie on", () => {
    const hanging = around({ "0,1,0": WIRE }, false);
    same(wireBoxes(hanging), [
      [3, 3, 3, 5, 5, 5],
      [3, 5, 3, 5, 8, 5],
      [3, 0, 3, 5, 3, 5],
    ]);
  });
});
