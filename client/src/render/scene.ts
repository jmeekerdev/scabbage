import {
  AmbientLight,
  BoxGeometry,
  Color,
  DirectionalLight,
  Mesh,
  MeshLambertMaterial,
  PerspectiveCamera,
  Scene,
  WebGLRenderer,
} from "three";

/** A placeholder scene: one spinning grass block, proving WebGL works on the device. */
export function startScene(canvas: HTMLCanvasElement): void {
  const renderer = new WebGLRenderer({ canvas, antialias: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

  const scene = new Scene();
  scene.background = new Color(0x87ceeb);
  scene.add(new AmbientLight(0xffffff, 0.6));
  const sun = new DirectionalLight(0xffffff, 1.2);
  sun.position.set(3, 5, 2);
  scene.add(sun);

  const dirt = new MeshLambertMaterial({ color: 0x8b5a2b });
  const grass = new MeshLambertMaterial({ color: 0x5fbf3f });
  // BoxGeometry face order: +x, -x, +y (top), -y, +z, -z.
  const block = new Mesh(new BoxGeometry(1, 1, 1), [dirt, dirt, grass, dirt, dirt, dirt]);
  scene.add(block);

  const camera = new PerspectiveCamera(60, 1, 0.1, 100);
  camera.position.set(0, 1.2, 3);
  camera.lookAt(0, 0, 0);

  const resize = () => {
    const { clientWidth: w, clientHeight: h } = canvas;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  };
  window.addEventListener("resize", resize);
  resize();

  // setAnimationLoop uses requestAnimationFrame, which browsers pause in hidden tabs.
  renderer.setAnimationLoop((timeMs) => {
    block.rotation.y = timeMs / 1000;
    block.rotation.x = Math.sin(timeMs / 2000) * 0.3;
    renderer.render(scene, camera);
  });
}
