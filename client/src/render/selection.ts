import {
  BoxGeometry,
  BufferGeometry,
  Float32BufferAttribute,
  LineBasicMaterial,
  LineSegments,
  Mesh,
  MeshBasicMaterial,
  type Scene,
  SRGBColorSpace,
} from "three";
import type { Look } from "../looks/look";

/** How far world corner x,y,z of a block has moved, in blocks. */
export type Bend = (x: number, y: number, z: number) => number[];

const LINE_HALF = 0.502; // a hair outside the block so the lines aren't hidden in its faces
const SHADE_HALF = 0.501;

// The cube's 8 corners as 0/1 per axis, and its 12 edges as pairs of them.
const CORNERS = Array.from({ length: 8 }, (_, i) => [i & 1, (i >> 1) & 1, i >> 2]);
const EDGES = CORNERS.flatMap((_, a) =>
  [1, 2, 4].filter((bit) => !(a & bit)).map((bit) => [a, a | bit]),
);

/** Which corner each vertex of a unit box sits at. */
function cornerOf(g: BufferGeometry): number[] {
  const p = g.attributes.position;
  return Array.from(
    { length: p.count },
    (_, i) => (p.getX(i) > 0 ? 1 : 0) | (p.getY(i) > 0 ? 2 : 0) | (p.getZ(i) > 0 ? 4 : 0),
  );
}

/** A thin outline around the brush under the crosshair, bent to soft corners, darkening as it is dug. */
export class Selection {
  private readonly box = new LineSegments(
    new BufferGeometry().setAttribute(
      "position",
      new Float32BufferAttribute(new Float32Array(EDGES.length * 2 * 3), 3),
    ),
    new LineBasicMaterial({ color: 0x000000, transparent: true, opacity: 0.6 }),
  );
  private readonly shade = new MeshBasicMaterial({
    color: 0x000000,
    transparent: true,
    depthWrite: false,
  });
  private readonly shadeBox = new BoxGeometry(1, 1, 1);
  private readonly shadeCorners = cornerOf(this.shadeBox);
  private shape = "";

  /** bend says where soft corners moved each block corner, so the outline hugs the drawn shape. */
  constructor(
    scene: Scene,
    private readonly bend: Bend = () => [0, 0, 0],
  ) {
    this.box.name = "selection";
    this.box.visible = false;
    this.box.add(new Mesh(this.shadeBox, this.shade));
    scene.add(this.box);
  }

  /** Draws lines in the look's outline colour, so they match the rest of the picture. */
  setLook(look: Look): void {
    const c = look.after?.outline?.color ?? { r: 0, g: 0, b: 0 };
    this.box.material.color.setRGB(c.r, c.g, c.b, SRGBColorSpace);
  }

  /** `dug` runs from 0 to 1 while a survival player mines the cube. */
  show(cube: { x: number; y: number; z: number; size?: number } | undefined, dug = 0): void {
    this.box.visible = cube !== undefined;
    if (cube) {
      const size = cube.size ?? 1;
      const half = size / 2;
      this.box.position.set(cube.x + half, cube.y + half, cube.z + half);
      this.box.scale.setScalar(size);
      this.reshape(
        CORNERS.map(([i, j, k]) =>
          this.bend(cube.x + i * size, cube.y + j * size, cube.z + k * size).map((d) => d / size),
        ),
      );
    }
    this.shade.opacity = dug * 0.6;
    this.shade.visible = dug > 0;
  }

  /** Moves the outline's and shade's corners, re-uploading only when they changed. */
  private reshape(moves: number[][]): void {
    const key = moves.join();
    if (key === this.shape) return;
    this.shape = key;
    const at = (c: number, half: number) =>
      CORNERS[c].map((s, a) => (s - 0.5) * 2 * half + moves[c][a]);
    const lines = this.box.geometry.attributes.position as Float32BufferAttribute;
    EDGES.flat().forEach((c, i) =>
      lines.setXYZ(i, ...(at(c, LINE_HALF) as [number, number, number])),
    );
    lines.needsUpdate = true;
    this.box.geometry.computeBoundingSphere();
    const faces = this.shadeBox.attributes.position as Float32BufferAttribute;
    this.shadeCorners.forEach((c, i) =>
      faces.setXYZ(i, ...(at(c, SHADE_HALF) as [number, number, number])),
    );
    faces.needsUpdate = true;
  }
}
