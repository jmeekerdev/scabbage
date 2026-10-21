import {
  BufferAttribute,
  BufferGeometry,
  Group,
  Mesh,
  Sphere,
  Vector3,
  type Material,
} from "three";
import { CHUNK_SIZE } from "../world/constants";
import type { ChunkKey, ChunkPos } from "../world/coords";
import type { ChunkMesh, MeshData } from "../world/mesher";
import type { ChunkView } from "../world/world-client";

// Every chunk mesh fits in this sphere, soft corners included, so three.js can frustum-cull without computing bounds.
const BOUNDS = new Sphere(
  new Vector3(CHUNK_SIZE / 2, CHUNK_SIZE / 2, CHUNK_SIZE / 2),
  CHUNK_SIZE * 0.9,
);

/** Shows chunk meshes in the scene, freeing GPU buffers when chunks go away. */
export class ChunkMeshes implements ChunkView {
  readonly group = new Group();
  private readonly meshes = new Map<ChunkKey, Mesh[]>();

  constructor(
    private readonly opaque: Material,
    private readonly translucent: Material,
  ) {
    this.group.name = "chunks";
    this.group.userData.fixed = true; // meshes never move, so frame checks only see which are in view
  }

  show(key: ChunkKey, pos: ChunkPos, mesh: ChunkMesh): void {
    this.remove(key);
    const parts = [
      toMesh(mesh.opaque, this.opaque, pos),
      toMesh(mesh.translucent, this.translucent, pos),
    ].filter((m): m is Mesh => m !== undefined);
    if (parts.length === 0) return;
    this.group.add(...parts);
    this.meshes.set(key, parts);
  }

  remove(key: ChunkKey): void {
    for (const m of this.meshes.get(key) ?? []) {
      this.group.remove(m);
      m.geometry.dispose();
    }
    this.meshes.delete(key);
  }

  clear(): void {
    for (const key of [...this.meshes.keys()]) this.remove(key);
  }
}

function toMesh(data: MeshData, material: Material, pos: ChunkPos): Mesh | undefined {
  if (data.indices.length === 0) return undefined;
  const g = new BufferGeometry();
  g.setAttribute("position", new BufferAttribute(data.positions, 3));
  g.setAttribute("uv", new BufferAttribute(data.uvs, 2));
  g.setAttribute("face", new BufferAttribute(data.faces, 1));
  g.setAttribute("layer", new BufferAttribute(data.layers, 1));
  g.setAttribute("ao", new BufferAttribute(data.ao, 1));
  g.setAttribute("light", new BufferAttribute(data.light, 1));
  if (data.edges) g.setAttribute("edge", new BufferAttribute(data.edges, 4));
  if (data.shifts) g.setAttribute("shift", new BufferAttribute(data.shifts, 3));
  if (data.normals) g.setAttribute("normal", new BufferAttribute(data.normals, 3, true));
  g.setIndex(new BufferAttribute(data.indices, 1));
  g.boundingSphere = BOUNDS;
  const mesh = new Mesh(g, material);
  mesh.position.set(pos.x * CHUNK_SIZE, pos.y * CHUNK_SIZE, pos.z * CHUNK_SIZE);
  mesh.matrixAutoUpdate = false;
  mesh.updateMatrix();
  return mesh;
}
