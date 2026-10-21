import { MeshBasicMaterial, type Mesh } from "three";
import { describe, expect, it } from "vitest";
import type { ChunkMesh, MeshData } from "../world/mesher";
import { ChunkMeshes } from "./chunk-meshes";

/** One triangle of opaque mesh data and nothing translucent. */
function triangle(): ChunkMesh {
  const data = (indices: number[]): MeshData => ({
    positions: new Uint16Array(9),
    uvs: new Uint16Array(6),
    faces: new Uint8Array(3),
    layers: new Uint16Array(3),
    ao: new Uint8Array(3),
    light: new Uint8Array(3),
    indices: Uint32Array.from(indices),
  });
  return { opaque: data([0, 1, 2]), translucent: data([]) };
}

const pos = { x: 0, y: 0, z: 0 };

describe("ChunkMeshes", () => {
  it("draws new meshes with new materials while shown ones keep theirs", () => {
    const [before, after, see] = [1, 2, 3].map(() => new MeshBasicMaterial());
    const meshes = new ChunkMeshes(before, see);
    meshes.show("a", pos, triangle());
    meshes.useMaterials(after, see);
    meshes.show("b", pos, triangle());
    const [a, b] = meshes.group.children as Mesh[];
    expect(a.material).toBe(before);
    expect(b.material).toBe(after);
  });
});
