import { describe, expect, it } from "vitest";
import { chunkKey, type ChunkPos } from "./coords";
import type { ChunkMesh } from "./mesher";
import { WorldClient, type ChunkView, type Mesher } from "./world-client";

const AIR = Uint8Array.of(1, 1, 0);
const STONE = Uint8Array.of(1, 1, 1);
const bounds = { radius: 32, height: 32 }; // chunk columns -2..1, two layers

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
  shown = new Map<string, ChunkPos>();
  show(key: string, pos: ChunkPos) {
    this.shown.set(key, pos);
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
  const world = new WorldClient({ send: (b) => sent.push(b), mesher, view });
  world.start(bounds);
  return { sent, mesher, view, world };
}

/** Answers every outstanding request with stone below y=16 and air above. */
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
    const big = { radius: 512, height: 32 };
    const { sent, mesher, view, world } = setup();
    world.start(big);
    world.tick(0, 0, 1);
    while (sent.length) {
      answer(world, sent);
      world.tick(0, 0, 1);
    }
    mesher.finishAll();
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(true);
    world.tick(16 * 20, 0, 1);
    expect(view.shown.has(chunkKey(0, 0, 0))).toBe(false);
  });
});
