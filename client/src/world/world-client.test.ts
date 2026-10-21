import { describe, expect, it } from "vitest";
import { CHUNK_SIZE } from "./constants";
import { chunkKey, type ChunkPos } from "./coords";
import { REGION_VOLUME } from "./light";
import { PADDED_VOLUME, type ChunkMesh } from "./mesher";
import { WorldClient, type ChunkView, type Mesher } from "./world-client";

const AIR = Uint8Array.of(2, 1, 0);
const STONE = Uint8Array.of(2, 1, 1);
const TORCH = 2;
const WIRE = 3; // see-through and dark, like a wire turning on or off
const bounds = { radius: 64, height: 64 }; // chunk columns -2..1, two layers

class FakeMesher implements Mesher {
  jobs: { volume: Uint16Array; done: (m: ChunkMesh) => void }[] = [];
  constructor(private readonly slots = 100) {}
  capacity() {
    return this.slots - this.jobs.length;
  }
  mesh(volume: Uint16Array, done: (m: ChunkMesh) => void) {
    this.jobs.push({ volume, done });
  }
  finishAll() {
    const jobs = this.jobs;
    this.jobs = [];
    for (const j of jobs) j.done({} as ChunkMesh);
  }
}

class FakeView implements ChunkView {
  shown = new Map<string, ChunkMesh>();
  show(key: string, _pos: ChunkPos, mesh: ChunkMesh) {
    this.shown.set(key, mesh);
  }
  remove(key: string) {
    this.shown.delete(key);
  }
  clear() {
    this.shown.clear();
  }
}

function setup(slots?: number) {
  const sent: ChunkPos[][] = [];
  const mesher = new FakeMesher(slots);
  const view = new FakeView();
  const covered = new Set<string>();
  const cover = (x: number, z: number, yes: boolean) =>
    yes ? covered.add(`${x},${z}`) : covered.delete(`${x},${z}`);
  const world = new WorldClient({ send: (b) => sent.push(b), mesher, view, cover });
  world.start(bounds);
  world.setLights(Uint8Array.of(0, 0, 11, 0), Uint8Array.of(1, 0, 1, 1));
  return { sent, mesher, view, world, covered };
}

/** Answers every outstanding request with stone in the bottom layer of chunks and air above. */
function answer(world: WorldClient, sent: ChunkPos[][]) {
  for (const batch of sent.splice(0)) {
    world.receive(batch.map((pos) => ({ pos, data: pos.y === 0 ? STONE : AIR })));
  }
}

describe("WorldClient", () => {
  it("requests chunks around the player", () => {
    const { sent, world } = setup();
    world.tick(0, 0, 1);
    expect(sent.length).toBeGreaterThan(0);
    expect(sent[0][0]).toEqual({ x: 0, y: 0, z: 0 });
  });

  it("requests chunks again after restarting where it already was, e.g. on reconnecting", () => {
    const { sent, world } = loaded();
    world.start(bounds);
    world.tick(0, 0, 1);
    expect(sent.length).toBeGreaterThan(0);
    expect(sent[0][0]).toEqual({ x: 0, y: 0, z: 0 });
  });

  it("meshes chunks within view once their neighbors arrive, and shows the result", () => {
    const { sent, mesher, view, world } = setup();
    world.tick(0, 0, 1);
    while (sent.length) {
      answer(world, sent);
      world.tick(0, 0, 1);
    }
    expect(mesher.jobs.length).toBeGreaterThan(0);
    mesher.finishAll();
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(true);
    // Two layers in five columns (center plus four neighbors at view distance 1).
    expect(view.shown.size).toBe(10);
  });

  it("never runs more mesh jobs than the mesher has room for", () => {
    const { sent, mesher, world } = setup(2);
    world.tick(0, 0, 1);
    while (sent.length) {
      answer(world, sent);
      world.tick(0, 0, 1);
    }
    expect(mesher.jobs.length).toBe(2);
  });

  it("drops meshes from a previous connection", () => {
    const { sent, mesher, view, world } = setup();
    world.tick(0, 0, 1);
    while (sent.length) {
      answer(world, sent);
      world.tick(0, 0, 1);
    }
    world.start(bounds);
    mesher.finishAll();
    expect(view.shown.size).toBe(0);
  });

  it("removes chunks the player has left behind", () => {
    const big = { radius: 1536, height: 64 };
    const { sent, mesher, view, world } = setup();
    world.start(big);
    world.tick(0, 0, 1);
    while (sent.length) {
      answer(world, sent);
      world.tick(0, 0, 1);
    }
    mesher.finishAll();
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(true);
    world.tick(CHUNK_SIZE * 20, 0, 1);
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(false);
  });

  it("tells which columns have every layer drawn, until they are dropped", () => {
    const env = loaded();
    expect([...env.covered].sort()).toEqual(["-1,0", "0,-1", "0,0", "0,1", "1,0"]);
    env.world.tick(CHUNK_SIZE * 20, 0, 1);
    expect(env.covered.size).toBe(0);
  });

  it("re-meshes a changed chunk, and the neighbors whose border holds the block", () => {
    const { sent, mesher, view, world } = loaded();
    world.setBlock(0, CHUNK_SIZE - 1, 5, 0); // top layer of chunk 0,0,0
    expect(world.blockAt(0, CHUNK_SIZE - 1, 5)).toBe(0);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(4); // itself, the -x neighbor, the one above and their shared edge
    expect(sent.length).toBe(0);
    expect(view.shown.size).toBe(10);
  });

  it("re-meshes every chunk a new light reaches, with a light region", () => {
    const { mesher, world } = loaded();
    world.setBlock(5, 5, 5, TORCH);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(3); // its own chunk and the -x and -z ones the light reaches
    expect(mesher.jobs.every((j) => j.volume.length === REGION_VOLUME)).toBe(true);
  });

  it("re-meshes around a light that was removed, without a light region", () => {
    const { mesher, world } = loaded();
    world.setBlock(5, 5, 5, TORCH);
    world.tick(0, 0, 1);
    mesher.finishAll();
    world.setBlock(5, 5, 5, 0);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(3);
    expect(mesher.jobs.every((j) => j.volume.length === PADDED_VOLUME)).toBe(true);
  });

  it("re-meshes only the chunk itself for a change that leaves the light alone", () => {
    const { mesher, world } = loaded();
    world.setBlock(5, 5, 5, TORCH);
    world.setBlock(6, 5, 5, 0);
    world.tick(0, 0, 1);
    mesher.finishAll();
    world.setBlock(6, 5, 5, WIRE);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(1);
    expect(mesher.jobs[0]!.volume.length).toBe(REGION_VOLUME); // still lit by the torch
  });

  it("re-meshes every chunk the light reaches when a block starts blocking it", () => {
    const { mesher, world } = loaded();
    world.setBlock(5, 5, 5, TORCH);
    world.setBlock(6, 5, 5, 0);
    world.tick(0, 0, 1);
    mesher.finishAll();
    world.setBlock(6, 5, 5, 1);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(10); // every loaded neighbor: it may shade light from any side
  });

  it("keeps showing a stale chunk until it arrives again, then re-meshes it", () => {
    const { sent, mesher, view, world } = loaded();
    world.refetch([{ x: 0, y: 0, z: 0 }]);
    world.tick(0, 0, 1);
    expect(sent).toEqual([[{ x: 0, y: 0, z: 0 }]]);
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(true);
    expect(world.blockAt(0, 5, 0)).toBe(1);
    world.receive([{ pos: { x: 0, y: 0, z: 0 }, data: AIR }]);
    world.tick(0, 0, 1);
    expect(world.blockAt(0, 5, 0)).toBe(0);
    expect(mesher.jobs.length).toBeGreaterThan(0);
  });

  it("ignores changes to chunks that aren't loaded", () => {
    const { mesher, world } = loaded();
    world.setBlock(500, 5, 5, 1);
    world.tick(0, 0, 1);
    expect(mesher.jobs.length).toBe(0);
  });

  it("never lets an older mesh replace a newer one", () => {
    const { mesher, view, world } = loaded();
    world.setBlock(5, 5, 5, 0);
    world.tick(0, 0, 1);
    const older = mesher.jobs.splice(0);
    world.setBlock(6, 5, 5, 0);
    world.tick(0, 0, 1);
    mesher.finishAll();
    const newer = view.shown.get(chunkKey(0, 0, 0));
    for (const j of older) j.done({ tag: "old" } as unknown as ChunkMesh);
    expect(view.shown.get(chunkKey(0, 0, 0))).toBe(newer);
  });
});

/** A world at view distance 1 with every chunk loaded, meshed and shown. */
function loaded() {
  const env = setup();
  env.world.tick(0, 0, 1);
  while (env.sent.length) {
    answer(env.world, env.sent);
    env.world.tick(0, 0, 1);
  }
  env.mesher.finishAll();
  return env;
}
