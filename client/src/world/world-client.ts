import { decodeChunk } from "./chunk";
import { CHUNK_SIZE } from "./constants";
import { chunkKey, parseKey, type ChunkKey, type ChunkPos } from "./coords";
import { FACE_NORMALS, type ChunkMesh } from "./mesher";
import { ChunkStore } from "./store";
import { ChunkStreamer, type WorldBounds } from "./streamer";

export interface Mesher {
  /** How many more jobs can start now. */
  capacity(): number;
  mesh(volume: Uint16Array, done: (mesh: ChunkMesh) => void): void;
}

export interface ChunkView {
  show(key: ChunkKey, pos: ChunkPos, mesh: ChunkMesh): void;
  remove(key: ChunkKey): void;
  clear(): void;
}

export interface WorldClientDeps {
  send(positions: ChunkPos[]): void;
  mesher: Mesher;
  view: ChunkView;
}

export interface ReceivedChunk {
  pos: ChunkPos;
  data: Uint8Array;
}

/** Keeps the chunks around the player fetched, meshed and shown. */
export class WorldClient {
  private store = new ChunkStore({ radius: 0, height: 0 });
  private streamer = new ChunkStreamer({ radius: 0, height: 0 });
  private readonly toMesh = new Set<ChunkKey>();
  private readonly meshed = new Map<ChunkKey, number>(); // key → id of its latest mesh job
  private jobs = 0;
  private center = { cx: 0, cz: 0, vd: 0 };

  constructor(private readonly deps: WorldClientDeps) {}

  /** Begins a fresh world, e.g. after (re)connecting; results from before are discarded. */
  start(bounds: WorldBounds): void {
    this.store = new ChunkStore(bounds);
    this.streamer = new ChunkStreamer(bounds);
    this.toMesh.clear();
    this.meshed.clear();
    this.deps.view.clear();
  }

  /** Stores arrived chunks and queues any that just became meshable. */
  receive(chunks: readonly ReceivedChunk[]): void {
    for (const { pos, data } of chunks) this.store.set(pos, decodeChunk(data));
    this.streamer.received(chunks.map((c) => c.pos));
    for (const { pos } of chunks) {
      this.consider(pos.x, pos.y, pos.z);
      for (const [dx, dy, dz] of FACE_NORMALS) this.consider(pos.x + dx, pos.y + dy, pos.z + dz);
    }
  }

  /** The block at world position x,y,z, or undefined if it isn't loaded. */
  blockAt(x: number, y: number, z: number): number | undefined {
    return this.store.blockAt(x, y, z);
  }

  /** Changes a loaded block and re-meshes every chunk whose mesh shows it. */
  setBlock(x: number, y: number, z: number, block: number): void {
    if (!this.store.setBlock(x, y, z, block)) return;
    const S = CHUNK_SIZE;
    const [cx, cy, cz] = [x, y, z].map((v) => Math.floor(v / S));
    const [lx, ly, lz] = [x - cx * S, y - cy * S, z - cz * S];
    this.remesh(cx, cy, cz);
    for (const [dx, dy, dz] of FACE_NORMALS) {
      const edge = (l: number, d: number) => (d < 0 && l === 0) || (d > 0 && l === S - 1);
      if (edge(lx, dx) || edge(ly, dy) || edge(lz, dz)) this.remesh(cx + dx, cy + dy, cz + dz);
    }
  }

  /** Call every frame with the player's block position; does little unless something is due. */
  tick(x: number, z: number, viewDistance: number): void {
    const cx = Math.floor(x / CHUNK_SIZE);
    const cz = Math.floor(z / CHUNK_SIZE);
    const c = this.center;
    if (c.cx !== cx || c.cz !== cz || c.vd !== viewDistance) {
      this.center = { cx, cz, vd: viewDistance };
      for (const key of this.streamer.update(cx, cz, viewDistance)) this.drop(key);
      this.store.forEachKey((key) => {
        const p = parseKey(key);
        this.consider(p.x, p.y, p.z);
      });
    }
    for (let batch = this.streamer.nextBatch(); batch; batch = this.streamer.nextBatch()) {
      this.deps.send(batch);
    }
    this.pumpMeshing();
  }

  private consider(x: number, y: number, z: number): void {
    const key = chunkKey(x, y, z);
    if (!this.meshed.has(key) && this.inView(x, z) && this.store.ready(x, y, z)) {
      this.toMesh.add(key);
    }
  }

  private remesh(x: number, y: number, z: number): void {
    const key = chunkKey(x, y, z);
    if (this.meshed.delete(key)) this.toMesh.add(key);
  }

  private inView(x: number, z: number): boolean {
    const { cx, cz, vd } = this.center;
    return (x - cx) ** 2 + (z - cz) ** 2 <= vd * vd;
  }

  private pumpMeshing(): void {
    const { mesher, view } = this.deps;
    for (const key of this.toMesh) {
      if (mesher.capacity() <= 0) return;
      this.toMesh.delete(key);
      const p = parseKey(key);
      if (!this.store.ready(p.x, p.y, p.z)) continue;
      const job = ++this.jobs; // ids never repeat, so results from before start() never match
      this.meshed.set(key, job);
      mesher.mesh(this.store.padded(p.x, p.y, p.z), (mesh) => {
        if (this.meshed.get(key) === job) view.show(key, p, mesh);
      });
    }
  }

  private drop(key: ChunkKey): void {
    this.store.delete(key);
    this.toMesh.delete(key);
    this.meshed.delete(key);
    this.deps.view.remove(key);
  }
}
