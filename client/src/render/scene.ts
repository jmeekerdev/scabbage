import { Color, Fog, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import { FrameLimiter, pixelRatioFor, qualityFor, type Settings } from "../settings/settings";
import { CHUNK_SIZE } from "../world/constants";

const SKY = 0x87ceeb;

export interface GameScene {
  readonly scene: Scene;
  readonly camera: PerspectiveCamera;
  /** Applies new settings live; antialiasing only changes on reload. */
  apply(s: Settings): void;
  /** Runs fn before each drawn frame. */
  onFrame(fn: (timeMs: number) => void): void;
}

/** Creates the renderer, sky and camera, and draws whenever the frame limiter allows. */
export function startScene(canvas: HTMLCanvasElement, initial: Settings): GameScene {
  // WebGL fixes antialiasing when the context is created, so it comes from the startup preset.
  const renderer = new WebGLRenderer({ canvas, antialias: qualityFor(initial.preset).antialias });
  const scene = new Scene();
  scene.background = new Color(SKY);
  scene.fog = new Fog(SKY);
  const camera = new PerspectiveCamera(70, 1, 0.1);
  const limiter = new FrameLimiter(initial.fpsCap);
  const frameHooks: ((timeMs: number) => void)[] = [];

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
    for (const fn of frameHooks) fn(timeMs);
    renderer.render(scene, camera);
  });
  return { scene, camera, apply, onFrame: (fn) => frameHooks.push(fn) };
}

/** Fades terrain into the sky before the far plane so chunk edges never pop into view. */
function setViewDistance(fog: Fog, camera: PerspectiveCamera, blocks: number): void {
  fog.near = blocks * 0.6;
  fog.far = blocks;
  camera.far = blocks + CHUNK_SIZE;
  camera.updateProjectionMatrix();
}
