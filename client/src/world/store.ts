import { blockAt, type Chunk } from "./chunk";
import { CHUNK_SIZE as S } from "./constants";
import { chunkKey, type ChunkKey, type ChunkPos } from "./coords";
import { FACE_NORMALS, PADDED_VOLUME, paddedIndex } from "./mesher";
import { columnInside, type WorldBounds } from "./streamer";

/** The client's copy of loaded chunks. */
export class ChunkStore {
  private readonly chunks = new Map<ChunkKey, Chunk>();
  private readonly layers: number;

  constructor(private readonly bounds: WorldBounds) {
    this.layers = Math.ceil(bounds.height / S);
  }

  set(p: ChunkPos, c: Chunk): void {
    this.chunks.set(chunkKey(p.x, p.y, p.z), c);
  }

  get(x: number, y: number, z: number): Chunk | undefined {
    return this.chunks.get(chunkKey(x, y, z));
  }

  delete(key: ChunkKey): void {
    this.chunks.delete(key);
  }

  clear(): void {
    this.chunks.clear();
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

  /** Copies the chunk and the touching layer of each face neighbor; anything missing is air. */
  padded(x: number, y: number, z: number, out = new Uint16Array(PADDED_VOLUME)): Uint16Array {
    out.fill(0);
    const self = this.get(x, y, z);
    if (self) copyRegion(out, self, [0, 0, 0], [S, S, S], [0, 0, 0]);
    for (const n of FACE_NORMALS) {
      const neighbor = this.get(x + n[0], y + n[1], z + n[2]);
      if (!neighbor) continue;
      // The slab one block past this chunk's face is the neighbor's opposite edge.
      const from = n.map((d) => (d < 0 ? S - 1 : 0));
      const size = n.map((d) => (d === 0 ? S : 1));
      const to = n.map((d) => (d > 0 ? S : d < 0 ? -1 : 0));
      copyRegion(out, neighbor, from, size, to);
    }
    return out;
  }

  private inside(x: number, y: number, z: number): boolean {
    return y >= 0 && y < this.layers && columnInside(x, z, this.bounds);
  }
}

/** Copies a box of size blocks starting at `from` in chunk c to `to` in a padded volume. */
function copyRegion(
  out: Uint16Array,
  c: Chunk,
  from: number[],
  size: number[],
  to: number[],
): void {
  for (let y = 0; y < size[1]; y++) {
    for (let z = 0; z < size[2]; z++) {
      for (let x = 0; x < size[0]; x++) {
        out[paddedIndex(to[0] + x, to[1] + y, to[2] + z)] = blockAt(
          c,
          from[0] + x,
          from[1] + y,
          from[2] + z,
        );
      }
    }
  }
}
