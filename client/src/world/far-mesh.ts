import { CHUNK_SIZE } from "./constants";

/** Samples along each side of a far tile; must match the server's world.FarCells. */
export const FAR_CELLS = 8;
/** Far tiles along each side of a region, which is drawn as one mesh. */
export const FAR_REGION = 8;

const STEP = CHUNK_SIZE / FAR_CELLS; // blocks between samples
const SPAN = FAR_REGION * FAR_CELLS; // samples along a region's side

/** One chunk column sketched by the server: per sample, x fastest, the y above its top block and that block. */
export interface FarTile {
  tops: readonly number[];
  blocks: readonly number[];
}

export interface FarMeshData {
  positions: Float32Array; // relative to the region's low corner
  colors: Uint8Array; // sRGB, already shaded by slope
  indices: Uint16Array;
}

/**
 * A surface through the samples of region rx,rz's tiles and the first ones past its +x and +z
 * edges, so neighboring regions meet. Samples are drawn at their cells' low corners, so a tile's
 * surface reaches exactly to its column's edges. `colors` holds RGB per block ID.
 */
export function farRegionMesh(
  tileAt: (tx: number, tz: number) => FarTile | undefined,
  rx: number,
  rz: number,
  colors: Uint8Array,
): FarMeshData {
  const side = SPAN + 1;
  const top = (sx: number, sz: number) => sampleAt(tileAt, rx * SPAN + sx, rz * SPAN + sz, "tops");
  const vertex = new Int32Array(side * side).fill(-1);
  const positions: number[] = [];
  const shaded: number[] = [];
  for (let sz = 0; sz < side; sz++) {
    for (let sx = 0; sx < side; sx++) {
      const y = top(sx, sz);
      if (y === undefined) continue;
      vertex[sz * side + sx] = positions.length / 3;
      positions.push(sx * STEP, y, sz * STEP);
      const block = sampleAt(tileAt, rx * SPAN + sx, rz * SPAN + sz, "blocks") ?? 0;
      const shade = slopeShade(
        y,
        top(sx - 1, sz),
        top(sx + 1, sz),
        top(sx, sz - 1),
        top(sx, sz + 1),
      );
      for (let c = 0; c < 3; c++) shaded.push(Math.round((colors[block * 3 + c] ?? 0) * shade));
    }
  }
  return {
    positions: Float32Array.from(positions),
    colors: Uint8Array.from(shaded),
    indices: Uint16Array.from(gridTriangles(vertex, side)),
  };
}

/** The sample at global sample position sx,sz, if its tile has arrived. */
function sampleAt(
  tileAt: (tx: number, tz: number) => FarTile | undefined,
  sx: number,
  sz: number,
  field: keyof FarTile,
): number | undefined {
  const tx = Math.floor(sx / FAR_CELLS);
  const tz = Math.floor(sz / FAR_CELLS);
  const i = (sz - tz * FAR_CELLS) * FAR_CELLS + (sx - tx * FAR_CELLS);
  return tileAt(tx, tz)?.[field][i];
}

/** Like a top face in full light when flat, darker as the ground tilts; missing neighbors count as level. */
function slopeShade(y: number, ...around: (number | undefined)[]): number {
  const [w, e, n, s] = around.map((h) => h ?? y);
  const dx = (e - w) / (2 * STEP);
  const dz = (s - n) / (2 * STEP);
  return 0.6 + 0.4 / Math.sqrt(1 + dx * dx + dz * dz);
}

/** Two triangles for every grid cell whose four corners all have a vertex, facing up. */
function gridTriangles(vertex: Int32Array, side: number): number[] {
  const out: number[] = [];
  for (let sz = 0; sz + 1 < side; sz++) {
    for (let sx = 0; sx + 1 < side; sx++) {
      const a = vertex[sz * side + sx];
      const b = vertex[sz * side + sx + 1];
      const c = vertex[(sz + 1) * side + sx];
      const d = vertex[(sz + 1) * side + sx + 1];
      if (a < 0 || b < 0 || c < 0 || d < 0) continue;
      out.push(a, c, b, b, c, d);
    }
  }
  return out;
}
