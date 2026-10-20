import { describe, expect, it } from "vitest";
import { Shape } from "./shapes";
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
  return { layers, transparent, translucent, shape, light: new Uint8Array(n) };
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
    for (let x = 0; x < 16; x++) for (let z = 0; z < 16; z++) slab.push([x, 0, z, STONE]);
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
        [15, 5, 5, STONE],
        [16, 5, 5, STONE],
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
  for (let x = 0; x < 16; x++) for (let z = 0; z < 16; z++) out.push([x, 0, z, STONE]);
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
