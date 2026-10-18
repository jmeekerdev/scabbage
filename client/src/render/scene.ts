import {
  AmbientLight,
  BoxGeometry,
  Color,
  DirectionalLight,
  Fog,
  Mesh,
  MeshLambertMaterial,
  PerspectiveCamera,
  Scene,
  WebGLRenderer,
} from "three";
import { FrameLimiter, pixelRatioFor, qualityFor, type Settings } from "../settings/settings";
import { CHUNK_SIZE } from "../world/constants";

const SKY = 0x87ceeb;

export interface GameScene {
  /** Applies new settings live; antialiasing only changes on reload. */
  apply(s: Settings): void;
}

/** A placeholder scene: one spinning grass block, proving WebGL works on the device. */
export function startScene(canvas: HTMLCanvasElement, initial: Settings): GameScene {
  // WebGL fixes antialiasing when the context is created, so it comes from the startup preset.
  const renderer = new WebGLRenderer({ canvas, antialias: qualityFor(initial.preset).antialias });
  const scene = new Scene();
  scene.background = new Color(SKY);
  scene.fog = new Fog(SKY);
  scene.add(new AmbientLight(0xffffff, 0.6));
  const sun = new DirectionalLight(0xffffff, 1.2);
  sun.position.set(3, 5, 2);
  scene.add(sun);

  const dirt = new MeshLambertMaterial({ color: 0x8b5a2b });
  const grass = new MeshLambertMaterial({ color: 0x5fbf3f });
  // BoxGeometry face order: +x, -x, +y (top), -y, +z, -z.
  const block = new Mesh(new BoxGeometry(1, 1, 1), [dirt, dirt, grass, dirt, dirt, dirt]);
  scene.add(block);

  const camera = new PerspectiveCamera(60, 1, 0.1);
  camera.position.set(0, 1.2, 3);
  camera.lookAt(0, 0, 0);
  const limiter = new FrameLimiter(initial.fpsCap);

  const resize = () => {
    const { clientWidth: w, clientHeight: h } = canvas;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  };
  window.addEventListener("resize", resize);

  const apply = (s: Settings) => {
    renderer.setPixelRatio(pixelRatioFor(s.preset, window.devicePixelRatio));
    setViewDistance(scene.fog as Fog, camera, s.viewDistance * CHUNK_SIZE);
    limiter.fps = s.fpsCap;
    resize();
  };
  apply(initial);

  // setAnimationLoop uses requestAnimationFrame, which browsers pause in hidden tabs.
  renderer.setAnimationLoop((timeMs) => {
    if (!limiter.ready(timeMs)) return;
    block.rotation.y = timeMs / 1000;
    block.rotation.x = Math.sin(timeMs / 2000) * 0.3;
    renderer.render(scene, camera);
  });
  return { apply };
}

/** Fades terrain into the sky before the far plane so chunk edges never pop into view. */
function setViewDistance(fog: Fog, camera: PerspectiveCamera, blocks: number): void {
  fog.near = blocks * 0.6;
  fog.far = blocks;
  camera.far = blocks + CHUNK_SIZE;
  camera.updateProjectionMatrix();
}
