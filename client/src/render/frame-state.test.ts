import { Group, Mesh, MeshBasicMaterial, PerspectiveCamera, Scene } from "three";
import { beforeEach, describe, expect, it } from "vitest";
import { FrameState } from "./frame-state";

describe("frame state", () => {
  let scene: Scene;
  let camera: PerspectiveCamera;
  let state: FrameState;
  let box: Mesh<never, MeshBasicMaterial>;

  beforeEach(() => {
    scene = new Scene();
    camera = new PerspectiveCamera();
    box = new Mesh(undefined, new MeshBasicMaterial()) as Mesh<never, MeshBasicMaterial>;
    scene.add(new Group().add(box));
    state = new FrameState();
    state.changed(scene, camera);
  });

  it("sees nothing new in an untouched scene", () => {
    expect(state.changed(scene, camera)).toBe(false);
  });

  it("notices the camera moving, turning or zooming", () => {
    camera.position.x = 1;
    expect(state.changed(scene, camera)).toBe(true);
    camera.rotation.y = 1;
    expect(state.changed(scene, camera)).toBe(true);
    camera.aspect = 2;
    camera.updateProjectionMatrix();
    expect(state.changed(scene, camera)).toBe(true);
    expect(state.changed(scene, camera)).toBe(false);
  });

  it("notices objects moving, appearing, hiding and going", () => {
    box.position.y = 1;
    expect(state.changed(scene, camera)).toBe(true);
    box.scale.x = 2;
    expect(state.changed(scene, camera)).toBe(true);
    scene.add(new Mesh());
    expect(state.changed(scene, camera)).toBe(true);
    box.visible = false;
    expect(state.changed(scene, camera)).toBe(true);
    box.removeFromParent();
    expect(state.changed(scene, camera)).toBe(true);
  });

  it("notices material swaps, fades and new textures", () => {
    box.material = new MeshBasicMaterial();
    expect(state.changed(scene, camera)).toBe(true);
    box.material.opacity = 0.5;
    expect(state.changed(scene, camera)).toBe(true);
    box.material.color.setRGB(1, 0, 0);
    expect(state.changed(scene, camera)).toBe(true);
  });

  it("ignores what happens inside hidden objects", () => {
    box.visible = false;
    state.changed(scene, camera);
    box.position.z = 3;
    expect(state.changed(scene, camera)).toBe(false);
  });

  it("can be told to draw anyway", () => {
    state.invalidate();
    expect(state.changed(scene, camera)).toBe(true);
    expect(state.changed(scene, camera)).toBe(false);
  });
});
