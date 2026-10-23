import { describe, expect, it } from "vitest";
import { History } from "./undo";

const [AIR, STONE, PLANKS] = [0, 1, 5];

/** A tiny world: a map from "x,y,z" to block, air elsewhere. */
function world(blocks: Record<string, number> = {}) {
  const at = new Map(Object.entries(blocks));
  return {
    blockAt: (x: number, y: number, z: number) => at.get(`${x},${y},${z}`) ?? AIR,
    set: (x: number, y: number, z: number, b: number) => at.set(`${x},${y},${z}`, b),
  };
}

describe("History", () => {
  it("undoes a fill by clearing what it placed", () => {
    const w = world();
    const h = new History();
    h.record(
      [
        { x: 0, y: 0, z: 0 },
        { x: 1, y: 0, z: 0 },
      ],
      w.blockAt,
    );
    w.set(0, 0, 0, PLANKS);
    w.set(1, 0, 0, PLANKS);
    expect(h.undoSteps(w.blockAt)).toEqual([
      { cube: { x: 0, y: 0, z: 0, size: 1 }, block: AIR },
      { cube: { x: 1, y: 0, z: 0, size: 1 }, block: AIR },
    ]);
  });

  it("undoes a clear by putting the old blocks back, bottom first", () => {
    const w = world({ "0,1,0": STONE, "0,0,0": PLANKS });
    const h = new History();
    h.record(
      [
        { x: 0, y: 1, z: 0 },
        { x: 0, y: 0, z: 0 },
      ],
      w.blockAt,
    );
    w.set(0, 1, 0, AIR);
    w.set(0, 0, 0, AIR);
    expect(h.undoSteps(w.blockAt)).toEqual([
      { cube: { x: 0, y: 0, z: 0, size: 1 }, block: PLANKS },
      { cube: { x: 0, y: 1, z: 0, size: 1 }, block: STONE },
    ]);
  });

  it("keeps the first block a cell held and skips cells already back", () => {
    const w = world({ "0,0,0": STONE, "1,0,0": STONE });
    const h = new History();
    h.record(
      [
        { x: 0, y: 0, z: 0 },
        { x: 1, y: 0, z: 0 },
      ],
      w.blockAt,
    );
    w.set(0, 0, 0, AIR);
    h.record([{ x: 0, y: 0, z: 0 }], w.blockAt);
    w.set(0, 0, 0, PLANKS);
    expect(h.undoSteps(w.blockAt)).toEqual([
      { cube: { x: 0, y: 0, z: 0, size: 1 }, block: AIR },
      { cube: { x: 0, y: 0, z: 0, size: 1 }, block: STONE },
    ]);
    expect(h.size).toBe(2);
  });
});

describe("History in brushes", () => {
  it("takes back a block of cells with one big brush", () => {
    const w = world();
    const h = new History();
    const cube = [0, 1].flatMap((x) => [0, 1].flatMap((y) => [0, 1].map((z) => ({ x, y, z }))));
    h.record(cube, w.blockAt);
    for (const c of cube) w.set(c.x, c.y, c.z, PLANKS);
    expect(h.undoSteps(w.blockAt)).toEqual([{ cube: { x: 0, y: 0, z: 0, size: 2 }, block: AIR }]);
  });
});
