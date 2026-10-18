import type { ChunkBatch, WorldInfo } from "../pb/scabbage/v1/net_pb";
import type { ChunkPos } from "../world/coords";
import { WorldClient } from "../world/world-client";
import { buildRenderTable } from "./blocks";
import { blockTextures, opaqueMaterial, translucentMaterial } from "./chunk-material";
import { ChunkMeshes } from "./chunk-meshes";
import type { GameScene } from "./scene";
import { WorkerMesher } from "./worker-mesher";

export interface WorldRenderer {
  /** Starts (or restarts, after reconnecting) streaming the described world. */
  start(info: WorldInfo): void;
  receive(batch: ChunkBatch): void;
  /** Keeps chunks around block position x,z loaded and drawn; call once per frame. */
  tick(x: number, z: number, viewDistance: number): void;
}

/** Streams, meshes and draws the voxel world into the scene. */
export function createWorldRenderer(
  game: GameScene,
  request: (chunks: ChunkPos[]) => void,
): WorldRenderer {
  const textures = blockTextures();
  const meshes = new ChunkMeshes(opaqueMaterial(textures), translucentMaterial(textures));
  game.scene.add(meshes.group);
  const mesher = new WorkerMesher();
  const world = new WorldClient({ send: request, mesher, view: meshes });
  let started = false;

  return {
    start(info) {
      mesher.setTable(buildRenderTable(info.blocks));
      world.start({ radius: info.radius, height: info.height });
      started = true;
    },
    receive(batch) {
      world.receive(batch.chunks.flatMap((c) => (c.pos ? [{ pos: c.pos, data: c.data }] : [])));
    },
    tick(x, z, viewDistance) {
      if (started) world.tick(x, z, viewDistance);
    },
  };
}
