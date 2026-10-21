import { describe, expect, it } from "vitest";
import { FAR_CELLS, FAR_REGION, farRegionMesh, type FarTile } from "./far-mesh";

const SAMPLES = FAR_CELLS * FAR_CELLS;
const RED = Uint8Array.of(0, 0, 0, 200, 0, 0); // block 1 is red

function flat(top: number): FarTile {
  return { tops: Array(SAMPLES).fill(top), blocks: Array(SAMPLES).fill(1) };
}

function tiles(...at: [number, number, FarTile][]) {
  const map = new Map(at.map(([x, z, t]) => [`${x},${z}`, t]));
  return (x: number, z: number) => map.get(`${x},${z}`);
}

const triangles = (m: { indices: ArrayLike<number> }) => m.indices.length / 3;

describe("farRegionMesh", () => {
  it("draws a tile as a surface through its samples, lit like a top face", () => {
    const m = farRegionMesh(tiles([0, 0, flat(100)]), 0, 0, RED);
    expect(triangles(m)).toBe((FAR_CELLS - 1) ** 2 * 2);
    for (let i = 1; i < m.positions.length; i += 3) expect(m.positions[i]).toBe(100);
    expect([...m.colors.subarray(0, 3)]).toEqual([200, 0, 0]);
  });

  it("joins tiles side by side, and reaches into the next region to close the gap", () => {
    const pair = tiles([0, 0, flat(100)], [1, 0, flat(100)]);
    expect(triangles(farRegionMesh(pair, 0, 0, RED))).toBe(
      (2 * FAR_CELLS - 1) * (FAR_CELLS - 1) * 2,
    );
    const across = tiles([FAR_REGION - 1, 0, flat(100)], [FAR_REGION, 0, flat(100)]);
    expect(triangles(farRegionMesh(across, 0, 0, RED))).toBe(FAR_CELLS * (FAR_CELLS - 1) * 2);
    expect(triangles(farRegionMesh(across, 1, 0, RED))).toBe((FAR_CELLS - 1) ** 2 * 2);
  });

  it("places samples at their cells' corners, relative to the region's corner", () => {
    const m = farRegionMesh(tiles([FAR_REGION + 1, 0, flat(7)]), 1, 0, RED);
    const step = 32 / FAR_CELLS;
    expect([...m.positions.subarray(3, 6)]).toEqual([32 + step, 7, 0]);
  });

  it("shades slopes darker than flat ground", () => {
    const slope: FarTile = {
      tops: Array.from({ length: SAMPLES }, (_, i) => 100 + (i % FAR_CELLS) * 4),
      blocks: Array(SAMPLES).fill(1),
    };
    const m = farRegionMesh(tiles([0, 0, slope]), 0, 0, RED);
    expect(m.colors[0]).toBeLessThan(200);
    expect(m.colors[0]).toBeGreaterThan(100);
  });
});
