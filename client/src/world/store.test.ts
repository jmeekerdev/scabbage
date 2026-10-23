import { describe, expect, it } from "vitest";
import { CHUNK_SIZE as S } from "./constants";
import { chunkIndex, CHUNK_VOLUME, type Chunk } from "./chunk";
import { chunkKey } from "./coords";
import { LIGHT_MARGIN, REGION_VOLUME, regionIndex } from "./light";
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
    expect(v[paddedIndex(S, 5, 6)]).toBe(7);
    expect(v[paddedIndex(S, 5, 7)]).toBe(0);
    expect(v[paddedIndex(5, -1, 5)]).toBe(0); // below the world
  });

  it("pads corners and edges from diagonal neighbors too", () => {
    const s = new ChunkStore(bounds);
    s.set({ x: 1, y: 1, z: -1 }, uniform(4));
    expect(s.padded(0, 0, 0)[paddedIndex(S, S, -1)]).toBe(4);
  });

  it("copies a light region reaching LIGHT_MARGIN into every neighbor", () => {
    const s = new ChunkStore(bounds);
    s.set({ x: -1, y: 0, z: 0 }, uniform(3));
    const r = s.region(0, 0, 0, LIGHT_MARGIN);
    expect(r).toHaveLength(REGION_VOLUME);
    expect(r[regionIndex(-LIGHT_MARGIN, 5, 5)]).toBe(3);
    expect(r[regionIndex(0, 5, 5)]).toBe(0);
  });

  it("looks up blocks by world position", () => {
    const s = new ChunkStore(bounds);
    const blocks = new Uint16Array(CHUNK_VOLUME);
    blocks[chunkIndex(S - 1, 2, 0)] = 9;
    s.set({ x: -1, y: 1, z: 0 }, { fill: 0, blocks });
    expect(s.blockAt(-1, S + 2, 0)).toBe(9);
    expect(s.blockAt(-2, S + 2, 0)).toBe(0);
    expect(s.blockAt(0, S + 2, 0)).toBeUndefined(); // not loaded
  });

  it("sets blocks in loaded chunks only, expanding uniform chunks", () => {
    const s = new ChunkStore(bounds);
    s.set({ x: 0, y: 0, z: 0 }, uniform(1));
    expect(s.setBlock(3, 4, 5, 7)).toBe(true);
    expect(s.blockAt(3, 4, 5)).toBe(7);
    expect(s.blockAt(3, 4, 6)).toBe(1);
    expect(s.setBlock(-1, 0, 0, 7)).toBe(false);
  });

  describe("topAt", () => {
    const bounds = { radius: 64, height: 2 * S };
    it("finds the highest block in a column across layers, or -1 for none", () => {
      const s = new ChunkStore(bounds);
      s.set({ x: 0, y: 0, z: 0 }, uniform(1));
      s.set({ x: 0, y: 1, z: 0 }, uniform(0));
      expect(s.topAt(5, 7)).toBe(S - 1);
      s.setBlock(5, S + 4, 7, 2);
      expect(s.topAt(5, 7)).toBe(S + 4);
      expect(s.topAt(6, 7)).toBe(S - 1);
      expect(s.topAt(S + 1, 0)).toBe(-1); // nothing loaded there
    });

    it("drops when the top block is broken and rises when a chunk arrives above", () => {
      const s = new ChunkStore(bounds);
      s.set({ x: 0, y: 0, z: 0 }, uniform(0));
      s.setBlock(1, 9, 1, 3);
      expect(s.topAt(1, 1)).toBe(9);
      s.setBlock(1, 9, 1, 0);
      expect(s.topAt(1, 1)).toBe(-1);
      s.set({ x: 0, y: 1, z: 0 }, uniform(4));
      expect(s.topAt(1, 1)).toBe(2 * S - 1);
    });

    it("counts each change, so views of the tops know when to look again", () => {
      const s = new ChunkStore(bounds);
      const v0 = s.version;
      s.set({ x: 0, y: 0, z: 0 }, uniform(0));
      s.setBlock(1, 1, 1, 3);
      s.delete(chunkKey(0, 0, 0));
      expect(s.version).toBe(v0 + 3);
    });
  });
});
