import { PerspectiveCamera, Scene, WebGLRenderer } from "three";
import { Avatar } from "./avatars";
import { daylight } from "./chunk-material";

const TURN_PER_MS = 0.0008;

/** A slowly turning figure wearing skin pixels that are being edited; its own small WebGL view. */
export class SkinPreview {
  readonly canvas = document.createElement("canvas");
  private readonly renderer: WebGLRenderer;
  private readonly scene = new Scene();
  private readonly camera = new PerspectiveCamera(30, 3 / 4, 0.1, 20);
  private readonly avatar: Avatar;
  private frame = 0;

  /** `pixels` is shared with the editor, so call refresh() after changing it. */
  constructor(pixels: Uint8Array, color: number) {
    this.canvas.width = 150;
    this.canvas.height = 200;
    this.canvas.className = "preview";
    this.renderer = new WebGLRenderer({ canvas: this.canvas, alpha: true });
    this.avatar = new Avatar(undefined, color, pixels);
    this.avatar.root.scale.setScalar(1); // shown in metres, the units the camera is placed in
    this.scene.add(this.avatar.root);
    this.camera.position.set(0, 1.1, 4.4);
    this.camera.lookAt(0, 0.9, 0);
    const spin = (ms: number) => {
      this.avatar.root.rotation.y = Math.PI + ms * TURN_PER_MS;
      this.render();
      this.frame = requestAnimationFrame(spin);
    };
    this.frame = requestAnimationFrame(spin);
  }

  refresh(): void {
    this.avatar.refreshSkin();
  }

  /** Draws at full daylight so the preview is clear even at night. */
  private render(): void {
    const sky = daylight.value;
    daylight.value = 1;
    this.renderer.render(this.scene, this.camera);
    daylight.value = sky;
  }

  dispose(): void {
    cancelAnimationFrame(this.frame);
    this.avatar.dispose();
    this.renderer.dispose();
    this.renderer.forceContextLoss();
  }
}
