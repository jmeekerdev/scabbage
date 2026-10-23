import {
  BufferGeometry,
  DoubleSide,
  Float32BufferAttribute,
  Mesh,
  MeshBasicMaterial,
  type Scene,
} from "three";
import type { Cell } from "../player/brush";
import type { Preview } from "../autopilot/preview";

const ADD = [0.35, 0.85, 1]; // light blue for blocks a job will build
const REMOVE = [1, 0.3, 0.25]; // red for blocks it will break
const PUFF = 0.02; // faces sit a hair outside the cell, clear of the real block's faces

// Each face: the neighbor it's hidden by, and its four corners as 0/1 offsets.
const FACES: [number[], number[][]][] = [
  [
    [1, 0, 0],
    [
      [1, 0, 0],
      [1, 1, 0],
      [1, 1, 1],
      [1, 0, 1],
    ],
  ],
  [
    [-1, 0, 0],
    [
      [0, 0, 0],
      [0, 0, 1],
      [0, 1, 1],
      [0, 1, 0],
    ],
  ],
  [
    [0, 1, 0],
    [
      [0, 1, 0],
      [0, 1, 1],
      [1, 1, 1],
      [1, 1, 0],
    ],
  ],
  [
    [0, -1, 0],
    [
      [0, 0, 0],
      [1, 0, 0],
      [1, 0, 1],
      [0, 0, 1],
    ],
  ],
  [
    [0, 0, 1],
    [
      [0, 0, 1],
      [1, 0, 1],
      [1, 1, 1],
      [0, 1, 1],
    ],
  ],
  [
    [0, 0, -1],
    [
      [0, 0, 0],
      [0, 1, 0],
      [1, 1, 0],
      [1, 0, 0],
    ],
  ],
];

/** One mesh of the outside faces of a preview's cells, colored by whether they're built or broken. */
export function ghostGeometry(p: Preview): BufferGeometry {
  const pos: number[] = [];
  const color: number[] = [];
  const index: number[] = [];
  for (const [cells, rgb] of [
    [p.add, ADD],
    [p.remove, REMOVE],
  ] as const) {
    const has = new Set(cells.map((c) => `${c.x},${c.y},${c.z}`));
    for (const c of cells) addFaces(c, has, rgb, pos, color, index);
  }
  return new BufferGeometry()
    .setAttribute("position", new Float32BufferAttribute(pos, 3))
    .setAttribute("color", new Float32BufferAttribute(color, 3))
    .setIndex(index);
}

function addFaces(
  c: Cell,
  has: ReadonlySet<string>,
  rgb: readonly number[],
  pos: number[],
  color: number[],
  index: number[],
): void {
  for (const [[nx, ny, nz], corners] of FACES) {
    if (has.has(`${c.x + nx!},${c.y + ny!},${c.z + nz!}`)) continue;
    const first = pos.length / 3;
    for (const [i, j, k] of corners) {
      pos.push(c.x + puff(i!), c.y + puff(j!), c.z + puff(k!));
      color.push(...rgb);
    }
    index.push(first, first + 1, first + 2, first, first + 2, first + 3);
  }
}

const puff = (corner: number) => (corner ? 1 + PUFF : -PUFF);

/** A see-through picture of what a job waiting for /yes would change. */
export class Ghost {
  private readonly material = new MeshBasicMaterial({
    vertexColors: true,
    transparent: true,
    opacity: 0.35,
    depthWrite: false,
    side: DoubleSide,
  });
  private mesh?: Mesh;

  constructor(private readonly scene: Scene) {}

  show(p: Preview | undefined): void {
    if (this.mesh) {
      this.scene.remove(this.mesh);
      this.mesh.geometry.dispose();
      this.mesh = undefined;
    }
    if (!p) return;
    this.mesh = new Mesh(ghostGeometry(p), this.material);
    this.mesh.name = "ghost";
    this.scene.add(this.mesh);
  }
}
