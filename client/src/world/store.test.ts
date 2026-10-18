import { describe, expect, it } from "vitest";
import { chunkIndex, CHUNK_VOLUME, type Chunk } from "./chunk";
import { chunkKey } from "./coords";
import { paddedIndex } from "./mesher";
import { ChunkStore } from "./store";

const uniform = (fill: number): Chunk => ({ fill });
const bounds = { radius: 64, height: 32 }; // chunk columns -4..3, two layers

describe("ChunkStore", () => {
  it("is ready to mesh only once every neighbor inside the world is loaded", () => {
    const s = new ChunkStore(bounds);
    s.set({ x: 0, y: 0, z: 0 }, uniform(1));
    expect(s.ready(0, 0, 0)).toBe(false);
    for (const [x, y, z] of [
      [1, 0, 0],
      [-1, 0, 0],
      [0, 1, 0],
      [0, 0, 1],
      [0, 0, -1],
    ]) {
      s.set({ x, y, z }, uniform(0));
    }
    expect(s.ready(0, 0, 0)).toBe(true); // below y=0 is outside the world
    s.delete(chunkKey(1, 0, 0));
    expect(s.ready(0, 0, 0)).toBe(false);
  });

  it("treats the world border as finished", () => {
    const s = new ChunkStore(bounds);
    for (const [x, y, z] of [
      [3, 0, 0],
      [2, 0, 0],
      [3, 1, 0],
      [3, 0, 1],
      [3, 0, -1],
    ]) {
      s.set({ x, y, z }, uniform(0));
    }
    expect(s.ready(3, 0, 0)).toBe(true);
  });

  it("pads a chunk with the facing slabs of its neighbors", () => {
    const s = new ChunkStore(bounds);
    const blocks = new Uint16Array(CHUNK_VOLUME);
    blocks[chunkIndex(0, 5, 6)] = 7;
    s.set({ x: 1, y: 0, z: 0 }, { fill: 0, blocks });
    s.set({ x: 0, y: 0, z: 0 }, uniform(2));
    const v = s.padded(0, 0, 0);
    expect(v[paddedIndex(3, 3, 3)]).toBe(2);
    expect(v[paddedIndex(16, 5, 6)]).toBe(7);
    expect(v[paddedIndex(16, 5, 7)]).toBe(0);
    expect(v[paddedIndex(5, -1, 5)]).toBe(0); // below the world
  });
});
