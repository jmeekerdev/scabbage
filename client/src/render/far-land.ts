import {
  BufferAttribute,
  BufferGeometry,
  FrontSide,
  Group,
  Mesh,
  ShaderMaterial,
  UniformsLib,
  UniformsUtils,
  DataTexture,
  RedFormat,
} from "three";
import { CHUNK_SIZE } from "../world/constants";
import { FAR_REGION, farRegionMesh, type FarTile } from "../world/far-mesh";
import { ChunkStreamer, type WorldBounds } from "../world/streamer";
import type { Look } from "../looks/look";
import { daylight } from "./chunk-material";
import { applyLook, LOOK_LIGHT_GLSL, lookFeatures, lookUniforms } from "./look-shader";

/** How many chunks away far land reaches for a view distance in chunks. */
export const farDistance = (viewDistance: number) => viewDistance * 3;

// Rebuilding a region takes about a millisecond, so a few per frame keep frames smooth.
const REBUILDS_PER_TICK = 2;
const REGION_BLOCKS = FAR_REGION * CHUNK_SIZE;
// Columns the chunks draw are marked in a square this many columns across, wrapping around; it
// is wider than the drawn chunks and far land together, so no two of them share a mark.
const MASK = 64;

export interface ArrivedTile extends FarTile {
  pos?: { x: number; z: number };
}

type Column = { x: number; z: number };

/** Keeps a rough sketch of the land past the view distance fetched and drawn, fading into fog. */
export class FarLand {
  readonly group = new Group();
  private readonly tiles = new Map<string, FarTile>();
  private readonly regions = new Map<string, Mesh>();
  private readonly dirty = new Set<string>();
  private streamer = new ChunkStreamer({ radius: 0, height: CHUNK_SIZE });
  private colors: Uint8Array = new Uint8Array(0);
  private readonly mask = coverMask();
  private readonly material: ShaderMaterial;
  private masked = false; // the mask changed since the last tick
  private readonly tileAt = (x: number, z: number) => this.tiles.get(`${x},${z}`);

  constructor(
    private readonly send: (columns: Column[]) => void,
    look: Look,
  ) {
    this.material = farMaterial(this.mask, look);
    this.group.name = "far land";
    this.group.userData.fixed = true; // regions are replaced, never changed in place
  }

  /** Begins a fresh world, forgetting everything from before. */
  start(bounds: WorldBounds): void {
    this.streamer = new ChunkStreamer(
      { radius: bounds.radius, height: CHUNK_SIZE },
      { batchSize: 64 },
    );
    this.tiles.clear();
    this.dirty.clear();
    for (const key of [...this.regions.keys()]) this.removeRegion(key);
    this.mask.image.data?.fill(0);
    this.mask.needsUpdate = true;
  }

  /** Hides or shows far land over a chunk column, as the chunks there start or stop being drawn. */
  cover(x: number, z: number, covered: boolean): void {
    const wrap = (v: number) => ((v % MASK) + MASK) % MASK;
    this.mask.image.data![wrap(z) * MASK + wrap(x)] = covered ? 255 : 0;
    this.mask.needsUpdate = true;
    this.masked = true;
  }

  /** Lights far land the way a look lights blocks. */
  setLook(look: Look): void {
    applyLook(this.material, look);
  }

  /** Each block's top color, RGB per block ID; every region is redrawn with them. */
  setColors(colors: Uint8Array): void {
    this.colors = colors;
    for (const key of this.tiles.keys()) this.touch(key);
  }

  receive(tiles: readonly ArrivedTile[]): void {
    const arrived = tiles.flatMap((t) => (t.pos ? [{ ...t, pos: t.pos }] : []));
    this.streamer.received(arrived.map(({ pos }) => ({ x: pos.x, y: 0, z: pos.z })));
    for (const { pos, tops, blocks } of arrived) {
      const key = `${pos.x},${pos.z}`;
      this.tiles.set(key, { tops, blocks });
      this.touch(key);
    }
  }

  /** Call every frame with the player's block position; returns true when what it hides changed. */
  tick(x: number, z: number, viewDistance: number): boolean {
    const cx = Math.floor(x / CHUNK_SIZE);
    const cz = Math.floor(z / CHUNK_SIZE);
    for (const key of this.streamer.update(cx, cz, farDistance(viewDistance) - 1)) {
      const [tx, , tz] = key.split(",");
      this.tiles.delete(`${tx},${tz}`);
      this.touch(`${tx},${tz}`);
    }
    for (let b = this.streamer.nextBatch(); b; b = this.streamer.nextBatch())
      this.send(b.map(({ x, z }) => ({ x, z })));
    this.rebuildSome();
    const masked = this.masked;
    this.masked = false;
    return masked;
  }

  /** Marks the regions a tile's samples are drawn in: its own, and those it closes the gap for. */
  private touch(tileKey: string): void {
    const [tx, tz] = tileKey.split(",").map(Number);
    const rx = Math.floor(tx / FAR_REGION);
    const rz = Math.floor(tz / FAR_REGION);
    const westEdge = tx === rx * FAR_REGION;
    const northEdge = tz === rz * FAR_REGION;
    this.dirty.add(`${rx},${rz}`);
    if (westEdge) this.dirty.add(`${rx - 1},${rz}`);
    if (northEdge) this.dirty.add(`${rx},${rz - 1}`);
    if (westEdge && northEdge) this.dirty.add(`${rx - 1},${rz - 1}`);
  }

  private rebuildSome(): void {
    let budget = REBUILDS_PER_TICK;
    for (const key of this.dirty) {
      if (budget-- <= 0) return;
      this.dirty.delete(key);
      this.rebuild(key);
    }
  }

  private rebuild(key: string): void {
    this.removeRegion(key);
    const [rx, rz] = key.split(",").map(Number);
    const data = farRegionMesh(this.tileAt, rx, rz, this.colors);
    if (data.indices.length === 0) return;
    const g = new BufferGeometry();
    g.setAttribute("position", new BufferAttribute(data.positions, 3));
    g.setAttribute("color", new BufferAttribute(data.colors, 3, true));
    g.setAttribute("normal", new BufferAttribute(data.normals, 3, true));
    g.setIndex(new BufferAttribute(data.indices, 1));
    g.computeBoundingSphere();
    const mesh = new Mesh(g, this.material);
    mesh.position.set(rx * REGION_BLOCKS, 0, rz * REGION_BLOCKS);
    mesh.matrixAutoUpdate = false;
    mesh.updateMatrix();
    this.group.add(mesh);
    this.regions.set(key, mesh);
  }

  private removeRegion(key: string): void {
    const mesh = this.regions.get(key);
    if (!mesh) return;
    this.group.remove(mesh);
    mesh.geometry.dispose();
    this.regions.delete(key);
  }
}

const vertexShader = /* glsl */ `
attribute vec3 color;
varying vec3 vColor;
varying vec3 vNormal;
varying vec3 vWorld;
varying vec2 vColumn;
#include <fog_pars_vertex>
void main() {
  vColor = pow(color, vec3(2.2)); // sRGB to linear, like the block textures
  vNormal = normal;
  vec4 world = modelMatrix * vec4(position, 1.0);
  vWorld = world.xyz;
  vColumn = world.xz / CHUNK_SIZE;
  vec4 mvPosition = viewMatrix * world;
  gl_Position = projectionMatrix * mvPosition;
  #include <fog_vertex>
}`;

const fragmentShader = /* glsl */ `
uniform float daylight;
uniform sampler2D covered; // marks the columns the chunks draw
varying vec3 vColor;
varying vec3 vNormal;
varying vec3 vWorld;
varying vec2 vColumn;
#include <fog_pars_fragment>
${LOOK_LIGHT_GLSL}
void main() {
  ivec2 column = ivec2(mod(floor(vColumn), MASK));
  if (texelFetch(covered, column, 0).r > 0.5) discard;
  vec3 n = normalize(vNormal);
  vec3 shine = lookShine(n, normalize(cameraPosition - vWorld));
  gl_FragColor = vec4((vColor * lookLight(n) + shine) * daylight, 1.0);
  #include <colorspace_fragment>
  #include <fog_fragment>
}`;

function coverMask(): DataTexture {
  const mask = new DataTexture(new Uint8Array(MASK * MASK), MASK, MASK, RedFormat);
  mask.needsUpdate = true;
  return mask;
}

function farMaterial(covered: DataTexture, look: Look): ShaderMaterial {
  const uniforms = UniformsUtils.merge([UniformsLib.fog]);
  uniforms.daylight = daylight;
  Object.assign(uniforms, lookUniforms);
  uniforms.covered = { value: covered };
  return new ShaderMaterial({
    uniforms,
    vertexShader,
    fragmentShader,
    fog: true,
    side: FrontSide,
    defines: { CHUNK_SIZE: CHUNK_SIZE.toFixed(1), MASK: MASK.toFixed(1), ...lookFeatures(look) },
  });
}
