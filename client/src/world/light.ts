import { METER, CHUNK_SIZE as S } from "./constants";
import { PADDED, PADDED_VOLUME, paddedIndex } from "./mesher";

/** The brightest a block glows, as the server's world.MaxLight; light fades one per block. */
export const MAX_LIGHT = 10 * METER;

/** Blocks around a chunk that can still light its one-block border. */
export const LIGHT_MARGIN = MAX_LIGHT;
export const REGION = S + 2 * LIGHT_MARGIN;
export const REGION_VOLUME = REGION ** 3;

/** Index into a light region; each coordinate runs from -LIGHT_MARGIN to S - 1 + LIGHT_MARGIN. */
export function regionIndex(x: number, y: number, z: number): number {
  const m = LIGHT_MARGIN;
  return ((y + m) * REGION + (z + m)) * REGION + (x + m);
}

const STEPS = [1, -1, REGION, -REGION, REGION * REGION, -REGION * REGION];

/** Per block ID: whether it is see-through, whether light passes it, and how brightly it glows. */
export interface LightTable {
  transparent: Uint8Array;
  passes: Uint8Array;
  emits: Uint8Array;
}

/**
 * Floods light from glowing blocks through those it passes, brightest first so each
 * cell is settled the first time it is reached. Returns levels for the padded chunk.
 */
export function blockLight(
  region: Uint16Array,
  { transparent, passes, emits }: LightTable,
): Uint8Array {
  const level = new Uint8Array(REGION_VOLUME);
  const buckets: number[][] = Array.from({ length: MAX_LIGHT + 1 }, () => []);
  for (let i = 0; i < REGION_VOLUME; i++) {
    const l = Math.min(emits[region[i]] ?? 0, MAX_LIGHT);
    if (l > 0) {
      level[i] = l;
      buckets[l].push(i);
    }
  }
  for (let l = MAX_LIGHT; l > 1; l--) {
    for (const i of buckets[l]) {
      if (level[i] !== l) continue;
      for (const step of STEPS) {
        const n = i + step;
        if (!sameRow(i, n, step) || level[n] >= l - 1 || passes[region[n]] !== 1) continue;
        level[n] = l - 1;
        buckets[l - 1].push(n);
      }
    }
  }
  lightPartFilled(level, region, transparent, passes);
  return cropToPadded(level);
}

/** Gives stairs and slabs in the padded chunk one less than their brightest neighbor, for their faces. */
function lightPartFilled(
  level: Uint8Array,
  region: Uint16Array,
  transparent: Uint8Array,
  passes: Uint8Array,
): void {
  for (let y = -1; y <= S; y++) {
    for (let z = -1; z <= S; z++) {
      for (let x = -1; x <= S; x++) {
        const i = regionIndex(x, y, z);
        const b = region[i];
        if (passes[b] === 1 || transparent[b] !== 1) continue;
        let best = 0;
        for (const step of STEPS)
          if (passes[region[i + step]] === 1) best = Math.max(best, level[i + step]);
        level[i] = Math.max(level[i], best - 1);
      }
    }
  }
}

/** Whether stepping from i to n stays inside the region instead of wrapping around an edge. */
function sameRow(i: number, n: number, step: number): boolean {
  if (n < 0 || n >= REGION_VOLUME) return false;
  if (Math.abs(step) === 1) return Math.floor(i / REGION) === Math.floor(n / REGION);
  if (Math.abs(step) === REGION) return Math.floor(i / REGION ** 2) === Math.floor(n / REGION ** 2);
  return true;
}

/** The chunk and its one-block border, cut out of a light region. */
export function cropToPadded<T extends Uint8Array | Uint16Array>(region: T): T {
  const out = new (region.constructor as new (n: number) => T)(PADDED_VOLUME);
  for (let y = -1; y <= S; y++) {
    for (let z = -1; z <= S; z++) {
      const from = regionIndex(-1, y, z);
      out.set(region.subarray(from, from + PADDED), paddedIndex(-1, y, z));
    }
  }
  return out;
}
