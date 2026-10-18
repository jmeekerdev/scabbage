import { describe, expect, it } from "vitest";
import { chunkKey, type ChunkPos } from "./coords";
import { ChunkStreamer, columnsAround, type WorldBounds } from "./streamer";

const big: WorldBounds = { radius: 512, height: 128 };
const columns = (batch: ChunkPos[]) => new Set(batch.map((p) => `${p.x},${p.z}`));

/** Requests every batch, answering each at once, and returns all positions asked for. */
function drain(s: ChunkStreamer): ChunkPos[] {
  const all: ChunkPos[] = [];
  for (let b = s.nextBatch(); b; b = s.nextBatch()) {
    all.push(...b);
    s.received(b);
  }
  return all;
}

describe("columnsAround", () => {
  it("covers a circle, nearest first", () => {
    const cols = columnsAround(0, 0, 2, big);
    expect(cols[0]).toEqual({ x: 0, z: 0 });
    expect(cols).toContainEqual({ x: 2, z: 0 });
    expect(cols).not.toContainEqual({ x: 2, z: 2 });
    const dist = cols.map((c) => c.x * c.x + c.z * c.z);
    expect(dist).toEqual([...dist].sort((a, b) => a - b));
  });

  it("stops at the world border", () => {
    // A 32-block-wide world holds chunk columns -1 and 0 on each axis.
    const cols = columnsAround(0, 0, 4, { radius: 16, height: 128 });
    expect(cols).toHaveLength(4);
    expect(columnsAround(0, 0, 2, { radius: -1, height: 128 }).length).toBeGreaterThan(4);
  });
});

describe("ChunkStreamer", () => {
  it("requests whole columns, nearest first, one radius beyond the view", () => {
    const s = new ChunkStreamer(big, { batchSize: 8 });
    s.update(0, 0, 2);
    const first = s.nextBatch()!;
    expect(first).toHaveLength(8);
    expect(columns(first)).toEqual(new Set(["0,0"]));
    expect(first.map((p) => p.y)).toEqual([0, 1, 2, 3, 4, 5, 6, 7]);
    const all = drain(s);
    expect(columns(all).has("3,0")).toBe(true);
    expect(columns(all).has("4,0")).toBe(false);
  });

  it("limits batches in flight until they arrive", () => {
    const s = new ChunkStreamer(big, { batchSize: 4, maxInFlight: 2 });
    s.update(0, 0, 2);
    const a = s.nextBatch()!;
    expect(s.nextBatch()).toBeDefined();
    expect(s.nextBatch()).toBeUndefined();
    s.received(a);
    expect(s.nextBatch()).toBeDefined();
  });

  it("never requests a chunk twice", () => {
    const s = new ChunkStreamer(big, { batchSize: 16 });
    s.update(0, 0, 3);
    const all = drain(s).map((p) => chunkKey(p.x, p.y, p.z));
    expect(new Set(all).size).toBe(all.length);
    s.update(0, 0, 3);
    expect(s.nextBatch()).toBeUndefined();
  });

  it("drops chunks well behind a moving player and fetches new ones ahead", () => {
    const s = new ChunkStreamer(big, { batchSize: 64 });
    s.update(0, 0, 2);
    drain(s);
    const dropped = s.update(10, 0, 2);
    expect(dropped).toContain(chunkKey(-3, 0, 0));
    expect(dropped).not.toContain(chunkKey(8, 0, 0));
    expect(columns(drain(s)).has("12,0")).toBe(true);
    expect(s.has(chunkKey(-3, 0, 0))).toBe(false);
  });

  it("keeps chunks just past the edge so small moves don't thrash", () => {
    const s = new ChunkStreamer(big, { batchSize: 64 });
    s.update(0, 0, 2);
    drain(s);
    expect(s.update(1, 0, 2)).toEqual([]);
  });

  it("reset forgets everything, e.g. after reconnecting", () => {
    const s = new ChunkStreamer(big, { batchSize: 8, maxInFlight: 1 });
    s.update(0, 0, 2);
    s.nextBatch();
    s.reset();
    s.update(0, 0, 2);
    expect(columns(s.nextBatch()!)).toEqual(new Set(["0,0"]));
  });
});
