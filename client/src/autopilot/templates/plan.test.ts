import { describe, expect, it } from "vitest";
import { placePlan, type Plan } from "./plan";

const names = new Map([
  ["planks", 8],
  ["glass", 10],
  ["brick", 11],
]);
const at = { x: 100, y: 10, z: 200 };

const plan = (
  layers: string[][],
  palette: Record<string, string> = { p: "planks", g: "glass" },
): Plan => ({
  name: "test",
  palette,
  layers,
});

function placed(p: Plan, facing: "north" | "east" | "south" | "west" = "north", material?: string) {
  const cells = placePlan(p, at, facing, material, names);
  if (typeof cells === "string") throw new Error(cells);
  return cells;
}

describe("placePlan", () => {
  it("reads rows left to right as seen from the front", () => {
    // Facing north, the viewer looks south at the front, so their left is east (+x).
    const cells = placed(plan([["gpp"]]));
    expect(cells.find((c) => c.block === 10)).toMatchObject({ x: 101, y: 10, z: 200 });
  });

  it("puts later rows behind the front and later layers above", () => {
    const cells = placed(
      plan([
        ["p", "g"],
        ["g", "_"],
      ]),
    );
    expect(cells).toContainEqual({ x: 100, y: 10, z: 200, block: 8 });
    expect(cells).toContainEqual({ x: 100, y: 10, z: 201, block: 10 });
    expect(cells).toContainEqual({ x: 100, y: 11, z: 200, block: 10 });
    expect(cells).toContainEqual({ x: 100, y: 11, z: 201, block: 0 });
  });

  it("leaves '.' cells alone", () => {
    expect(placed(plan([["p.p"]]))).toHaveLength(2);
  });

  it("turns to face each way, keeping the front's middle at the spot", () => {
    for (const facing of ["north", "east", "south", "west"] as const) {
      const cells = placed(plan([["ppp", "ppp"]]), facing);
      expect(cells, facing).toContainEqual({ ...at, block: 8 });
      expect(cells, facing).toHaveLength(6);
    }
    const east = placed(plan([["p", "g"]]), "east");
    expect(east.find((c) => c.block === 10)).toMatchObject({ x: 99, z: 200 }); // behind is west
  });

  it("swaps its most-used block for the material", () => {
    const blocks = placed(plan([["ppg"]]), "north", "brick").map((c) => c.block);
    expect(blocks.sort()).toEqual([10, 11, 11]);
  });

  it("says when a block or material is unknown", () => {
    expect(placePlan(plan([["p"]], { p: "lava" }), at, "north", undefined, names)).toMatch(
      /no block named lava/,
    );
    expect(placePlan(plan([["p"]]), at, "north", "lava", names)).toMatch(/no block named lava/);
  });
});
