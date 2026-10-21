import { describe, expect, it } from "vitest";
import { CHUNK_SIZE as S } from "./constants";
import { Shape } from "./shapes";
import { Smooth } from "./soft-corners";
import {
  FACE_NORMALS,
  meshChunk,
  PADDED_VOLUME,
  paddedIndex,
  POSITION_SCALE,
  type MeshData,
  type RenderTable,
} from "./mesher";

const STONE = 1;
const DIRT = 2;
const WATER = 3;
const GLASS = 4;
const SLAB = 5;
const STAIRS = 6;

// Each block uses layer = id*10 + face so tests can tell faces apart.
function table(): RenderTable {
  const n = 7;
  const layers = new Int16Array(n * 6).fill(-1);
  for (let id = 1; id < n; id++) for (let f = 0; f < 6; f++) layers[id * 6 + f] = id * 10 + f;
  const transparent = Uint8Array.from([1, 0, 0, 1, 1, 1, 1]);
  const translucent = Uint8Array.from([0, 0, 0, 1, 0, 0, 0]);
  const shape = Uint8Array.from([0, 0, 0, 0, 0, Shape.SLAB, Shape.STAIRS_NORTH]);
  const { EMPTY, NATURAL, BUILT } = Smooth;
  const smooth = Uint8Array.from([EMPTY, NATURAL, NATURAL, EMPTY, BUILT, BUILT, BUILT]);
  return { layers, transparent, translucent, shape, light: new Uint8Array(n), smooth };
}

function volume(blocks: [number, number, number, number][]): Uint16Array {
  const v = new Uint16Array(PADDED_VOLUME);
  for (const [x, y, z, b] of blocks) v[paddedIndex(x, y, z)] = b;
  return v;
}

const quads = (m: MeshData) => m.indices.length / 6;

/** Vertex i of a mesh as [x, y, z] in blocks. */
const vertex = (m: MeshData, i: number) =>
  [0, 1, 2].map((a) => m.positions[i * 3 + a] / POSITION_SCALE);

/** Upward quads, as the index of each one's first vertex. */
const tops = (m: MeshData) =>
  Array.from({ length: quads(m) }, (_, q) => q * 4).filter((i) => m.faces[i] === 2);

describe("edge flags", () => {
  const edges = { edges: true, softCorners: 0 };

  it("are left out unless a look asks for them", () => {
    expect(meshChunk(volume([[5, 5, 5, STONE]]), table()).opaque.edges).toBeUndefined();
  });

  it("give each corner its distance plus one to every open side, in blocks", () => {
    const m = meshChunk(volume([[5, 5, 5, STONE]]), table(), undefined, edges).opaque;
    const [top] = tops(m);
    expect([...m.edges!.subarray(top * 4, top * 4 + 4)]).toEqual([1, 2, 1, 2]);
  });

  it("mark nothing where the surface carries on into the next chunk", () => {
    const m = meshChunk(volume([...floor(), [-1, 0, 4, STONE]]), table(), undefined, edges).opaque;
    const onEdge = tops(m).filter((i) => vertex(m, i)[0] === 0 && vertex(m, i)[2] <= 4);
    const closed = onEdge.filter((i) => m.edges![i * 4 + 2] === 0);
    expect(closed.length).toBeGreaterThan(0);
  });

  it("merge only cells whose sides agree", () => {
    const half = Array.from({ length: S }, (_, z): [number, number, number, number] => [
      -1,
      0,
      z,
      STONE,
    ]);
    const plain = meshChunk(volume([...floor(), ...half.slice(0, 8)]), table()).opaque;
    const flagged = meshChunk(volume([...floor(), ...half.slice(0, 8)]), table(), undefined, edges);
    expect(tops(plain)).toHaveLength(1);
    expect(tops(flagged.opaque)).toHaveLength(2);
  });
});

describe("soft corners", () => {
  const soft = { edges: false, softCorners: 0.5 };

  it("leave flat ground merged, moving and tilting only around a step", () => {
    const wide: [number, number, number, number][] = [];
    for (let x = -1; x <= S; x++) for (let z = -1; z <= S; z++) wide.push([x, 0, z, STONE]);
    const m = meshChunk(volume([...wide, [8, 1, 8, STONE]]), table(), undefined, soft).opaque;
    const [big] = tops(m);
    expect(vertex(m, big)).toEqual([0, 1, 0]);
    expect([...m.shifts!.subarray(big * 3, big * 3 + 3)]).toEqual([0, 0, 0]);
    expect([...m.normals!.subarray(big * 3, big * 3 + 3)]).toEqual([0, 127, 0]);
    const [peak] = topVertices(m, 9, 2, 9);
    expect([...m.shifts!.subarray(peak * 3, peak * 3 + 3)]).toEqual([-3, -3, -3]);
  });

  it("add no triangles beyond one quad per visible face", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [6, 5, 5, STONE],
      ]),
      table(),
      undefined,
      soft,
    );
    expect(quads(m.opaque)).toBe(10);
  });

  it("keep built blocks and their neighbors' shared corners still", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [6, 5, 5, GLASS],
      ]),
      table(),
      undefined,
      soft,
    );
    const [shared] = topVertices(m.opaque, 6, 6, 6);
    expect([...m.opaque.shifts!.subarray(shared * 3, shared * 3 + 3)]).toEqual([0, 0, 0]);
    expect(m.opaque.shifts!.filter((s) => s !== 0).length).toBeGreaterThan(0);
  });

  it("leave water where it is", () => {
    const m = meshChunk(volume([[5, 5, 5, WATER]]), table(), undefined, soft).translucent;
    expect(m.shifts!.every((s) => s === 0)).toBe(true);
  });
});

describe("shaped blocks", () => {
  it("draws a slab half a block tall, hiding the face on the floor", () => {
    const m = meshChunk(volume([...floor(), [5, 1, 5, SLAB]]), table()).opaque;
    const slabTop = topVertices(m, 5, 1.5, 5).concat(topVertices(m, 6, 1.5, 6));
    expect(slabTop).toHaveLength(2);
    expect(quads(m)).toBe(6 + 5);
  });

  it("draws stairs as a lower slab and an upper back half", () => {
    const m = meshChunk(volume([[5, 5, 5, STAIRS]]), table()).opaque;
    expect(quads(m)).toBe(12);
    const ys = new Set(Array.from({ length: m.faces.length }, (_, i) => vertex(m, i)[1]));
    expect([...ys].sort()).toEqual([5, 5.5, 6]);
  });

  it("shows a full block's face beside a shape but hides the shape's face against it", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [6, 5, 5, SLAB],
      ]),
      table(),
    ).opaque;
    expect(quads(m)).toBe(6 + 5);
  });
});

describe("block light", () => {
  it("lights faces near a glow and leaves the rest dark", () => {
    const light = new Uint8Array(PADDED_VOLUME);
    light[paddedIndex(5, 1, 5)] = 10;
    const m = meshChunk(volume(floor()), table(), light).opaque;
    const near = topVertices(m, 5, 1, 5).concat(topVertices(m, 6, 1, 6));
    expect(near.some((i) => m.light[i] > 0)).toBe(true);
    expect(topVertices(m, 0, 1, 0).every((i) => m.light[i] === 0)).toBe(true);
  });

  it("is dark everywhere without a light volume", () => {
    const m = meshChunk(volume(floor()), table()).opaque;
    expect(m.light.every((l) => l === 0)).toBe(true);
  });
});

describe("meshChunk", () => {
  it("draws nothing for empty air", () => {
    const { opaque, translucent } = meshChunk(new Uint16Array(PADDED_VOLUME), table());
    expect(quads(opaque) + quads(translucent)).toBe(0);
  });

  it("draws six faces for a lone block", () => {
    const { opaque, translucent } = meshChunk(volume([[5, 5, 5, STONE]]), table());
    expect(quads(opaque)).toBe(6);
    expect(quads(translucent)).toBe(0);
    expect(opaque.positions.length).toBe(24 * 3);
  });

  it("merges a flat slab into one quad per side", () => {
    const slab: [number, number, number, number][] = [];
    for (let x = 0; x < S; x++) for (let z = 0; z < S; z++) slab.push([x, 0, z, STONE]);
    expect(quads(meshChunk(volume(slab), table()).opaque)).toBe(6);
  });

  it("does not merge different blocks and hides faces they share", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [6, 5, 5, DIRT],
      ]),
      table(),
    ).opaque;
    expect(quads(m)).toBe(10);
  });

  it("hides faces covered by a neighboring chunk's block", () => {
    const m = meshChunk(
      volume([
        [S - 1, 5, 5, STONE],
        [S, 5, 5, STONE],
      ]),
      table(),
    ).opaque;
    expect(quads(m)).toBe(5);
  });

  it("puts water in the translucent pass and shows land beneath it", () => {
    const { opaque, translucent } = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [5, 6, 5, WATER],
      ]),
      table(),
    );
    expect(quads(opaque)).toBe(6);
    expect(quads(translucent)).toBe(5);
  });

  it("hides faces between two blocks of the same see-through kind", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, GLASS],
        [6, 5, 5, GLASS],
      ]),
      table(),
    ).opaque;
    expect(quads(m)).toBe(6);
  });

  it("winds every triangle to face outward", () => {
    const m = meshChunk(volume([[5, 5, 5, STONE]]), table()).opaque;
    for (let q = 0; q < quads(m); q++) {
      const [a, b, c] = [0, 1, 2].map((k) => vertex(m, m.indices[q * 6 + k]));
      const u = [b[0] - a[0], b[1] - a[1], b[2] - a[2]];
      const v = [c[0] - a[0], c[1] - a[1], c[2] - a[2]];
      const cross = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
      ];
      const n = FACE_NORMALS[m.faces[m.indices[q * 6]]];
      expect(cross[0] * n[0] + cross[1] * n[1] + cross[2] * n[2]).toBeGreaterThan(0);
    }
  });

  it("tags each face with its texture layer", () => {
    const m = meshChunk(volume([[5, 5, 5, DIRT]]), table()).opaque;
    for (let i = 0; i < m.faces.length; i++) expect(m.layers[i]).toBe(DIRT * 10 + m.faces[i]);
  });

  it("maps texture v to height on side faces so textures stay upright", () => {
    const m = meshChunk(
      volume([
        [5, 5, 5, STONE],
        [5, 6, 5, STONE],
        [5, 7, 5, STONE],
      ]),
      table(),
    ).opaque;
    for (let i = 0; i < m.faces.length; i++) {
      if (m.faces[i] === 0) {
        expect(m.uvs[i * 2 + 1]).toBe(m.positions[i * 3 + 1]);
      }
    }
  });

  it("shows every face's texture unmirrored from outside, so digits read right", () => {
    const m = meshChunk(volume([[5, 5, 5, STONE]]), table()).opaque;
    for (let q = 0; q < quads(m); q++) {
      const [p0, p1, p3] = [q * 4, q * 4 + 1, q * 4 + 3].map((i) => vertex(m, i));
      const uv = (i: number) => [m.uvs[i * 2], m.uvs[i * 2 + 1]];
      const [[u0, v0], [u1, v1], [u3, v3]] = [q * 4, q * 4 + 1, q * 4 + 3].map(uv);
      const turn = (u1 - u0) * (v3 - v0) - (u3 - u0) * (v1 - v0);
      const [e1, e2] = [p1.map((c, a) => c - p0[a]), p3.map((c, a) => c - p0[a])];
      const n = FACE_NORMALS[m.faces[q * 4]];
      const out = [1, 2, 0].reduce(
        (s, a, i) => s + n[i] * (e1[a] * e2[(a + 1) % 3] - e1[(a + 1) % 3] * e2[a]),
        0,
      );
      expect(out * turn, `face ${m.faces[q * 4]}`).toBeGreaterThan(0);
    }
  });

  it("leaves a lone block fully lit", () => {
    const m = meshChunk(volume([[5, 5, 5, STONE]]), table()).opaque;
    expect([...m.ao].every((a) => a === 3)).toBe(true);
  });

  it("darkens floor corners tucked against walls", () => {
    const m = meshChunk(volume([...floor(), [4, 1, 5, STONE], [5, 1, 4, STONE]]), table()).opaque;
    const at = (x: number, z: number) => topVertices(m, x, 1, z).map((i) => m.ao[i]);
    expect(at(5, 5)).toEqual([0, 0]); // walls on two sides
    expect(at(4, 5).every((a) => a < 3)).toBe(true); // touches one wall
    expect(at(10, 10).every((a) => a === 3)).toBe(true);
  });

  it("splits shaded quads along their brighter diagonal", () => {
    const m = meshChunk(volume([...floor(), [4, 1, 5, STONE], [5, 1, 4, STONE]]), table()).opaque;
    for (let q = 0; q < quads(m); q++) {
      const tri = (t: number) => [0, 1, 2].map((k) => m.indices[q * 6 + t * 3 + k]);
      const diagonal = tri(0).filter((i) => tri(1).includes(i));
      const others = tri(0)
        .concat(tri(1))
        .filter((i) => !diagonal.includes(i));
      const sum = (ids: number[]) => ids.reduce((s, i) => s + m.ao[i], 0);
      expect(sum(diagonal)).toBeGreaterThanOrEqual(sum(others));
    }
  });
});

/** A stone floor across the whole chunk at y = 0. */
function floor(): [number, number, number, number][] {
  const out: [number, number, number, number][] = [];
  for (let x = 0; x < S; x++) for (let z = 0; z < S; z++) out.push([x, 0, z, STONE]);
  return out;
}

/** Vertices of upward faces sitting at corner (x, y, z). */
function topVertices(m: MeshData, x: number, y: number, z: number): number[] {
  const out: number[] = [];
  for (let i = 0; i < m.faces.length; i++) {
    const [vx, vy, vz] = vertex(m, i);
    if (m.faces[i] === 2 && vx === x && vy === y && vz === z) out.push(i);
  }
  return out;
}
