import { blockAt, chunkIndex, CHUNK_VOLUME, type Chunk } from "./chunk";
import { CHUNK_SIZE as S } from "./constants";
import { chunkKey, type ChunkKey, type ChunkPos } from "./coords";
import { FACE_NORMALS } from "./mesher";
import { columnInside, type WorldBounds } from "./streamer";

/** The client's copy of loaded chunks. */
export class ChunkStore {
  private readonly chunks = new Map<ChunkKey, Chunk>();
  private readonly tops = new Map<string, Int16Array>(); // per chunk column, highest block y or -1
  private readonly layers: number;
  /** Goes up on every change, so anything derived from the blocks knows to look again. */
  version = 0;

  constructor(private readonly bounds: WorldBounds) {
    this.layers = Math.ceil(bounds.height / S);
  }

  set(p: ChunkPos, c: Chunk): void {
    this.chunks.set(chunkKey(p.x, p.y, p.z), c);
    this.changed(p.x, p.z);
  }

  get(x: number, y: number, z: number): Chunk | undefined {
    return this.chunks.get(chunkKey(x, y, z));
  }

  /** The block at world position x,y,z, or undefined if its chunk isn't loaded. */
  blockAt(x: number, y: number, z: number): number | undefined {
    const c = this.get(Math.floor(x / S), Math.floor(y / S), Math.floor(z / S));
    return c && blockAt(c, mod(x), mod(y), mod(z));
  }

  /** Changes one block; returns false if its chunk isn't loaded. */
  setBlock(x: number, y: number, z: number, block: number): boolean {
    const c = this.get(Math.floor(x / S), Math.floor(y / S), Math.floor(z / S));
    if (!c) return false;
    if (!c.blocks && c.fill === block) return true;
    c.blocks ??= new Uint16Array(CHUNK_VOLUME).fill(c.fill);
    c.blocks[chunkIndex(mod(x), mod(y), mod(z))] = block;
    this.changed(Math.floor(x / S), Math.floor(z / S));
    return true;
  }

  delete(key: ChunkKey): void {
    if (!this.chunks.delete(key)) return;
    const [x, , z] = key.split(",").map(Number);
    this.changed(x, z);
  }

  clear(): void {
    this.chunks.clear();
    this.tops.clear();
    this.version++;
  }

  /** The y of the highest loaded block that isn't air in column x,z, or -1 if there is none. */
  topAt(x: number, z: number): number {
    const [cx, cz] = [Math.floor(x / S), Math.floor(z / S)];
    const column = `${cx},${cz}`;
    let tops = this.tops.get(column);
    if (!tops) {
      tops = this.columnTops(cx, cz);
      this.tops.set(column, tops);
    }
    return tops[mod(z) * S + mod(x)];
  }

  private changed(cx: number, cz: number): void {
    this.tops.delete(`${cx},${cz}`);
    this.version++;
  }

  /** Every column's top in a chunk column, scanning layers from the top down. */
  private columnTops(cx: number, cz: number): Int16Array {
    const tops = new Int16Array(S * S).fill(-1);
    let left = S * S;
    for (let cy = this.layers - 1; cy >= 0 && left > 0; cy--) {
      const c = this.get(cx, cy, cz);
      if (!c || (!c.blocks && c.fill === 0)) continue;
      for (let i = 0; i < S * S; i++) {
        if (tops[i] >= 0) continue;
        const y = highestIn(c, i % S, Math.floor(i / S));
        if (y < 0) continue;
        tops[i] = cy * S + y;
        left--;
      }
    }
    return tops;
  }

  forEachKey(fn: (key: ChunkKey) => void): void {
    for (const key of this.chunks.keys()) fn(key);
  }

  /** Whether the chunk can mesh correctly: it and every in-world neighbor are loaded. */
  ready(x: number, y: number, z: number): boolean {
    if (!this.get(x, y, z)) return false;
    return FACE_NORMALS.every(([dx, dy, dz]) => {
      const [nx, ny, nz] = [x + dx, y + dy, z + dz];
      return !this.inside(nx, ny, nz) || this.get(nx, ny, nz) !== undefined;
    });
  }

  /** The chunk plus a one-block border from its neighbors, as the mesher reads it. */
  padded(x: number, y: number, z: number): Uint16Array {
    return this.region(x, y, z, 1);
  }

  /**
   * Copies the chunk and every block within `margin` of it from all 26 neighbors; anything
   * missing is air. Indexed like paddedIndex, with each coordinate from -margin.
   */
  region(x: number, y: number, z: number, margin: number): Uint16Array {
    const size = S + 2 * margin;
    const out = new Uint16Array(size ** 3);
    for (let dy = -1; dy <= 1; dy++) {
      for (let dz = -1; dz <= 1; dz++) {
        for (let dx = -1; dx <= 1; dx++) {
          const c = this.get(x + dx, y + dy, z + dz);
          if (c) copyNeighbor(out, size, margin, c, [dx, dy, dz]);
        }
      }
    }
    return out;
  }

  private inside(x: number, y: number, z: number): boolean {
    return y >= 0 && y < this.layers && columnInside(x, z, this.bounds);
  }
}

/** The highest local y in chunk column x,z that isn't air, or -1. */
function highestIn(c: Chunk, x: number, z: number): number {
  if (!c.blocks) return c.fill === 0 ? -1 : S - 1;
  for (let y = S - 1; y >= 0; y--) if (c.blocks[chunkIndex(x, y, z)] !== 0) return y;
  return -1;
}

const mod = (v: number) => ((v % S) + S) % S;

/** Copies the part of neighbor c (at offset d from the chunk) within margin into a region. */
function copyNeighbor(out: Uint16Array, size: number, margin: number, c: Chunk, d: number[]): void {
  const from = d.map((o) => (o < 0 ? S - margin : 0));
  const to = d.map((o) => (o > 0 ? margin : S));
  const at = (l: number, a: number) => l + d[a] * S + margin;
  for (let y = from[1]; y < to[1]; y++) {
    for (let z = from[2]; z < to[2]; z++) {
      const row = (at(y, 1) * size + at(z, 2)) * size;
      for (let x = from[0]; x < to[0]; x++) out[row + at(x, 0)] = blockAt(c, x, y, z);
    }
  }
}
