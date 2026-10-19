import { Fog, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import { FrameLimiter, pixelRatioFor, qualityFor, type Settings } from "../settings/settings";
import { CHUNK_SIZE } from "../world/constants";
import { daylight } from "./chunk-material";
import { skyAt } from "./sky";
import { SkyView } from "./sky-view";

export interface GameScene {
  readonly scene: Scene;
  readonly camera: PerspectiveCamera;
  /** Applies new settings live; antialiasing only changes on reload. */
  apply(s: Settings): void;
  /** How lit the world is right now, from MIN_LIGHT at night to 1 by day. */
  daylight(): number;
  /** Runs fn before each drawn frame. */
  onFrame(fn: (timeMs: number) => void): void;
}

/** Creates the renderer, sky and camera, and draws whenever the frame limiter allows. */
export function startScene(canvas: HTMLCanvasElement, initial: Settings): GameScene {
  // WebGL fixes antialiasing when the context is created, so it comes from the startup preset.
  const renderer = new WebGLRenderer({ canvas, antialias: qualityFor(initial.preset).antialias });
  const scene = new Scene();
  scene.fog = new Fog(0);
  const sky = new SkyView(scene);
  const camera = new PerspectiveCamera(70, 1, 0.1);
  const limiter = new FrameLimiter(initial.fpsCap);
  const frameHooks: ((timeMs: number) => void)[] = [];

  const resize = () => {
    const { clientWidth: w, clientHeight: h } = canvas;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  };
  // iOS fires window resize before layout settles on rotation; the observer sees the final size.
  new ResizeObserver(resize).observe(canvas);

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
    const now = skyAt(Date.now());
    daylight.value = now.daylight;
    sky.update(now, camera, camera.far * 0.8);
    renderer.render(scene, camera);
  });
  return {
    scene,
    camera,
    apply,
    daylight: () => daylight.value,
    onFrame: (fn) => frameHooks.push(fn),
  };
}

/** Fades terrain into the sky before the far plane so chunk edges never pop into view. */
function setViewDistance(fog: Fog, camera: PerspectiveCamera, blocks: number): void {
  fog.near = blocks * 0.6;
  fog.far = blocks;
  camera.far = blocks + CHUNK_SIZE;
  camera.updateProjectionMatrix();
}
