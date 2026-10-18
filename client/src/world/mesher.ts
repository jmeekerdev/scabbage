import { CHUNK_SIZE as S } from "./constants";

/** Padded volumes hold a chunk plus a one-block border copied from its neighbors. */
export const PADDED = S + 2;
export const PADDED_VOLUME = PADDED ** 3;

/** Index into a padded volume; each coordinate runs from -1 to 16. */
export function paddedIndex(x: number, y: number, z: number): number {
  return ((y + 1) * PADDED + (z + 1)) * PADDED + (x + 1);
}

/** Face order matches three.js BoxGeometry: +x, -x, +y, -y, +z, -z. */
export const FACE_NORMALS = [
  [1, 0, 0],
  [-1, 0, 0],
  [0, 1, 0],
  [0, -1, 0],
  [0, 0, 1],
  [0, 0, -1],
] as const;

/** Per-block rendering facts, indexed by block ID (layers by ID * 6 + face). */
export interface RenderTable {
  layers: Int16Array; // texture layer per face, -1 for air
  transparent: Uint8Array; // neighbors' faces stay visible
  translucent: Uint8Array; // drawn in the blended pass (water)
}

export interface MeshData {
  positions: Uint8Array; // chunk-local corners, 0–16
  uvs: Uint8Array; // in blocks, so textures repeat once per block
  faces: Uint8Array;
  layers: Uint16Array;
  indices: Uint16Array | Uint32Array;
}

export interface ChunkMesh {
  opaque: MeshData;
  translucent: MeshData;
}

/** Builds greedy-merged quads for every visible face in a padded chunk volume. */
export function meshChunk(volume: Uint16Array, table: RenderTable): ChunkMesh {
  const opaque = new MeshBuilder();
  const translucent = new MeshBuilder();
  const mask = new Int32Array(S * S);
  for (let face = 0; face < 6; face++) {
    for (let slice = 0; slice < S; slice++) {
      fillMask(mask, volume, table, face, slice);
      greedy(mask, (block, i, j, w, h) => {
        const out = table.translucent[block] ? translucent : opaque;
        out.quad(face, slice, i, j, w, h, table.layers[block * 6 + face]);
      });
    }
  }
  return { opaque: opaque.build(), translucent: translucent.build() };
}

/** Axes of a face: d is its normal axis, u and v span the face (u × v points along +d). */
function axes(face: number): [d: number, u: number, v: number] {
  const d = face >> 1;
  return [d, (d + 1) % 3, (d + 2) % 3];
}

/** Marks each cell in one slice with the block whose face shows there, or 0. */
function fillMask(
  mask: Int32Array,
  vol: Uint16Array,
  table: RenderTable,
  face: number,
  slice: number,
): void {
  const [d, u, v] = axes(face);
  const n = FACE_NORMALS[face];
  const p = [0, 0, 0];
  p[d] = slice;
  for (let j = 0; j < S; j++) {
    for (let i = 0; i < S; i++) {
      p[u] = i;
      p[v] = j;
      const block = vol[paddedIndex(p[0], p[1], p[2])];
      const next = vol[paddedIndex(p[0] + n[0], p[1] + n[1], p[2] + n[2])];
      mask[i + j * S] = faceVisible(block, next, table) ? block : 0;
    }
  }
}

function faceVisible(block: number, neighbor: number, table: RenderTable): boolean {
  return block !== 0 && block !== neighbor && table.transparent[neighbor] === 1;
}

/** Covers equal non-zero mask cells with as few rectangles as possible, clearing the mask. */
function greedy(
  mask: Int32Array,
  emit: (block: number, i: number, j: number, w: number, h: number) => void,
): void {
  for (let j = 0; j < S; j++) {
    for (let i = 0; i < S;) {
      const block = mask[i + j * S];
      if (block === 0) {
        i++;
        continue;
      }
      let w = 1;
      while (i + w < S && mask[i + w + j * S] === block) w++;
      let h = 1;
      while (j + h < S && rowMatches(mask, block, i, j + h, w)) h++;
      for (let y = j; y < j + h; y++) mask.fill(0, i + y * S, i + w + y * S);
      emit(block, i, j, w, h);
      i += w;
    }
  }
}

function rowMatches(mask: Int32Array, block: number, i: number, j: number, w: number): boolean {
  for (let k = i; k < i + w; k++) if (mask[k + j * S] !== block) return false;
  return true;
}

/** Texture axes per normal axis: side faces put v on height so textures stand upright. */
const TEX_AXES: readonly [number, number][] = [
  [2, 1], // x faces: u = z, v = y
  [0, 2], // y faces: u = x, v = z
  [0, 1], // z faces: u = x, v = y
];

/** Accumulates quads into growable typed arrays. */
class MeshBuilder {
  private positions = new Uint8Array(256 * 3);
  private uvs = new Uint8Array(256 * 2);
  private faces = new Uint8Array(256);
  private layers = new Uint16Array(256);
  private vertices = 0;

  quad(
    face: number,
    slice: number,
    i: number,
    j: number,
    w: number,
    h: number,
    layer: number,
  ): void {
    this.reserve(this.vertices + 4);
    const [d, u, v] = axes(face);
    const [tu, tv] = TEX_AXES[d];
    const plane = face & 1 ? slice : slice + 1;
    const corners = [
      [i, j],
      [i + w, j],
      [i + w, j + h],
      [i, j + h],
    ];
    for (const [cu, cv] of corners) {
      const p = [0, 0, 0];
      p[d] = plane;
      p[u] = cu;
      p[v] = cv;
      const k = this.vertices++;
      this.positions.set(p, k * 3);
      this.uvs[k * 2] = p[tu];
      this.uvs[k * 2 + 1] = p[tv];
      this.faces[k] = face;
      this.layers[k] = layer;
    }
  }

  build(): MeshData {
    const n = this.vertices;
    const indices = n > 0xffff ? new Uint32Array((n / 4) * 6) : new Uint16Array((n / 4) * 6);
    for (let q = 0; q < n / 4; q++) {
      const base = q * 4;
      // Positive faces wind 0-1-2 counter-clockwise from outside; negative faces reverse it.
      const order = this.faces[base] & 1 ? [0, 2, 1, 0, 3, 2] : [0, 1, 2, 0, 2, 3];
      for (let k = 0; k < 6; k++) indices[q * 6 + k] = base + order[k];
    }
    return {
      positions: this.positions.slice(0, n * 3),
      uvs: this.uvs.slice(0, n * 2),
      faces: this.faces.slice(0, n),
      layers: this.layers.slice(0, n),
      indices,
    };
  }

  private reserve(vertices: number): void {
    if (vertices <= this.faces.length) return;
    const grow = <T extends Uint8Array | Uint16Array>(a: T): T => {
      const next = new (a.constructor as new (n: number) => T)(a.length * 2);
      next.set(a);
      return next;
    };
    this.positions = grow(this.positions);
    this.uvs = grow(this.uvs);
    this.faces = grow(this.faces);
    this.layers = grow(this.layers);
  }
}
