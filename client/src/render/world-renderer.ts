import type { DataArrayTexture, ShaderMaterial } from "three";
import type { ChunkBatch, FarLand as FarLandMessage, WorldInfo } from "../pb/scabbage/v1/net_pb";
import type { Look } from "../looks/look";
import type { ChunkPos } from "../world/coords";
import type { RenderTable } from "../world/mesher";
import { cornerShift, Smooth } from "../world/soft-corners";
import type { Roofs } from "./precipitation";
import { WorldClient } from "../world/world-client";
import { buildAtlas, type Atlas } from "./atlas";
import { buildRenderTable, type BlockTypeInfo } from "./blocks";
import { blockTextures, opaqueMaterial, translucentMaterial } from "./chunk-material";
import { applyLook, lookUniforms, meshOptions, setLookUniforms } from "./look-shader";
import { layerColors, layerColorTexture, topColors } from "./palette";
import { ChunkMeshes } from "./chunk-meshes";
import { setEntityLook } from "./entity-material";
import { FarLand } from "./far-land";
import type { GameScene } from "./scene";
import { Veil } from "./veil";
import { WorkerMesher } from "./worker-mesher";

// Longest the veil waits for chunks to rebuild before lifting anyway.
const REBUILD_WAIT_MS = 4000;

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
  /** The highest block in each loaded column, for rain to stop at. */
  readonly roofs: Roofs;
  /** How far soft corners moved world corner x,y,z in the drawn look, in blocks. */
  softCorner(x: number, y: number, z: number): number[];
  /** Changes a loaded block, e.g. from a server update or a predicted edit. */
  setBlock(x: number, y: number, z: number, block: number): void;
  /** Draws the world in another look. */
  setLook(look: Look): void;
  /** Keeps chunks around block position x,z loaded and drawn; call once per frame. */
  tick(x: number, z: number, viewDistance: number): void;
}

function sameMesh(a: Look, b: Look): boolean {
  const [x, y] = [meshOptions(a), meshOptions(b)];
  return x.edges === y.edges && x.softCorners === y.softCorners;
}

/** The opaque and translucent chunk materials for a look. */
function lookMaterials(textures: DataArrayTexture, look: Look): [ShaderMaterial, ShaderMaterial] {
  return [opaqueMaterial(textures, look), translucentMaterial(textures, look)];
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
  setEntityLook(initialLook);
  let textures = blockTextures(buildAtlas([], []));
  let materials = lookMaterials(textures, initialLook);
  let retired: ShaderMaterial[] = []; // still drawing chunks not yet rebuilt for the current look
  let look = initialLook;
  const meshes = new ChunkMeshes(...materials);
  game.scene.add(meshes.group);
  const mesher = new WorkerMesher();
  mesher.setOptions(meshOptions(initialLook));
  const far = new FarLand(requestFar, initialLook);
  const cover = (x: number, z: number, covered: boolean) => far.cover(x, z, covered);
  const world = new WorldClient({ send: request, mesher, view: meshes, cover });
  game.scene.add(far.group);
  let started = false;
  let art: BlockArt | undefined;
  const veil = new Veil();
  let waiting: Look | undefined; // shown once the veil has closed
  let rebuildingSince: number | undefined;

  function show(next: Look): void {
    look = next;
    setLookUniforms(look);
    setEntityLook(look);
    for (const m of materials) applyLook(m, look);
    far.setLook(look);
    paint();
    game.invalidate();
  }

  /** Swaps in a look with other mesh data once the veil has closed, and lifts it when chunks are rebuilt. */
  function rebuildBehindVeil(now: number): void {
    if (waiting && veil.closed(now)) {
      retired.push(...materials);
      materials = lookMaterials(textures, waiting);
      meshes.useMaterials(...materials);
      show(waiting);
      waiting = undefined;
      mesher.setOptions(meshOptions(look));
      world.remeshAll();
      rebuildingSince = now;
    }
    if (retired.length && world.settled()) {
      for (const m of retired) m.dispose();
      retired = [];
    }
    const rebuilt = world.settled() || now - (rebuildingSince ?? now) > REBUILD_WAIT_MS;
    if (rebuildingSince !== undefined && rebuilt) {
      veil.open(now);
      rebuildingSince = undefined;
    }
    const amount = veil.amount(now);
    if (amount === lookUniforms.veil.value) return;
    lookUniforms.veil.value = amount;
    game.invalidate();
  }

  /** Gives each layer its flat color in the look, for untextured blocks and far land. */
  function paint(): void {
    if (!art) return;
    const colors = layerColors(art.atlas, look.surface.palette, look.surface.solid);
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
      const old = textures;
      textures = blockTextures(atlas);
      for (const m of [...materials, ...retired]) m.uniforms.blocks.value = textures;
      old.dispose();
      mesher.setTable(table);
      world.setLights(table.light, table.passes);
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
    roofs: world,
    softCorner(x, y, z) {
      const { softCorners } = meshOptions(look);
      const smooth = art?.table.smooth;
      if (!softCorners || !smooth) return [0, 0, 0];
      const kindAt = (bx: number, by: number, bz: number) =>
        smooth[world.blockAt(bx, by, bz) ?? -1] ?? Smooth.BUILT;
      return cornerShift(x, y, z, kindAt, softCorners);
    },
    setBlock: (x, y, z, block) => world.setBlock(x, y, z, block),
    setLook(next) {
      if (next === (waiting ?? look)) return;
      if (sameMesh(next, look)) {
        waiting = undefined;
        show(next);
        return;
      }
      waiting = next;
      veil.close(performance.now());
    },
    tick(x, z, viewDistance) {
      if (!started) return;
      world.tick(x, z, viewDistance);
      if (far.tick(x, z, viewDistance)) game.invalidate(); // the frame check can't see what it hides
      rebuildBehindVeil(performance.now());
    },
  };
}
