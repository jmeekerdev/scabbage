import { CHUNK_SIZE as S } from "./constants";
import { SHAPE_BOXES, Shape, type Box } from "./shapes";
import { Smooth, SoftCorners } from "./soft-corners";

/** Mesh positions and uvs count eighths of a block, so shapes like slabs land on whole numbers. */
export const POSITION_SCALE = 8;

/** Padded volumes hold a chunk plus a one-block border copied from its neighbors. */
export const PADDED = S + 2;
export const PADDED_VOLUME = PADDED ** 3;

/** Index into a padded volume; each coordinate runs from -1 to CHUNK_SIZE. */
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
  shape: Uint8Array; // a Shape; anything but a cube is drawn box by box
  light: Uint8Array; // how brightly it glows
  smooth: Uint8Array; // a Smooth kind, for soft corners
}

/** Extra mesh data a look asks for; see Look.shape. */
export interface MeshOptions {
  edges: boolean;
  softCorners: number;
}

export const PLAIN_MESH: MeshOptions = { edges: false, softCorners: 0 };

export interface MeshData {
  positions: Uint16Array; // chunk-local corners in eighths, 0 to CHUNK_SIZE * 8
  uvs: Uint16Array; // in eighths, so textures repeat once per block
  faces: Uint8Array;
  layers: Uint16Array;
  ao: Uint8Array; // 0 (tucked in a corner) to 3 (open), per vertex
  light: Uint8Array; // block light, 0 to MAX_LIGHT, per vertex
  edges?: Uint8Array; // per vertex, distance + 1 in blocks to each open side (-u, +u, -v, +v), 0 if closed
  shifts?: Int8Array; // per vertex, its soft corner's move in eighths
  normals?: Int8Array; // per vertex, scaled to ±127
  indices: Uint16Array | Uint32Array;
}

export interface ChunkMesh {
  opaque: MeshData;
  translucent: MeshData;
}

/** A padded chunk volume with its block light levels; no light means darkness everywhere. */
interface Volume {
  blocks: Uint16Array;
  light?: Uint8Array;
  table: RenderTable;
  edges: boolean;
  soft?: SoftCorners;
}

/**
 * Builds greedy-merged quads for every visible cube face in a padded chunk volume, then
 * draws other shapes box by box. `light` holds a block light level per padded cell.
 */
export function meshChunk(
  blocks: Uint16Array,
  table: RenderTable,
  light?: Uint8Array,
  options = PLAIN_MESH,
): ChunkMesh {
  const soft = options.softCorners > 0;
  const vol: Volume = {
    blocks,
    light,
    table,
    edges: options.edges,
    soft: soft ? new SoftCorners(blocks, table.smooth, options.softCorners) : undefined,
  };
  const opaque = new MeshBuilder(options.edges, soft);
  const translucent = new MeshBuilder(options.edges, soft);
  const mask = new Int32Array(S * S);
  const lights = new Int32Array(S * S);
  const sides = new Uint8Array(S * S);
  for (let face = 0; face < 6; face++) {
    for (let slice = 0; slice < S; slice++) {
      fillMask(mask, lights, sides, vol, face, slice);
      greedy(mask, lights, sides, (key, light, open, i, j, w, h) => {
        const block = key & 0xffff;
        const out = table.translucent[block] ? translucent : opaque;
        const plane = (face & 1 ? slice : slice + 1) * POSITION_SCALE;
        const P = POSITION_SCALE;
        out.quad(
          face,
          plane,
          i * P,
          j * P,
          (i + w) * P,
          (j + h) * P,
          table.layers[block * 6 + face],
          key >>> 16,
          light,
          open,
        );
        if (vol.soft && table.smooth[block] === Smooth.NATURAL) out.soften(vol.soft);
      });
    }
  }
  meshShapes(vol, opaque);
  return { opaque: opaque.build(), translucent: translucent.build() };
}

const OPEN = 0xff; // all four corners unshaded

/** Draws each non-cube block's boxes, skipping faces flat against an opaque neighbor. */
function meshShapes(vol: Volume, out: MeshBuilder): void {
  const { blocks, table } = vol;
  for (let y = 0; y < S; y++) {
    for (let z = 0; z < S; z++) {
      for (let x = 0; x < S; x++) {
        const block = blocks[paddedIndex(x, y, z)];
        const shape = table.shape[block] ?? Shape.CUBE;
        if (block === 0 || shape === Shape.CUBE) continue;
        const l = lightAt(vol, x, y, z) * 0x1111; // the same level at all four corners
        for (const box of SHAPE_BOXES[shape] ?? []) {
          for (let face = 0; face < 6; face++) {
            if (hiddenByNeighbor(vol, box, face, x, y, z)) continue;
            emitBoxFace(out, box, face, [x, y, z], table.layers[block * 6 + face], l);
          }
        }
      }
    }
  }
}

function hiddenByNeighbor(vol: Volume, box: Box, face: number, x: number, y: number, z: number) {
  const d = face >> 1;
  const onEdge = face & 1 ? box[d] === 0 : box[d + 3] === POSITION_SCALE;
  const n = FACE_NORMALS[face];
  return (
    onEdge && vol.table.transparent[vol.blocks[paddedIndex(x + n[0], y + n[1], z + n[2])]] === 0
  );
}

function emitBoxFace(
  out: MeshBuilder,
  box: Box,
  face: number,
  cell: number[],
  layer: number,
  light: number,
): void {
  const [d, u, v] = axes(face);
  const at = (a: number, hi: boolean) => cell[a] * POSITION_SCALE + box[hi ? a + 3 : a];
  const plane = at(d, (face & 1) === 0);
  out.quad(face, plane, at(u, false), at(v, false), at(u, true), at(v, true), layer, OPEN, light);
}

function lightAt(vol: Volume, x: number, y: number, z: number): number {
  return vol.light ? vol.light[paddedIndex(x, y, z)] : 0;
}

/** Axes of a face: d is its normal axis, u and v span the face (u × v points along +d). */
function axes(face: number): [d: number, u: number, v: number] {
  const d = face >> 1;
  return [d, (d + 1) % 3, (d + 2) % 3];
}

/** Corner directions along (u, v), in the order quads list their corners. */
const CORNERS = [
  [-1, -1],
  [1, -1],
  [1, 1],
  [-1, 1],
] as const;

/** Index steps along x, y and z in a padded volume. */
const STRIDE = [1, PADDED * PADDED, PADDED] as const;

/**
 * Marks each visible cube face cell with its block and corner shading (block | ao << 16),
 * or 0, its four corner light levels packed 4 bits each, and its open sides (see openSides).
 */
function fillMask(
  mask: Int32Array,
  lights: Int32Array,
  sides: Uint8Array,
  vol: Volume,
  face: number,
  slice: number,
): void {
  const { blocks, table } = vol;
  const [d, u, v] = axes(face);
  const su = STRIDE[u];
  const sv = STRIDE[v];
  const toFront = FACE_NORMALS[face][d] * STRIDE[d];
  const origin = paddedIndex(0, 0, 0) + slice * STRIDE[d];
  for (let j = 0; j < S; j++) {
    for (let i = 0; i < S; i++) {
      const at = origin + i * su + j * sv;
      const block = blocks[at];
      const front = at + toFront;
      const k = i + j * S;
      if (!faceVisible(block, blocks[front], table)) {
        mask[k] = 0;
        lights[k] = 0;
        continue;
      }
      mask[k] = block | (cornerShades(vol, front, su, sv) << 16);
      lights[k] = vol.light ? cornerLights(vol, vol.light, front, su, sv) : 0;
      sides[k] = vol.edges ? openSides(vol, at, front - at, su, sv) : 0;
      if (vol.soft && table.smooth[block] === Smooth.NATURAL) {
        const cell = [0, 0, 0];
        cell[d] = slice + (face & 1 ? 0 : 1);
        cell[u] = i;
        cell[v] = j;
        if (softens(vol.soft, cell, face)) sides[k] |= LONE;
      }
    }
  }
}

// Side bits of a face cell: -u, +u, -v, +v; a lone cell never merges, since its corners move.
const U_SIDES = 0b0011;
const V_SIDES = 0b1100;
const LONE = 0x10;

/** Sides where the shape really ends: the cell beside differs and nothing covers the edge. */
function openSides(vol: Volume, at: number, toFront: number, su: number, sv: number): number {
  const { blocks, table } = vol;
  let bits = 0;
  [-su, su, -sv, sv].forEach((step, k) => {
    const beside = at + step;
    if (blocks[beside] !== blocks[at] && table.transparent[blocks[beside + toFront]] === 1)
      bits |= 1 << k;
  });
  return bits;
}

/** Whether any corner of a face cell, the low one at `corner`, moves or tilts away from the face. */
function softens(soft: SoftCorners, corner: number[], face: number): boolean {
  const [, u, v] = axes(face);
  const facing = FACE_NORMALS[face].map((n) => n * 127);
  for (let c = 0; c < 4; c++) {
    const p = [...corner];
    p[u] += c & 1;
    p[v] += c >> 1;
    const [x, y, z] = p;
    if (soft.shift(x, y, z).some((s) => s !== 0)) return true;
    const n = soft.normal(x, y, z);
    if (n.some((a) => a !== 0) && n.some((a, i) => a !== facing[i])) return true;
  }
  return false;
}

/** Smooth light per face corner: the average of the open cells around it in front of the face. */
function cornerLights(
  vol: Volume,
  light: Uint8Array,
  front: number,
  su: number,
  sv: number,
): number {
  const { blocks, table } = vol;
  const open = (at: number) => (table.transparent[blocks[at]] === 1 ? light[at] : -1);
  const center = open(front);
  let packed = 0;
  for (let k = 0; k < 4; k++) {
    const du = CORNERS[k][0] * su;
    const dv = CORNERS[k][1] * sv;
    const a = open(front + du);
    const b = open(front + dv);
    const c = a < 0 && b < 0 ? -1 : open(front + du + dv);
    let sum = 0;
    let count = 0;
    for (const l of [center, a, b, c])
      if (l >= 0) {
        sum += l;
        count++;
      }
    packed |= Math.round(sum / count) << (k * 4);
  }
  return packed;
}

/** Packs 2-bit occlusion for each face corner, from the blocks around the cell in front. */
function cornerShades(vol: Volume, front: number, su: number, sv: number): number {
  const { blocks, table } = vol;
  const solid = (at: number) => (table.transparent[blocks[at]] === 0 ? 1 : 0);
  let packed = 0;
  for (let k = 0; k < 4; k++) {
    const du = CORNERS[k][0] * su;
    const dv = CORNERS[k][1] * sv;
    const a = solid(front + du);
    const b = solid(front + dv);
    const ao = a && b ? 0 : 3 - a - b - solid(front + du + dv);
    packed |= ao << (k * 2);
  }
  return packed;
}

function faceVisible(block: number, neighbor: number, table: RenderTable): boolean {
  const cube = (table.shape[block] ?? Shape.CUBE) === Shape.CUBE;
  return block !== 0 && cube && block !== neighbor && table.transparent[neighbor] === 1;
}

/**
 * Covers equal non-zero mask cells (with equal light) in as few rectangles as possible, so that each
 * rectangle side is open or closed all along. Sides inside a rectangle are always closed, since the
 * cell beside is the same block.
 */
function greedy(
  mask: Int32Array,
  lights: Int32Array,
  sides: Uint8Array,
  emit: (
    key: number,
    light: number,
    open: number,
    i: number,
    j: number,
    w: number,
    h: number,
  ) => void,
): void {
  const joins = (a: number, b: number, along: number) =>
    mask[a] === mask[b] &&
    lights[a] === lights[b] &&
    !(sides[a] & LONE) &&
    (sides[a] & along) === (sides[b] & along);
  for (let j = 0; j < S; j++) {
    for (let i = 0; i < S;) {
      const k = i + j * S;
      if (mask[k] === 0) {
        i++;
        continue;
      }
      let w = 1;
      while (!(sides[k] & LONE) && i + w < S && joins(k + w, k, V_SIDES)) w++;
      let h = 1;
      grow: while (!(sides[k] & LONE) && j + h < S) {
        const row = k + h * S;
        for (let x = 0; x < w; x++)
          if (!joins(row + x, k + x, U_SIDES) || !joins(row + x, row, V_SIDES)) break grow;
        h++;
      }
      const open =
        (sides[k] & 0b0101) | (sides[k + w - 1] & 0b0010) | (sides[k + (h - 1) * S] & 0b1000);
      emit(mask[k], lights[k], open, i, j, w, h);
      for (let y = 0; y < h; y++) mask.fill(0, k + y * S, k + y * S + w);
      i += w;
    }
  }
}

/** Texture axes per normal axis: side faces put v on height so textures stand upright. */
const TEX_AXES: readonly [number, number][] = [
  [2, 1], // x faces: u = z, v = y
  [0, 2], // y faces: u = x, v = z
  [0, 1], // z faces: u = x, v = y
];

// Faces whose u above runs leftward seen from outside; flipping keeps their textures unmirrored.
const FLIP_U = [true, false, true, false, false, true];
const CHUNK_SPAN = S * POSITION_SCALE;

/** Accumulates quads into growable typed arrays. */
class MeshBuilder {
  private positions = new Uint16Array(256 * 3);
  private uvs = new Uint16Array(256 * 2);
  private faces = new Uint8Array(256);
  private layers = new Uint16Array(256);
  private ao = new Uint8Array(256);
  private light = new Uint8Array(256);
  private edges = new Uint8Array(0);
  private shifts = new Int8Array(0);
  private normals = new Int8Array(0);
  private vertices = 0;

  constructor(
    private readonly withEdges = false,
    private readonly withSoft = false,
  ) {
    if (withEdges) this.edges = new Uint8Array(256 * 4);
    if (withSoft) this.shifts = new Int8Array(256 * 3);
    if (withSoft) this.normals = new Int8Array(256 * 3);
  }

  /** One face rectangle from (u0, v0) to (u1, v1) lying in `plane` along the face's normal axis. */
  quad(
    face: number,
    plane: number,
    u0: number,
    v0: number,
    u1: number,
    v1: number,
    layer: number,
    shades: number,
    lights: number,
    open = 0,
  ): void {
    this.reserve(this.vertices + 4);
    const [d, u, v] = axes(face);
    const [tu, tv] = TEX_AXES[d];
    for (let c = 0; c < 4; c++) {
      const k = this.vertices++;
      const p = k * 3;
      this.positions[p + d] = plane;
      this.positions[p + u] = c === 1 || c === 2 ? u1 : u0;
      this.positions[p + v] = c >= 2 ? v1 : v0;
      const texU = this.positions[p + tu];
      this.uvs[k * 2] = FLIP_U[face] ? CHUNK_SPAN - texU : texU;
      this.uvs[k * 2 + 1] = this.positions[p + tv];
      this.faces[k] = face;
      this.layers[k] = layer;
      this.ao[k] = (shades >> (c * 2)) & 3;
      this.light[k] = (lights >> (c * 4)) & 15;
      if (this.withEdges)
        this.edgeDistances(k, open, c, (u1 - u0) / POSITION_SCALE, (v1 - v0) / POSITION_SCALE);
      if (this.withSoft)
        for (let a = 0; a < 3; a++) this.normals[p + a] = FACE_NORMALS[face][a] * 127;
    }
  }

  /** Moves the last quad's corners to where the soft corners put them, lit by their slopes. */
  soften(soft: SoftCorners): void {
    for (let k = this.vertices - 4; k < this.vertices; k++) {
      const [x, y, z] = [0, 1, 2].map((a) => this.positions[k * 3 + a] / POSITION_SCALE);
      this.shifts.set(soft.shift(x, y, z), k * 3);
      const n = soft.normal(x, y, z);
      if (n.some((a) => a !== 0)) this.normals.set(n, k * 3);
    }
  }

  /** Corner c's distance plus one to each open side of a w by h quad; closed sides stay 0. */
  private edgeDistances(k: number, open: number, c: number, w: number, h: number): void {
    const farU = c === 1 || c === 2;
    const farV = c >= 2;
    const along = [farU ? w : 0, farU ? 0 : w, farV ? h : 0, farV ? 0 : h];
    for (let side = 0; side < 4; side++)
      this.edges[k * 4 + side] = open & (1 << side) ? along[side] + 1 : 0;
  }

  build(): MeshData {
    const n = this.vertices;
    const indices = n > 0xffff ? new Uint32Array((n / 4) * 6) : new Uint16Array((n / 4) * 6);
    for (let q = 0; q < n / 4; q++) {
      const base = q * 4;
      const order = triangles(this.faces[base] & 1, this.flipped(base));
      for (let k = 0; k < 6; k++) indices[q * 6 + k] = base + order[k];
    }
    return {
      positions: this.positions.slice(0, n * 3),
      uvs: this.uvs.slice(0, n * 2),
      faces: this.faces.slice(0, n),
      layers: this.layers.slice(0, n),
      ao: this.ao.slice(0, n),
      light: this.light.slice(0, n),
      ...(this.withEdges && { edges: this.edges.slice(0, n * 4) }),
      ...(this.withSoft && {
        shifts: this.shifts.slice(0, n * 3),
        normals: this.normals.slice(0, n * 3),
      }),
      indices,
    };
  }

  /** Splitting along the brighter diagonal keeps corner shading smooth instead of streaky. */
  private flipped(base: number): boolean {
    const a = this.ao;
    return a[base] + a[base + 2] < a[base + 1] + a[base + 3];
  }

  private reserve(vertices: number): void {
    if (vertices <= this.faces.length) return;
    const grow = <T extends Uint8Array | Uint16Array | Int8Array>(a: T): T => {
      const next = new (a.constructor as new (n: number) => T)(a.length * 2);
      next.set(a);
      return next;
    };
    this.positions = grow(this.positions);
    this.uvs = grow(this.uvs);
    this.faces = grow(this.faces);
    this.layers = grow(this.layers);
    this.ao = grow(this.ao);
    this.light = grow(this.light);
    if (this.withEdges) this.edges = grow(this.edges);
    if (this.withSoft) this.shifts = grow(this.shifts);
    if (this.withSoft) this.normals = grow(this.normals);
  }
}

/** Index order for a quad; positive faces wind counter-clockwise from outside, negative reverse it. */
function triangles(negative: number, flip: boolean): number[] {
  const order = flip ? [1, 2, 3, 1, 3, 0] : [0, 1, 2, 0, 2, 3];
  return negative ? [order[0], order[2], order[1], order[3], order[5], order[4]] : order;
}
