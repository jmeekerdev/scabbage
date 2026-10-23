import { describe, expect, it } from "vitest";
import { blockLight, cropToPadded, LIGHT_MARGIN, REGION_VOLUME, regionIndex } from "./light";
import { paddedIndex } from "./mesher";
import { CHUNK_SIZE, METER } from "./constants";

const [AIR, STONE, TORCH, STAIRS] = [0, 1, 2, 3];
const transparent = Uint8Array.from([1, 0, 1, 1]);
const passes = Uint8Array.from([1, 0, 1, 0]);
const light = Uint8Array.from([0, 0, 11, 0]);
const lit = (r: Uint16Array, emits = light) => blockLight(r, { transparent, passes, emits });

function region(blocks: [number, number, number, number][]): Uint16Array {
  const r = new Uint16Array(REGION_VOLUME);
  for (const [x, y, z, b] of blocks) r[regionIndex(x, y, z)] = b;
  return r;
}

describe("blockLight", () => {
  it("fades by one per block from a torch", () => {
    const out = lit(region([[5, 5, 5, TORCH]]));
    expect(out[paddedIndex(5, 5, 5)]).toBe(11);
    expect(out[paddedIndex(6, 5, 5)]).toBe(10);
    expect(out[paddedIndex(7, 6, 5)]).toBe(8);
    expect(out[paddedIndex(16, 5, 5)]).toBe(0);
  });

  it("reaches about ten meters from the brightest light", () => {
    const bright = Uint8Array.from([0, 0, 255]);
    const out = lit(region([[0, 5, 5, TORCH]]), bright);
    expect(out[paddedIndex(9 * METER, 5, 5)]).toBeGreaterThan(0);
    expect(out[paddedIndex(CHUNK_SIZE, 5, 5)]).toBe(0);
  });

  it("reaches in from torches outside the chunk", () => {
    const out = lit(region([[-8, 5, 5, TORCH]]));
    expect(out[paddedIndex(-1, 5, 5)]).toBe(11 - 7);
  });

  it("is blocked by opaque blocks", () => {
    const walls = wallAt(6, STONE);
    const out = lit(region([...walls, [5, 5, 5, TORCH]]));
    expect(out[paddedIndex(6, 5, 5)]).toBe(0);
    expect(out[paddedIndex(7, 5, 5)]).toBe(0);
  });

  it("stops at stairs and slabs, which take the light beside them for their own faces", () => {
    const out = lit(region([...wallAt(6, STAIRS), [5, 5, 5, TORCH]]));
    expect(out[paddedIndex(6, 5, 5)]).toBe(10);
    expect(out[paddedIndex(7, 5, 5)]).toBe(0);
  });
});

/** A wall across the whole region at x. */
function wallAt(x: number, block: number): [number, number, number, number][] {
  const walls: [number, number, number, number][] = [];
  for (let y = -LIGHT_MARGIN; y < CHUNK_SIZE + LIGHT_MARGIN; y++) {
    for (let z = -LIGHT_MARGIN; z < CHUNK_SIZE + LIGHT_MARGIN; z++) walls.push([x, y, z, block]);
  }
  return walls;
}

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
