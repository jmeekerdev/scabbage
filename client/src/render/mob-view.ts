import { BoxGeometry, Group, type Material, Mesh, type Scene } from "three";
import { MobKind } from "../pb/scabbage/v1/net_pb";
import { MOB_SIZE, type MobState } from "../player/mobs";
import { litMesh } from "./entity-material";
import { METER } from "../world/constants";

const box = (w: number, h: number, d: number) => new BoxGeometry(w, h, d);
const paint = (color: number, opacity = 1) => litMesh({ color, transparent: opacity < 1, opacity });

// Every mob shares these.
const shapes = {
  wool: box(0.8, 0.6, 1),
  head: box(0.42, 0.42, 0.42),
  leg: box(0.18, 0.45, 0.18),
  jelly: box(0.8, 0.8, 0.8),
  core: box(0.4, 0.4, 0.4),
  eye: box(0.12, 0.16, 0.02),
  crownBand: box(0.5, 0.08, 0.5),
  crownPoint: box(0.08, 0.14, 0.08),
  golemBody: box(1.2, 1.1, 0.7),
  golemHead: box(0.6, 0.55, 0.55),
  golemArm: box(0.34, 1.2, 0.34),
  golemLeg: box(0.42, 0.95, 0.42),
};
const paints = {
  wool: paint(0xf4f4f0),
  skin: paint(0xe0b8a0), // a sheared sheep
  face: paint(0x4a4440),
  jelly: paint(0x66d066, 0.7),
  core: paint(0x2e8b3e),
  eye: paint(0x101010),
  hurt: paint(0xff5050),
  gold: paint(0xf0c43c),
  rock: paint(0x8a8a90),
  mossy: paint(0x6f8a5a),
  glow: paint(0xffa030),
};

function part(shape: BoxGeometry, material: Material, x: number, y: number, z: number): Mesh {
  const m = new Mesh(shape, material);
  m.position.set(x, y, z);
  m.userData.paint = material;
  return m;
}

const SHEEP_MODELED = 1.3; // meters tall, as sheep() builds it

/** Shrinks models built bigger than their mob's body. */
const MODEL_SCALE: Partial<Record<MobKind, number>> = {
  [MobKind.SHEEP]: MOB_SIZE[MobKind.SHEEP].height / METER / SHEEP_MODELED,
};

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

const KING_SIZE = 2.5; // a king slime is a slime this many times bigger

function kingSlime(): Group {
  const g = slime();
  g.add(part(shapes.crownBand, paints.gold, 0, 0.84, 0));
  for (const [x, z] of [
    [-0.21, -0.21],
    [0.21, -0.21],
    [-0.21, 0.21],
    [0.21, 0.21],
  ] as const)
    g.add(part(shapes.crownPoint, paints.gold, x, 0.95, z));
  const outer = new Group();
  g.scale.setScalar(KING_SIZE);
  outer.add(g);
  return outer;
}

function golem(): Group {
  const g = new Group();
  g.add(part(shapes.golemBody, paints.rock, 0, 1.5, 0));
  g.add(part(shapes.golemHead, paints.mossy, 0, 2.33, 0.05));
  g.add(part(shapes.eye, paints.glow, -0.14, 2.38, 0.33));
  g.add(part(shapes.eye, paints.glow, 0.14, 2.38, 0.33));
  for (const x of [-0.78, 0.78]) g.add(part(shapes.golemArm, paints.rock, x, 1.35, 0));
  for (const x of [-0.3, 0.3]) g.add(part(shapes.golemLeg, paints.mossy, x, 0.475, 0));
  return g;
}

const BUILDERS: Partial<Record<MobKind, () => Group>> = {
  [MobKind.SLIME]: slime,
  [MobKind.KING_SLIME]: kingSlime,
  [MobKind.GOLEM]: golem,
};

interface Figure {
  group: Group; // placed in the world, squashed by hops
  body: Group; // modeled in meters, scaled to blocks
  hopMs: number; // time spent moving, for the slime's bounce
  last?: MobState;
}

/** Sheep, slimes and bosses, built from a few shared boxes. */
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
    const body = (BUILDERS[m.kind] ?? sheep)();
    body.scale.setScalar(METER * (MODEL_SCALE[m.kind] ?? 1));
    const f = { group: new Group().add(body), body, hopMs: 0 };
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
    if (m.kind === MobKind.SLIME || m.kind === MobKind.KING_SLIME) bounce(f.group, f.hopMs);
    if (m.kind === MobKind.GOLEM) f.group.rotation.z = 0.05 * Math.sin(f.hopMs / 250);
    const wool = f.body.children[0] as Mesh;
    if (m.kind === MobKind.SHEEP) wool.userData.paint = m.sheared ? paints.skin : paints.wool;
    f.group.traverse((c) => {
      if (c instanceof Mesh) c.material = m.hurt ? paints.hurt : (c.userData.paint as Material);
    });
  }
}

/** Slimes hop along, squashing as they land. */
function bounce(g: Group, hopMs: number): void {
  if (hopMs === 0) {
    g.scale.set(1, 1, 1);
    return;
  }
  const phase = Math.abs(Math.sin((hopMs / 400) * Math.PI));
  g.position.y += 0.3 * METER * phase;
  g.scale.set(1 + 0.1 * (1 - phase), 0.85 + 0.15 * phase, 1 + 0.1 * (1 - phase));
}
