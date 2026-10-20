import {
  Frustum,
  Matrix4,
  type Camera,
  type Color,
  type Material,
  type Object3D,
  type Scene,
  type Sprite,
  type Texture,
} from "three";

type Drawn = Material & { color?: Color; map?: Texture | null };
type Drawable = Object3D & { material: Drawn | Drawn[] };

/** What differs from the last drawn frame: nothing, only objects in view, or the view itself. */
export type Change = "none" | "objects" | "view";

/** Remembers what the last drawn frame showed, so frames that would look the same can be skipped. */
export class FrameState {
  private lastView: number[] = [];
  private nextView: number[] = [];
  private lastObjects: number[] = [];
  private nextObjects: number[] = [];
  private forced = true;
  private readonly frustum = new Frustum();
  private readonly viewProjection = new Matrix4();

  /** Makes the next check report a change, for things it can't see such as shader uniforms. */
  invalidate(): void {
    this.forced = true;
  }

  /** Compares the scene and camera with the last frame passed to drew(). */
  check(scene: Scene, camera: Camera): Change {
    scene.updateMatrixWorld();
    camera.updateMatrixWorld();
    const view = this.nextView;
    view.length = 0;
    const { position: p, quaternion: q } = camera;
    view.push(p.x, p.y, p.z, q.x, q.y, q.z, q.w);
    for (const e of camera.projectionMatrix.elements) view.push(e);
    this.viewProjection.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse);
    this.frustum.setFromProjectionMatrix(this.viewProjection);
    const objects = this.nextObjects;
    objects.length = 0;
    this.record(scene, objects);
    if (this.forced || !same(view, this.lastView)) return "view";
    return same(objects, this.lastObjects) ? "none" : "objects";
  }

  /** Makes the frame last checked the one later checks compare against. */
  drew(): void {
    this.forced = false;
    const view = this.lastView;
    this.lastView = this.nextView;
    this.nextView = view;
    const objects = this.lastObjects;
    this.lastObjects = this.nextObjects;
    this.nextObjects = objects;
  }

  /** The first object that differs from the last drawn frame, to find what keeps a view busy. */
  culprit(scene: Scene): Object3D | undefined {
    const starts: [number, Object3D][] = [];
    const out: number[] = [];
    this.record(scene, out, starts);
    const last = this.lastObjects;
    let i = 0;
    while (i < out.length && out[i] === last[i]) i++;
    let found: Object3D | undefined;
    for (const [at, o] of starts) if (at <= i) found = o;
    return found;
  }

  /**
   * Appends how each shown object in view looks, so objects coming, going or changing out of
   * view leave the record alone.
   */
  private record(o: Object3D, out: number[], starts?: [number, Object3D][]): void {
    if (!o.visible) return;
    if (isDrawable(o) && this.inView(o)) {
      starts?.push([out.length, o]);
      out.push(o.id);
      for (const e of o.matrixWorld.elements) out.push(e);
      const m = o.material;
      for (const each of Array.isArray(m) ? m : [m]) recordMaterial(each, out);
    }
    if (o.userData.fixed === true) {
      this.recordFixed(o, out, starts);
      return;
    }
    for (const c of o.children) this.record(c, out, starts);
  }

  /** Notes only which children are in view, for groups whose meshes never move or change. */
  private recordFixed(o: Object3D, out: number[], starts?: [number, Object3D][]): void {
    for (const c of o.children) {
      if (!c.visible || !isDrawable(c) || !this.inView(c)) continue;
      starts?.push([out.length, c]);
      out.push(c.id);
    }
  }

  private inView(o: Drawable): boolean {
    if (!o.frustumCulled) return true;
    if ((o as Partial<Sprite>).isSprite) return this.frustum.intersectsSprite(o as Sprite);
    return this.frustum.intersectsObject(o);
  }
}

function isDrawable(o: Object3D): o is Drawable {
  return (o as Partial<Drawable>).material !== undefined;
}

// Numbers for materials, whose own ids three.js leaves out of its types.
const materialIds = new WeakMap<Material, number>();
let nextMaterialId = 0;

function materialId(m: Material): number {
  let id = materialIds.get(m);
  if (id === undefined) materialIds.set(m, (id = nextMaterialId++));
  return id;
}

function recordMaterial(m: Drawn, out: number[]): void {
  out.push(materialId(m), m.version, m.opacity, m.visible ? 1 : 0);
  if (m.color) out.push(m.color.r, m.color.g, m.color.b);
  if (m.map) out.push(m.map.id, m.map.version);
}

function same(a: readonly number[], b: readonly number[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}
