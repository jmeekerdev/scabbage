import { BoxGeometry, Group, type Material, Mesh, type Scene } from "three";
import { MobKind } from "../pb/scabbage/v1/net_pb";
import type { MobState } from "../player/mobs";
import { daylit, shaded } from "./avatars";

const box = (w: number, h: number, d: number) => shaded(new BoxGeometry(w, h, d));
const paint = (color: number, opacity = 1) =>
  daylit({ color, vertexColors: true, transparent: opacity < 1, opacity });

// Every mob shares these.
const shapes = {
  wool: box(0.8, 0.6, 1),
  head: box(0.42, 0.42, 0.42),
  leg: box(0.18, 0.45, 0.18),
  jelly: box(0.8, 0.8, 0.8),
  core: box(0.4, 0.4, 0.4),
  eye: box(0.12, 0.16, 0.02),
};
const paints = {
  wool: paint(0xf4f4f0),
  skin: paint(0xe0b8a0), // a sheared sheep
  face: paint(0x4a4440),
  jelly: paint(0x66d066, 0.7),
  core: paint(0x2e8b3e),
  eye: paint(0x101010),
  hurt: paint(0xff5050),
};

function part(shape: BoxGeometry, material: Material, x: number, y: number, z: number): Mesh {
  const m = new Mesh(shape, material);
  m.position.set(x, y, z);
  m.userData.paint = material;
  return m;
}

function sheep(): Group {
  const g = new Group();
  g.add(part(shapes.wool, paints.wool, 0, 0.75, 0));
  g.add(part(shapes.head, paints.face, 0, 1, 0.6));
  for (const [x, z] of [
    [-0.25, -0.35],
    [0.25, -0.35],
    [-0.25, 0.35],
    [0.25, 0.35],
  ] as const)
    g.add(part(shapes.leg, paints.face, x, 0.225, z));
  return g;
}

function slime(): Group {
  const g = new Group();
  g.add(part(shapes.core, paints.core, 0, 0.4, 0));
  g.add(part(shapes.jelly, paints.jelly, 0, 0.4, 0)); // after the core so it draws over it
  g.add(part(shapes.eye, paints.eye, -0.15, 0.5, 0.41));
  g.add(part(shapes.eye, paints.eye, 0.15, 0.5, 0.41));
  return g;
}

interface Figure {
  group: Group;
  hopMs: number; // time spent moving, for the slime's bounce
  last?: MobState;
}

/** Sheep and slimes, built from a few shared boxes. */
export class MobView {
  private readonly figures = new Map<number, Figure>();

  constructor(private readonly scene: Scene) {}

  /** Draws every mob as sampled for this frame; call once per frame. */
  update(mobs: readonly MobState[], dtMs: number): void {
    const seen = new Set<number>();
    for (const m of mobs) {
      seen.add(m.id);
      this.draw(this.figures.get(m.id) ?? this.add(m), m, dtMs);
    }
    for (const [id, f] of this.figures)
      if (!seen.has(id)) {
        this.scene.remove(f.group);
        this.figures.delete(id);
      }
  }

  private add(m: MobState): Figure {
    const f = { group: m.kind === MobKind.SLIME ? slime() : sheep(), hopMs: 0 };
    f.group.name = `mob ${m.id}`;
    this.scene.add(f.group);
    this.figures.set(m.id, f);
    return f;
  }

  private draw(f: Figure, m: MobState, dtMs: number): void {
    const moving = f.last !== undefined && (f.last.x !== m.x || f.last.z !== m.z);
    f.hopMs = moving ? f.hopMs + dtMs : 0;
    f.last = m;
    f.group.position.set(m.x, m.y, m.z);
    f.group.rotation.y = m.yaw;
    if (m.kind === MobKind.SLIME) bounce(f.group, f.hopMs);
    const wool = f.group.children[0] as Mesh;
    if (m.kind === MobKind.SHEEP) wool.userData.paint = m.sheared ? paints.skin : paints.wool;
    for (const c of f.group.children) {
      const mesh = c as Mesh;
      mesh.material = m.hurt ? paints.hurt : (mesh.userData.paint as Material);
    }
  }
}

/** Slimes hop along, squashing as they land. */
function bounce(g: Group, hopMs: number): void {
  if (hopMs === 0) {
    g.scale.set(1, 1, 1);
    return;
  }
  const phase = Math.abs(Math.sin((hopMs / 400) * Math.PI));
  g.position.y += 0.3 * phase;
  g.scale.set(1 + 0.1 * (1 - phase), 0.85 + 0.15 * phase, 1 + 0.1 * (1 - phase));
}
