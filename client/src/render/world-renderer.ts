import type { ShaderMaterial } from "three";
import type { ChunkBatch, FarLand as FarLandMessage, WorldInfo } from "../pb/scabbage/v1/net_pb";
import type { Look } from "../looks/look";
import type { ChunkPos } from "../world/coords";
import type { RenderTable } from "../world/mesher";
import { WorldClient } from "../world/world-client";
import { buildAtlas, type Atlas } from "./atlas";
import { buildRenderTable, type BlockTypeInfo } from "./blocks";
import { blockTextures, opaqueMaterial, translucentMaterial } from "./chunk-material";
import { applyLook, lookUniforms, setLookUniforms } from "./look-shader";
import { layerColors, layerColorTexture, topColors } from "./palette";
import { ChunkMeshes } from "./chunk-meshes";
import { FarLand } from "./far-land";
import type { GameScene } from "./scene";
import { WorkerMesher } from "./worker-mesher";

export interface WorldRenderer {
  /** Starts (or restarts, after reconnecting) streaming the described world; call setBlocks too. */
  start(info: WorldInfo): void;
  /** Swaps in a new block list and textures, e.g. after an admin approves a block. */
  setBlocks(blocks: readonly BlockTypeInfo[], textures: WorldInfo["textures"]): BlockArt;
  receive(batch: ChunkBatch): void;
  receiveFar(land: FarLandMessage): void;
  /** Fetches chunks again that changed too much on the server to send block by block. */
  refetch(chunks: readonly ChunkPos[]): void;
  /** The block at world position x,y,z, or undefined if it isn't loaded. */
  blockAt(x: number, y: number, z: number): number | undefined;
  /** Changes a loaded block, e.g. from a server update or a predicted edit. */
  setBlock(x: number, y: number, z: number, block: number): void;
  /** Draws the world in another look. */
  setLook(look: Look): void;
  /** Keeps chunks around block position x,z loaded and drawn; call once per frame. */
  tick(x: number, z: number, viewDistance: number): void;
}

export interface BlockArt {
  atlas: Atlas;
  table: RenderTable;
}

/** Streams, meshes and draws the voxel world into the scene. */
export function createWorldRenderer(
  game: GameScene,
  request: (chunks: ChunkPos[]) => void,
  requestFar: (columns: { x: number; z: number }[]) => void,
  initialLook: Look,
): WorldRenderer {
  setLookUniforms(initialLook);
  const textures = blockTextures(buildAtlas([], []));
  const materials: ShaderMaterial[] = [
    opaqueMaterial(textures, initialLook),
    translucentMaterial(textures, initialLook),
  ];
  let look = initialLook;
  const meshes = new ChunkMeshes(materials[0] as ShaderMaterial, materials[1] as ShaderMaterial);
  game.scene.add(meshes.group);
  const mesher = new WorkerMesher();
  const far = new FarLand(requestFar, initialLook);
  const cover = (x: number, z: number, covered: boolean) => far.cover(x, z, covered);
  const world = new WorldClient({ send: request, mesher, view: meshes, cover });
  game.scene.add(far.group);
  let started = false;
  let art: BlockArt | undefined;

  /** Gives each layer its flat colour in the look, for untextured blocks and far land. */
  function paint(): void {
    if (!art) return;
    const colors = layerColors(art.atlas, look.surface.palette);
    lookUniforms.layerColors.value?.dispose();
    lookUniforms.layerColors.value = layerColorTexture(colors);
    far.setColors(topColors(colors, art.table));
  }

  return {
    start(info) {
      world.start({ radius: info.radius, height: info.height });
      far.start({ radius: info.radius, height: info.height });
      started = true;
    },
    setBlocks(blocks, repainted) {
      const atlas = buildAtlas(blocks, repainted);
      const table = buildRenderTable(blocks, atlas.layerOf);
      const old = materials[0]?.uniforms.blocks?.value as { dispose(): void } | undefined;
      const tex = blockTextures(atlas);
      for (const m of materials) if (m.uniforms.blocks) m.uniforms.blocks.value = tex;
      old?.dispose();
      mesher.setTable(table);
      world.setLights(table.light, table.transparent);
      art = { atlas, table };
      paint();
      game.invalidate(); // the frame check can't see textures inside shader uniforms
      return art;
    },
    receive(batch) {
      world.receive(batch.chunks.flatMap((c) => (c.pos ? [{ pos: c.pos, data: c.data }] : [])));
    },
    receiveFar: (land) => far.receive(land.tiles),
    refetch: (chunks) => world.refetch(chunks),
    blockAt: (x, y, z) => world.blockAt(x, y, z),
    setBlock: (x, y, z, block) => world.setBlock(x, y, z, block),
    setLook(next) {
      if (next === look) return;
      look = next;
      setLookUniforms(look);
      for (const m of materials) applyLook(m, look);
      far.setLook(look);
      paint();
      game.invalidate();
    },
    tick(x, z, viewDistance) {
      if (!started) return;
      world.tick(x, z, viewDistance);
      if (far.tick(x, z, viewDistance)) game.invalidate(); // the frame check can't see what it hides
    },
  };
}
