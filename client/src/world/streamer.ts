import { CHUNK_SIZE } from "./constants";
import { chunkKey, type ChunkKey, type ChunkPos } from "./coords";

export interface WorldBounds {
  radius: number; // half the border width in blocks; negative means unbounded
  height: number; // world height in blocks
}

export interface StreamerOptions {
  batchSize?: number;
  maxInFlight?: number;
}

interface Column {
  x: number;
  z: number;
}

/** Chunk columns inside the border within r chunks of cx,cz, nearest first. */
export function columnsAround(cx: number, cz: number, r: number, bounds: WorldBounds): Column[] {
  const cols: Column[] = [];
  for (let x = cx - r; x <= cx + r; x++) {
    for (let z = cz - r; z <= cz + r; z++) {
      const d = (x - cx) ** 2 + (z - cz) ** 2;
      if (d <= r * r && columnInside(x, z, bounds)) cols.push({ x, z });
    }
  }
  const dist = (c: Column) => (c.x - cx) ** 2 + (c.z - cz) ** 2;
  return cols.sort((a, b) => dist(a) - dist(b));
}

/** Mirrors the server's World.Inside for the horizontal axes. */
export function columnInside(x: number, z: number, { radius }: WorldBounds): boolean {
  if (radius < 0) return true;
  const ox = x * CHUNK_SIZE;
  const oz = z * CHUNK_SIZE;
  return ox < radius && ox + CHUNK_SIZE > -radius && oz < radius && oz + CHUNK_SIZE > -radius;
}

/**
 * Decides which chunks to fetch and drop around the player. It fetches one ring past the view
 * so edge chunks have neighbors to mesh against, and keeps one more ring to avoid thrashing.
 */
export class ChunkStreamer {
  private readonly batchSize: number;
  private readonly maxInFlight: number;
  private readonly layers: number;
  private queue: ChunkPos[] = [];
  private cursor = 0;
  private readonly inFlight: ChunkPos[][] = []; // requests sent, oldest first
  private readonly pending = new Set<ChunkKey>();
  private readonly loaded = new Set<ChunkKey>();
  private center?: { cx: number; cz: number; vd: number };

  constructor(
    private readonly bounds: WorldBounds,
    opts: StreamerOptions = {},
  ) {
    this.batchSize = opts.batchSize ?? 32;
    this.maxInFlight = opts.maxInFlight ?? 2;
    this.layers = Math.ceil(bounds.height / CHUNK_SIZE);
  }

  /** Re-centers on chunk column cx,cz and returns the keys of chunks to unload. */
  update(cx: number, cz: number, viewDistance: number): ChunkKey[] {
    const c = this.center;
    if (c && c.cx === cx && c.cz === cz && c.vd === viewDistance) return [];
    this.center = { cx, cz, vd: viewDistance };
    this.queue = columnsAround(cx, cz, viewDistance + 1, this.bounds).flatMap((col) =>
      Array.from({ length: this.layers }, (_, y) => ({ x: col.x, y, z: col.z })),
    );
    this.cursor = 0;
    return this.evict(cx, cz, viewDistance + 2);
  }

  /** The next positions to request, or undefined when nothing is due or too much is in flight. */
  nextBatch(): ChunkPos[] | undefined {
    if (this.inFlight.length >= this.maxInFlight) return undefined;
    const batch: ChunkPos[] = [];
    for (; this.cursor < this.queue.length && batch.length < this.batchSize; this.cursor++) {
      const p = this.queue[this.cursor];
      const key = chunkKey(p.x, p.y, p.z);
      if (this.loaded.has(key) || this.pending.has(key)) continue;
      this.pending.add(key);
      batch.push(p);
    }
    if (batch.length === 0) return undefined;
    this.inFlight.push(batch);
    return batch;
  }

  /**
   * Records the reply to the oldest request. Chunks it left out (too far from where the
   * server thinks the player is) are asked for again after the next re-centering.
   */
  received(got: readonly ChunkPos[]): void {
    for (const p of this.inFlight.shift() ?? []) this.pending.delete(chunkKey(p.x, p.y, p.z));
    for (const p of got) this.loaded.add(chunkKey(p.x, p.y, p.z));
  }

  has(key: ChunkKey): boolean {
    return this.loaded.has(key);
  }

  reset(): void {
    this.queue = [];
    this.cursor = 0;
    this.inFlight.length = 0;
    this.pending.clear();
    this.loaded.clear();
    this.center = undefined;
  }

  private evict(cx: number, cz: number, keep: number): ChunkKey[] {
    const dropped: ChunkKey[] = [];
    for (const key of this.loaded) {
      const [x, , z] = key.split(",").map(Number);
      if ((x - cx) ** 2 + (z - cz) ** 2 > keep * keep) dropped.push(key);
    }
    for (const key of dropped) this.loaded.delete(key);
    return dropped;
  }
}
