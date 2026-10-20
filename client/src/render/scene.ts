import { Fog, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import { FrameLimiter, pixelRatioFor, qualityFor, type Settings } from "../settings/settings";
import { CHUNK_SIZE } from "../world/constants";
import { daylight } from "./chunk-material";
import { FrameState } from "./frame-state";
import { skyAt } from "./sky";
import { SkyView } from "./sky-view";

export interface GameScene {
  readonly scene: Scene;
  readonly camera: PerspectiveCamera;
  /** Applies new settings live; antialiasing only changes on reload. */
  apply(s: Settings): void;
  /** How lit the world is right now, from MIN_LIGHT at night to 1 by day. */
  daylight(): number;
  /** Runs fn on each frame the limiter allows, before deciding whether to draw it. */
  onFrame(fn: (timeMs: number) => void): void;
  /** Runs fn after each drawn frame; steady is false when the frame before it wasn't drawn. */
  onDrawn(fn: (timeMs: number, steady: boolean) => void): void;
  /** Draws the next frame even if nothing visible seems to have changed, e.g. after new textures. */
  invalidate(): void;
}

// The sky moves so slowly that stepping it once a second is unnoticeable, and lets idle frames go undrawn.
const SKY_STEP_MS = 1000;

/** Creates the renderer, sky and camera, and draws whenever something changed and the limiter allows. */
export function startScene(canvas: HTMLCanvasElement, initial: Settings): GameScene {
  // WebGL fixes antialiasing when the context is created, so it comes from the startup preset.
  const renderer = new WebGLRenderer({ canvas, antialias: qualityFor(initial.preset).antialias });
  const scene = new Scene();
  scene.fog = new Fog(0);
  const sky = new SkyView(scene);
  const camera = new PerspectiveCamera(70, 1, 0.1);
  const limiter = new FrameLimiter(initial.fpsCap);
  const frameHooks: ((timeMs: number) => void)[] = [];
  const drawnHooks: ((timeMs: number, steady: boolean) => void)[] = [];
  const frame = new FrameState();
  let steady = false;

  const resize = () => {
    const { clientWidth: w, clientHeight: h } = canvas;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    frame.invalidate(); // resizing clears the canvas
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
    const now = skyAt(Math.floor(Date.now() / SKY_STEP_MS) * SKY_STEP_MS);
    if (now.daylight !== daylight.value) frame.invalidate();
    daylight.value = now.daylight;
    sky.update(now, camera, camera.far * 0.8);
    if (!frame.changed(scene, camera)) {
      steady = false;
      return;
    }
    renderer.render(scene, camera);
    for (const fn of drawnHooks) fn(timeMs, steady);
    steady = true;
  });
  return {
    scene,
    camera,
    apply,
    daylight: () => daylight.value,
    onFrame: (fn) => frameHooks.push(fn),
    onDrawn: (fn) => drawnHooks.push(fn),
    invalidate: () => frame.invalidate(),
  };
}

/** Fades terrain into the sky before the far plane so chunk edges never pop into view. */
function setViewDistance(fog: Fog, camera: PerspectiveCamera, blocks: number): void {
  fog.near = blocks * 0.6;
  fog.far = blocks;
  camera.far = blocks + CHUNK_SIZE;
  camera.updateProjectionMatrix();
}
