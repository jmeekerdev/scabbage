import {
  BoxGeometry,
  EdgesGeometry,
  LineBasicMaterial,
  LineSegments,
  Mesh,
  MeshBasicMaterial,
  type Scene,
} from "three";

/** A thin outline around the brush under the crosshair, darkening as it is dug. */
export class Selection {
  private readonly box = new LineSegments(
    new EdgesGeometry(new BoxGeometry(1.004, 1.004, 1.004)),
    new LineBasicMaterial({ color: 0x000000, transparent: true, opacity: 0.6 }),
  );
  private readonly shade = new MeshBasicMaterial({
    color: 0x000000,
    transparent: true,
    depthWrite: false,
  });

  constructor(scene: Scene) {
    this.box.name = "selection";
    this.box.visible = false;
    this.box.add(new Mesh(new BoxGeometry(1.002, 1.002, 1.002), this.shade));
    scene.add(this.box);
  }

  /** `dug` runs from 0 to 1 while a survival player mines the cube. */
  show(cube: { x: number; y: number; z: number; size?: number } | undefined, dug = 0): void {
    this.box.visible = cube !== undefined;
    if (cube) {
      const half = (cube.size ?? 1) / 2;
      this.box.position.set(cube.x + half, cube.y + half, cube.z + half);
      this.box.scale.setScalar(cube.size ?? 1);
    }
    this.shade.opacity = dug * 0.6;
    this.shade.visible = dug > 0;
  }
}
