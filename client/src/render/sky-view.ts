import {
  Color,
  Mesh,
  MeshBasicMaterial,
  PlaneGeometry,
  SRGBColorSpace,
  type Camera,
  type Fog,
  type Scene,
} from "three";
import type { SkyState } from "./sky";

const TILT = 0.3; // the sun's path leans south so it is not straight overhead

/** Paints the sky and fog color and moves a blocky sun and moon around the player. */
export class SkyView {
  private readonly background = new Color();
  private readonly sun = disc(0xfff2a0, 0.09);
  private readonly moon = disc(0xe8ecf4, 0.06);

  constructor(private readonly scene: Scene) {
    scene.background = this.background;
    scene.add(this.sun, this.moon);
  }

  update(sky: SkyState, camera: Camera, distance: number): void {
    const { r, g, b } = sky.color;
    this.background.setRGB(r, g, b, SRGBColorSpace);
    (this.scene.fog as Fog).color.copy(this.background);
    place(this.sun, camera, sky.sun.x, sky.sun.y, distance);
    place(this.moon, camera, -sky.sun.x, -sky.sun.y, distance);
  }
}

/** A flat square sized as a fraction of its distance; drawn first so terrain always covers it. */
function disc(color: number, size: number): Mesh {
  const mesh = new Mesh(
    new PlaneGeometry(size, size),
    new MeshBasicMaterial({ color, fog: false, depthWrite: false }),
  );
  mesh.renderOrder = -1;
  return mesh;
}

function place(mesh: Mesh, camera: Camera, x: number, y: number, distance: number): void {
  mesh.visible = y > -0.1;
  mesh.scale.setScalar(distance);
  mesh.position.set(x, y, TILT).normalize().multiplyScalar(distance).add(camera.position);
  mesh.lookAt(camera.position);
}
