import { describe, expect, it } from "vitest";
import { placeTemplate, TEMPLATES, type Facing, type Part, type Size } from "./templates";

const key = (c: { x: number; y: number; z: number }) => `${c.x},${c.y},${c.z}`;
const names = new Map([
  ["log", 6],
  ["planks", 8],
  ["glass", 10],
  ["stone", 1],
  ["cobblestone", 9],
  ["brick", 11],
]);

function cellsOf(name: string, size?: Partial<Size>) {
  const t = TEMPLATES.get(name)!;
  return t.cells({ ...t.size, ...size });
}

const count = (cells: { part: Part }[], part: Part) => cells.filter((c) => c.part === part).length;

describe("templates", () => {
  it("keep every cell inside the asked-for size, each once", () => {
    for (const [name, t] of TEMPLATES) {
      const { w, d, h } = t.size;
      const cells = t.cells(t.size);
      expect(new Set(cells.map(key)).size, name).toBe(cells.length);
      for (const c of cells) {
        expect(c.x >= 0 && c.x < w && c.y >= 0 && c.y < h && c.z >= 0 && c.z < d, name).toBe(true);
      }
    }
  });

  it("make a solid wall and floor", () => {
    expect(cellsOf("wall", { w: 9, d: 1, h: 6 })).toHaveLength(54);
    expect(cellsOf("floor", { w: 4, d: 5, h: 1 })).toHaveLength(20);
    expect(count(cellsOf("wall"), "main")).toBe(cellsOf("wall").length);
  });

  it("make stairs that rise one block a step, with headroom cleared above", () => {
    const cells = cellsOf("stairs", { w: 1, d: 3, h: 3 });
    const solid = cells.filter((c) => c.part === "main").map(key);
    expect(solid.sort()).toEqual(["0,0,0", "0,0,1", "0,0,2", "0,1,1", "0,1,2", "0,2,2"].sort());
    expect(cells.some((c) => c.part === "air" && c.z === 0 && c.y === 1)).toBe(true);
  });

  it("leave a doorway 6 tall and at least 2 wide in the cabin's and tower's front", () => {
    for (const name of ["cabin", "tower"]) {
      const cells = cellsOf(name);
      const door = cells.filter((c) => c.z === 0 && c.part === "air");
      const xs = new Set(door.map((c) => c.x));
      const ys = new Set(door.map((c) => c.y));
      expect(xs.size, name).toBeGreaterThanOrEqual(2);
      expect(ys.size, name).toBe(6);
      expect(Math.min(...ys), name).toBe(1); // starts just above the floor
    }
  });

  it("give the cabin glass windows, a planks roof and a floor", () => {
    const cells = cellsOf("cabin");
    expect(count(cells, "window")).toBeGreaterThan(0);
    expect(count(cells, "roof")).toBeGreaterThan(0);
    expect(count(cells, "floor")).toBe(9 * 9);
  });

  it("clear the inside of the cabin and the tower", () => {
    for (const name of ["cabin", "tower"]) {
      const t = TEMPLATES.get(name)!;
      const inside = { x: Math.floor(t.size.w / 2), y: 2, z: Math.floor(t.size.d / 2) };
      const cell = t.cells(t.size).find((c) => key(c) === key(inside));
      expect(cell?.part, name).toBe("air");
    }
  });
});

describe("placeTemplate", () => {
  const at = { x: 100, y: 10, z: 200 };

  it("puts the front middle at the spot, building away from where it faces", () => {
    const placed = placeTemplate("floor", { w: 3, d: 2, h: 1 }, at, "north", undefined, names);
    if (typeof placed === "string") throw new Error(placed);
    const xs = placed.map((c) => c.x);
    const zs = placed.map((c) => c.z);
    expect([Math.min(...xs), Math.max(...xs)]).toEqual([99, 101]);
    expect([Math.min(...zs), Math.max(...zs)]).toEqual([200, 201]); // north faces -z, so it runs +z
    expect(placed.every((c) => c.y === 10 && c.block === 8)).toBe(true);
  });

  it.each<[Facing, number, number]>([
    ["east", -1, 0],
    ["south", 0, -1],
    ["west", 1, 0],
  ])("turns to face %s", (facing, dx, dz) => {
    const placed = placeTemplate("floor", { w: 1, d: 3, h: 1 }, at, facing, undefined, names);
    if (typeof placed === "string") throw new Error(placed);
    const far = placed.find((c) => c.x === at.x + 2 * dx && c.z === at.z + 2 * dz);
    expect(far).toBeDefined();
  });

  it("keeps the cell count whichever way it faces", () => {
    const counts = (["north", "east", "south", "west"] as const).map((f) => {
      const placed = placeTemplate("cabin", undefined, at, f, undefined, names);
      return typeof placed === "string" ? placed : new Set(placed.map(key)).size;
    });
    expect(new Set(counts).size).toBe(1);
  });

  it("swaps the main material, leaving windows and air alone", () => {
    const placed = placeTemplate("cabin", undefined, at, "north", "brick", names);
    if (typeof placed === "string") throw new Error(placed);
    const blocks = new Set(placed.map((c) => c.block));
    expect(blocks).toEqual(new Set([11, 8, 10, 0]));
  });

  it("refuses unknown templates and materials and sizes it can't make", () => {
    expect(placeTemplate("castle", undefined, at, "north", undefined, names)).toMatch(
      /no template called castle.*cabin/i,
    );
    expect(placeTemplate("wall", undefined, at, "north", "lava", names)).toMatch(
      /no block named lava/i,
    );
    expect(placeTemplate("cabin", { w: 3, d: 9, h: 13 }, at, "north", undefined, names)).toMatch(
      /at least 7 wide/,
    );
    expect(placeTemplate("wall", { w: 33, d: 1, h: 1 }, at, "north", undefined, names)).toMatch(
      /at most 32/,
    );
  });
});
