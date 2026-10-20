import type { Camera, Color, Material, Object3D, Scene, Texture } from "three";

type Drawn = Material & { color?: Color; map?: Texture | null };

/** Remembers what the last drawn frame showed, so frames that would look the same can be skipped. */
export class FrameState {
  private last: number[] = [];
  private next: number[] = [];
  private forced = true;

  /** Makes the next check report a change, for things it can't see such as shader uniforms. */
  invalidate(): void {
    this.forced = true;
  }

  /** Whether the scene or camera differ from the previous call. */
  changed(scene: Scene, camera: Camera): boolean {
    const out = this.next;
    out.length = 0;
    const { position: p, quaternion: q } = camera;
    out.push(p.x, p.y, p.z, q.x, q.y, q.z, q.w);
    for (const e of camera.projectionMatrix.elements) out.push(e);
    record(scene, out);
    const changed = this.forced || !same(out, this.last);
    this.forced = false;
    this.next = this.last;
    this.last = out;
    return changed;
  }
}

/** Appends everything about a visible object and its children that changes how they look. */
function record(o: Object3D, out: number[]): void {
  out.push(o.id, o.visible ? 1 : 0);
  if (!o.visible) return;
  const { position: p, quaternion: q, scale: s } = o;
  out.push(p.x, p.y, p.z, q.x, q.y, q.z, q.w, s.x, s.y, s.z);
  const material = (o as Object3D & { material?: Drawn | Drawn[] }).material;
  for (const m of Array.isArray(material) ? material : material ? [material] : [])
    recordMaterial(m, out);
  out.push(o.children.length);
  for (const c of o.children) record(c, out);
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
