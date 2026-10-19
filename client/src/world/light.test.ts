import { describe, expect, it } from "vitest";
import { blockLight, cropToPadded, LIGHT_MARGIN, REGION_VOLUME, regionIndex } from "./light";
import { paddedIndex } from "./mesher";

const [AIR, STONE, TORCH] = [0, 1, 2];
const transparent = Uint8Array.from([1, 0, 1]);
const light = Uint8Array.from([0, 0, 11]);

function region(blocks: [number, number, number, number][]): Uint16Array {
  const r = new Uint16Array(REGION_VOLUME);
  for (const [x, y, z, b] of blocks) r[regionIndex(x, y, z)] = b;
  return r;
}

describe("blockLight", () => {
  it("fades by one per block from a torch", () => {
    const lit = blockLight(region([[5, 5, 5, TORCH]]), transparent, light);
    expect(lit[paddedIndex(5, 5, 5)]).toBe(11);
    expect(lit[paddedIndex(6, 5, 5)]).toBe(10);
    expect(lit[paddedIndex(7, 6, 5)]).toBe(8);
    expect(lit[paddedIndex(16, 5, 5)]).toBe(0);
  });

  it("reaches in from torches outside the chunk", () => {
    const lit = blockLight(region([[-LIGHT_MARGIN + 2, 5, 5, TORCH]]), transparent, light);
    expect(lit[paddedIndex(-1, 5, 5)]).toBe(11 - (LIGHT_MARGIN - 3));
  });

  it("is blocked by opaque blocks", () => {
    const walls: [number, number, number, number][] = [];
    for (let y = -LIGHT_MARGIN; y < 16 + LIGHT_MARGIN; y++) {
      for (let z = -LIGHT_MARGIN; z < 16 + LIGHT_MARGIN; z++) walls.push([6, y, z, STONE]);
    }
    const lit = blockLight(region([...walls, [5, 5, 5, TORCH]]), transparent, light);
    expect(lit[paddedIndex(6, 5, 5)]).toBe(0);
    expect(lit[paddedIndex(7, 5, 5)]).toBe(0);
  });
});

describe("cropToPadded", () => {
  it("keeps the chunk and its one-block border", () => {
    const r = region([
      [-1, 0, 16, STONE],
      [3, 4, 5, TORCH],
      [-2, 0, 0, STONE],
    ]);
    const padded = cropToPadded(r);
    expect(padded[paddedIndex(-1, 0, 16)]).toBe(STONE);
    expect(padded[paddedIndex(3, 4, 5)]).toBe(TORCH);
    expect(padded.filter((b) => b !== AIR)).toHaveLength(2);
  });
});
