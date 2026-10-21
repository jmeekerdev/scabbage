import { CHUNK_VOLUME, decodeChunk, type Chunk } from "./chunk";
import { CHUNK_SIZE } from "./constants";
import { chunkKey, parseKey, type ChunkKey, type ChunkPos } from "./coords";
import { LIGHT_MARGIN } from "./light";
import { FACE_NORMALS, type ChunkMesh } from "./mesher";
import { ChunkStore } from "./store";
import { ChunkStreamer, type WorldBounds } from "./streamer";

export interface Mesher {
  /** How many more jobs can start now. */
  capacity(): number;
  /** Meshes a padded chunk volume, or a light region (see store.region) when lights are near. */
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
  /** Told when a chunk column starts or stops having every layer drawn. */
  cover?(x: number, z: number, covered: boolean): void;
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
  private emits: Uint8Array = new Uint8Array(0); // light level per block ID
  private transparent: Uint8Array = new Uint8Array(0); // 1 where light passes, per block ID
  private readonly glowing = new Map<ChunkKey, boolean>(); // cached: holds a glowing block
  private readonly stale = new Set<ChunkKey>(); // shown, but being fetched again
  private readonly drawn = new Map<string, Set<number>>(); // column → layers with a finished mesh
  private layers = 0;

  constructor(private readonly deps: WorldClientDeps) {}

  /** Begins a fresh world, e.g. after (re)connecting; results from before are discarded. */
  start(bounds: WorldBounds): void {
    this.store = new ChunkStore(bounds);
    this.streamer = new ChunkStreamer(bounds);
    this.center = { cx: 0, cz: 0, vd: 0 }; // so the first tick centers the new streamer, even in the same spot
    this.toMesh.clear();
    this.meshed.clear();
    this.glowing.clear();
    this.stale.clear();
    for (const [column, layers] of this.drawn)
      if (layers.size === this.layers) this.uncover(column);
    this.drawn.clear();
    this.layers = Math.ceil(bounds.height / CHUNK_SIZE);
    this.deps.view.clear();
  }

  /** How brightly each block ID glows and which let light through; meshes made from now on use it. */
  setLights(emits: Uint8Array, transparent: Uint8Array): void {
    this.emits = emits;
    this.transparent = transparent;
    this.glowing.clear();
  }

  /** Stores arrived chunks and queues any that just became meshable. */
  receive(chunks: readonly ReceivedChunk[]): void {
    for (const { pos, data } of chunks) {
      this.store.set(pos, decodeChunk(data));
      this.glowing.delete(chunkKey(pos.x, pos.y, pos.z));
    }
    this.streamer.received(chunks.map((c) => c.pos));
    for (const { pos } of chunks) {
      if (this.stale.delete(chunkKey(pos.x, pos.y, pos.z)))
        this.remeshAround(pos.x, pos.y, pos.z, () => true);
      this.consider(pos.x, pos.y, pos.z);
      for (const [dx, dy, dz] of FACE_NORMALS) this.consider(pos.x + dx, pos.y + dy, pos.z + dz);
      if (this.glows(pos.x, pos.y, pos.z)) this.remeshAround(pos.x, pos.y, pos.z, () => true);
    }
  }

  /** Fetches chunks again that changed on the server, showing the old ones until they arrive. */
  refetch(chunks: readonly ChunkPos[]): void {
    for (const p of chunks) {
      const key = chunkKey(p.x, p.y, p.z);
      if (this.streamer.refetch(key)) this.stale.add(key);
      else this.drop(key);
    }
  }

  /** The block at world position x,y,z, or undefined if it isn't loaded. */
  blockAt(x: number, y: number, z: number): number | undefined {
    return this.store.blockAt(x, y, z);
  }

  /** Changes a loaded block and re-meshes every chunk whose mesh shows it or its light. */
  setBlock(x: number, y: number, z: number, block: number): void {
    const old = this.store.blockAt(x, y, z) ?? 0;
    if (!this.store.setBlock(x, y, z, block)) return;
    const S = CHUNK_SIZE;
    const [cx, cy, cz] = [x, y, z].map((v) => Math.floor(v / S));
    const local = [x - cx * S, y - cy * S, z - cz * S];
    this.glowing.delete(chunkKey(cx, cy, cz));
    // A neighbor needs a new mesh if the block is within `reach` of its side.
    const near = (reach: number) => (d: number[]) =>
      d.every((o, a) => o === 0 || (o < 0 ? local[a] < reach : local[a] >= S - reach));
    this.remeshAround(cx, cy, cz, near(this.lightReach(old, block, cx, cy, cz)));
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

  /** Re-meshes the chunk and each neighbor at an offset `pick` accepts. */
  private remeshAround(x: number, y: number, z: number, pick: (d: number[]) => boolean): void {
    for (let dy = -1; dy <= 1; dy++) {
      for (let dz = -1; dz <= 1; dz++) {
        for (let dx = -1; dx <= 1; dx++)
          if (pick([dx, dy, dz])) this.remesh(x + dx, y + dy, z + dz);
      }
    }
  }

  /** How far swapping old for block changes light: its own glow, any light it now stops or lets through, or 1. */
  private lightReach(old: number, block: number, cx: number, cy: number, cz: number): number {
    const glow = Math.max(this.emits[old] ?? 0, this.emits[block] ?? 0);
    if (glow > 0) return Math.min(glow, LIGHT_MARGIN);
    const blocksLight = (b: number) => this.transparent[b] !== 1;
    return blocksLight(old) !== blocksLight(block) && this.lightsNear(cx, cy, cz)
      ? LIGHT_MARGIN
      : 1;
  }

  /** Whether a glowing block in this chunk or a neighbor could light it. */
  private lightsNear(x: number, y: number, z: number): boolean {
    for (let dy = -1; dy <= 1; dy++) {
      for (let dz = -1; dz <= 1; dz++) {
        for (let dx = -1; dx <= 1; dx++) if (this.glows(x + dx, y + dy, z + dz)) return true;
      }
    }
    return false;
  }

  private glows(x: number, y: number, z: number): boolean {
    const key = chunkKey(x, y, z);
    let glows = this.glowing.get(key);
    if (glows === undefined) {
      const c = this.store.get(x, y, z);
      glows = c !== undefined && holdsGlow(c, this.emits);
      this.glowing.set(key, glows);
    }
    return glows;
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
      const lit = this.lightsNear(p.x, p.y, p.z);
      const volume = lit
        ? this.store.region(p.x, p.y, p.z, LIGHT_MARGIN)
        : this.store.padded(p.x, p.y, p.z);
      mesher.mesh(volume, (mesh) => {
        if (this.meshed.get(key) !== job) return;
        view.show(key, p, mesh);
        this.markDrawn(p);
      });
    }
  }

  private markDrawn({ x, y, z }: ChunkPos): void {
    const column = `${x},${z}`;
    const layers = this.drawn.get(column) ?? new Set<number>();
    this.drawn.set(column, layers);
    const was = layers.size;
    layers.add(y);
    if (was < this.layers && layers.size === this.layers) this.deps.cover?.(x, z, true);
  }

  /** Forgets a layer's mesh, telling when that leaves its column partly drawn. */
  private undraw({ x, y, z }: ChunkPos): void {
    const column = `${x},${z}`;
    const layers = this.drawn.get(column);
    if (!layers?.delete(y)) return;
    if (layers.size === this.layers - 1) this.uncover(column);
    if (layers.size === 0) this.drawn.delete(column);
  }

  private uncover(column: string): void {
    const [x, z] = column.split(",").map(Number);
    this.deps.cover?.(x, z, false);
  }

  private drop(key: ChunkKey): void {
    this.undraw(parseKey(key));
    this.store.delete(key);
    this.glowing.delete(key);
    this.toMesh.delete(key);
    this.meshed.delete(key);
    this.deps.view.remove(key);
  }
}

function holdsGlow(c: Chunk, emits: Uint8Array): boolean {
  if (!c.blocks) return (emits[c.fill] ?? 0) > 0;
  for (let i = 0; i < CHUNK_VOLUME; i++) if ((emits[c.blocks[i]] ?? 0) > 0) return true;
  return false;
}
