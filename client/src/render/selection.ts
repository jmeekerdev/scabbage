import { BoxGeometry, EdgesGeometry, LineBasicMaterial, LineSegments, type Scene } from "three";

/** A thin outline around the block under the crosshair. */
export class Selection {
  private readonly box = new LineSegments(
    new EdgesGeometry(new BoxGeometry(1.004, 1.004, 1.004)),
    new LineBasicMaterial({ color: 0x000000, transparent: true, opacity: 0.6 }),
  );

  constructor(scene: Scene) {
    this.box.visible = false;
    scene.add(this.box);
  }

  show(block: { x: number; y: number; z: number } | undefined): void {
    this.box.visible = block !== undefined;
    if (block) this.box.position.set(block.x + 0.5, block.y + 0.5, block.z + 0.5);
  }
}
