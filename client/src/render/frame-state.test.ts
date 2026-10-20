import { BoxGeometry, Group, Mesh, MeshBasicMaterial, PerspectiveCamera, Scene } from "three";
import { beforeEach, describe, expect, it } from "vitest";
import { FrameState } from "./frame-state";

describe("frame state", () => {
  let scene: Scene;
  let camera: PerspectiveCamera;
  let state: FrameState;
  let box: Mesh<BoxGeometry, MeshBasicMaterial>;

  /** Checks the scene and, like the render loop, draws whatever changed. */
  const draw = () => {
    const change = state.check(scene, camera);
    state.drew();
    return change;
  };

  beforeEach(() => {
    scene = new Scene();
    camera = new PerspectiveCamera();
    box = new Mesh(new BoxGeometry(), new MeshBasicMaterial());
    box.position.z = -5; // in front of the camera
    scene.add(new Group().add(box));
    state = new FrameState();
    draw();
  });

  it("sees nothing new in an untouched scene", () => {
    expect(draw()).toBe("none");
  });

  it("tells the camera moving, turning or zooming apart from objects moving", () => {
    camera.position.x = 1;
    expect(draw()).toBe("view");
    camera.rotation.y = 0.1;
    expect(draw()).toBe("view");
    camera.aspect = 2;
    camera.updateProjectionMatrix();
    expect(draw()).toBe("view");
    box.position.y = 1;
    expect(draw()).toBe("objects");
    expect(draw()).toBe("none");
  });

  it("notices objects moving, appearing, hiding and going", () => {
    box.position.y = 1;
    expect(draw()).toBe("objects");
    box.scale.x = 2;
    expect(draw()).toBe("objects");
    box.parent!.position.x = 1;
    expect(draw()).toBe("objects");
    scene.add(new Mesh(new BoxGeometry()).translateZ(-3));
    expect(draw()).toBe("objects");
    box.visible = false;
    expect(draw()).toBe("objects");
    box.visible = true;
    draw();
    box.removeFromParent();
    expect(draw()).toBe("objects");
  });

  it("notices material swaps, fades and new colors", () => {
    box.material = new MeshBasicMaterial();
    expect(draw()).toBe("objects");
    box.material.opacity = 0.5;
    expect(draw()).toBe("objects");
    box.material.color.setRGB(1, 0, 0);
    expect(draw()).toBe("objects");
  });

  it("ignores what happens inside hidden objects", () => {
    box.visible = false;
    draw();
    box.position.z = 3;
    expect(draw()).toBe("none");
  });

  it("ignores objects moving out of view, but not leaving or entering it", () => {
    box.position.z = 50; // behind the camera
    expect(draw()).toBe("objects");
    box.position.x = 3;
    expect(draw()).toBe("none");
    box.position.set(0, 0, -5);
    expect(draw()).toBe("objects");
  });

  it("ignores objects appearing or going out of view", () => {
    const behind = new Mesh(new BoxGeometry()).translateZ(50);
    scene.add(behind);
    expect(draw()).toBe("none");
    behind.removeFromParent();
    expect(draw()).toBe("none");
  });

  it("compares against the last drawn frame, not the last check", () => {
    box.position.y = 1;
    state.check(scene, camera);
    expect(state.check(scene, camera)).toBe("objects");
  });

  it("only notes which meshes of a fixed group are in view", () => {
    const chunks = new Group();
    chunks.userData.fixed = true;
    scene.add(chunks.add(new Mesh(new BoxGeometry()).translateZ(-3)));
    expect(draw()).toBe("objects");
    chunks.children[0]!.position.x = 1; // fixed groups promise this never happens
    expect(draw()).toBe("none");
    chunks.add(new Mesh(new BoxGeometry()).translateZ(50));
    expect(draw()).toBe("none");
    chunks.add(new Mesh(new BoxGeometry()).translateZ(-4));
    expect(draw()).toBe("objects");
  });

  it("names the object that changed", () => {
    box.position.y = 1;
    expect(state.check(scene, camera)).toBe("objects");
    expect(state.culprit(scene)).toBe(box);
  });

  it("can be told to draw anyway", () => {
    state.invalidate();
    expect(draw()).toBe("view");
    expect(draw()).toBe("none");
  });
});
