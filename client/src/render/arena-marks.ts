import { BoxGeometry, Group, Mesh, MeshBasicMaterial, type Scene } from "three";

/** An arena's spawn or goal block, as the server sent it. */
export interface MarkInfo {
  x: number;
  y: number;
  z: number;
  team: string; // empty for everyone
  goal: boolean;
}

export const TEAM_COLORS: Record<string, number> = {
  "": 0xffffff,
  red: 0xe53935,
  blue: 0x1e88e5,
  green: 0x43a047,
  yellow: 0xfdd835,
};

const pad = new BoxGeometry(0.8, 0.06, 0.8).translate(0, 0.03, 0);
const pole = new BoxGeometry(0.08, 1.6, 0.08).translate(0, 0.8, 0);
const banner = new BoxGeometry(0.5, 0.35, 0.04).translate(0.29, 1.4, 0);
const poleMaterial = new MeshBasicMaterial({ color: 0xdddddd });
const materials = new Map<number, MeshBasicMaterial>();

// Unlit, so marks stay easy to spot at night.
function teamMaterial(team: string): MeshBasicMaterial {
  const color = TEAM_COLORS[team] ?? TEAM_COLORS[""];
  let m = materials.get(color);
  if (!m) materials.set(color, (m = new MeshBasicMaterial({ color })));
  return m;
}

/** Colored pads on arena spawns and flags on their goals. */
export class ArenaMarks {
  readonly group = new Group();

  constructor(scene: Scene) {
    scene.add(this.group);
  }

  set(zones: readonly { marks: readonly MarkInfo[] }[]): void {
    this.group.clear();
    for (const zone of zones) for (const m of zone.marks) this.group.add(markMesh(m));
  }
}

function markMesh(m: MarkInfo): Group {
  const g = new Group();
  g.position.set(m.x + 0.5, m.y, m.z + 0.5);
  if (m.goal) g.add(new Mesh(pole, poleMaterial), new Mesh(banner, teamMaterial(m.team)));
  else g.add(new Mesh(pad, teamMaterial(m.team)));
  return g;
}
