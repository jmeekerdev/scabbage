import { CHUNK_SIZE } from "./constants";

export const CHUNK_VOLUME = CHUNK_SIZE ** 3;

/** A CHUNK_SIZE³ block volume; uniform chunks keep only their fill block. */
export interface Chunk {
  fill: number;
  blocks?: Uint16Array;
}

/** Storage order shared with the server: x fastest, then z, then y. */
export function chunkIndex(x: number, y: number, z: number): number {
  return (y * CHUNK_SIZE + z) * CHUNK_SIZE + x;
}

export function blockAt(c: Chunk, x: number, y: number, z: number): number {
  return c.blocks ? c.blocks[chunkIndex(x, y, z)] : c.fill;
}

const ENCODING_VERSION = 2; // must match the server: 2 is 32-block chunks

/** Parses the server's palette + run-length chunk encoding (server/internal/world/codec.go). */
export function decodeChunk(data: Uint8Array): Chunk {
  const r = new Reader(data);
  if (r.byte() !== ENCODING_VERSION) throw new Error("unknown chunk encoding");
  const size = r.uvarint();
  if (size === 0 || size > CHUNK_VOLUME) throw new Error("bad chunk palette");
  const palette = Array.from({ length: size }, () => r.uvarint());
  const chunk: Chunk = { fill: palette[0] };
  if (size > 1) chunk.blocks = decodeRuns(r, palette);
  if (!r.done()) throw new Error("trailing chunk data");
  return chunk;
}

function decodeRuns(r: Reader, palette: number[]): Uint16Array {
  const blocks = new Uint16Array(CHUNK_VOLUME);
  for (let i = 0; i < CHUNK_VOLUME;) {
    const run = r.uvarint();
    const block = palette[r.uvarint()];
    if (run === 0 || run > CHUNK_VOLUME - i || block === undefined) {
      throw new Error("bad chunk run");
    }
    blocks.fill(block, i, i + run);
    i += run;
  }
  return blocks;
}

/** A byte cursor that throws on truncated input. */
class Reader {
  private off = 0;

  constructor(private readonly data: Uint8Array) {}

  byte(): number {
    if (this.off >= this.data.length) throw new Error("truncated chunk data");
    return this.data[this.off++];
  }

  // Values here stay far below 2^31, so plain number arithmetic is exact.
  uvarint(): number {
    let value = 0;
    for (let shift = 0; shift < 35; shift += 7) {
      const b = this.byte();
      value += (b & 0x7f) * 2 ** shift;
      if (b < 0x80) return value;
    }
    throw new Error("varint too long");
  }

  done(): boolean {
    return this.off === this.data.length;
  }
}
