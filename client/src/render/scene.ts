import { Fog, PerspectiveCamera, Scene, WebGLRenderer, type Object3D } from "three";
import { FrameLimiter, pixelRatioFor, qualityFor, type Settings } from "../settings/settings";
import { lookById } from "../looks/registry";
import type { Look } from "../looks/look";
import { CHUNK_SIZE } from "../world/constants";
import { AfterPass } from "./after-pass";
import { daylight } from "./chunk-material";
import { farDistance } from "./far-land";
import { FrameState, type Change } from "./frame-state";
import { Pacer } from "./pacer";
import { skyAt } from "./sky";
import { SkyView } from "./sky-view";

export interface GameScene {
  readonly scene: Scene;
  readonly camera: PerspectiveCamera;
  /** Applies new settings live, including the look's sky and fog; antialiasing only changes on reload. */
  apply(s: Settings): void;
  /** Shows a look's sky, fog and afterwards pass without changing the settings, e.g. while tuning one. */
  showLook(look: Look): void;
  /** How lit the world is right now, from MIN_LIGHT at night to 1 by day. */
  daylight(): number;
  /** Runs fn on each frame the limiter allows, before deciding whether to draw it. */
  onFrame(fn: (timeMs: number) => void): void;
  /** Runs fn after each drawn frame; steady is false when the frame before it wasn't drawn. */
  onDrawn(fn: (timeMs: number, steady: boolean, change: Change) => void): void;
  /** Draws the next frame even if nothing visible seems to have changed, e.g. after new textures. */
  invalidate(): void;
}

// The sky moves so slowly that stepping it once a second is unnoticeable, and lets idle frames go undrawn.
const SKY_STEP_MS = 1000;
// Things moving on their own, like mobs and other players, look smooth at this rate; the camera gets more.
const OBJECTS_FPS = 30;
// Input that should bring an idle loop straight back to full speed.
const WAKING_EVENTS = ["keydown", "pointerdown", "pointermove", "mousemove", "wheel", "touchstart"];

/** Creates the renderer, sky and camera, and draws whenever something changed and the limiter allows. */
export function startScene(canvas: HTMLCanvasElement, initial: Settings): GameScene {
  // WebGL fixes antialiasing when the context is created, so it comes from the startup preset.
  const { antialias } = qualityFor(initial.preset);
  const renderer = new WebGLRenderer({ canvas, antialias });
  const after = new AfterPass(antialias ? 4 : 0);
  const scene = new Scene();
  scene.fog = new Fog(0);
  const sky = new SkyView(scene);
  const camera = new PerspectiveCamera(70, 1, 0.1);
  const limiter = new FrameLimiter(initial.fpsCap);
  const frameHooks: ((timeMs: number) => void)[] = [];
  const drawnHooks: ((timeMs: number, steady: boolean, change: Change) => void)[] = [];
  const frame = new FrameState();
  let steady = false;
  let lastDrawnMs = -Infinity;

  const resize = () => {
    const { clientWidth: w, clientHeight: h } = canvas;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    frame.invalidate(); // resizing clears the canvas
  };
  // iOS fires window resize before layout settles on rotation; the observer sees the final size.
  new ResizeObserver(resize).observe(canvas);

  let look = lookById(initial.look);
  let viewDistance = initial.viewDistance;
  const showLook = (next: Look) => {
    look = next;
    sky.setLook(look.sky);
    after.setLook(look, false); // no overhead map view yet
    setViewDistance(scene.fog as Fog, camera, viewDistance, look);
    frame.invalidate();
  };
  const apply = (s: Settings) => {
    viewDistance = s.viewDistance;
    showLook(lookById(s.look));
    renderer.setPixelRatio(pixelRatioFor(s.preset, window.devicePixelRatio));
    limiter.fps = s.fpsCap;
    resize();
  };
  apply(initial);

  /** Runs one frame's hooks and draws it if anything changed, reporting what did. */
  const step = (timeMs: number): Change => {
    for (const fn of frameHooks) fn(timeMs);
    const now = skyAt(Math.floor(Date.now() / SKY_STEP_MS) * SKY_STEP_MS, look.sky);
    if (now.daylight !== daylight.value) frame.invalidate();
    daylight.value = now.daylight;
    sky.update(now, camera, camera.far * 0.8);
    const change = frame.check(scene, camera);
    const soon = change === "objects" && timeMs - lastDrawnMs < 1000 / OBJECTS_FPS;
    if (change === "none" || soon) {
      steady = false;
      return change;
    }
    if (DEBUG_FRAMES) logCulprit(change, frame.culprit(scene));
    if (after.active) after.render(renderer, scene, camera);
    else renderer.render(scene, camera);
    frame.drew();
    lastDrawnMs = timeMs;
    for (const fn of drawnHooks) fn(timeMs, steady, change);
    steady = true;
    return change;
  };

  // While the view is still, a timer replaces requestAnimationFrame so the page mostly sleeps.
  const pacer = new Pacer();
  let sleeping = false;
  let timer: number | undefined;
  const paced = (timeMs: number) => {
    pacer.frame(timeMs, step(timeMs));
    const ms = pacer.step();
    if (ms !== undefined) sleep(ms);
  };
  const sleep = (ms: number) => {
    if (!sleeping) renderer.setAnimationLoop(null);
    sleeping = true;
    // Hidden tabs wait for visibilitychange instead.
    timer = document.hidden ? undefined : window.setTimeout(idleTick, ms);
  };
  const idleTick = () => {
    paced(performance.now());
    if (pacer.step() === undefined) wake();
  };
  const wake = () => {
    pacer.wake(performance.now());
    if (!sleeping) return;
    sleeping = false;
    clearTimeout(timer);
    // setAnimationLoop uses requestAnimationFrame, which browsers pause in hidden tabs.
    renderer.setAnimationLoop((timeMs) => limiter.ready(timeMs) && paced(timeMs));
  };
  for (const type of WAKING_EVENTS) addEventListener(type, wake, { capture: true, passive: true });
  document.addEventListener("visibilitychange", () => !document.hidden && wake());
  sleeping = true;
  wake();
  return {
    scene,
    camera,
    apply,
    showLook,
    daylight: () => daylight.value,
    onFrame: (fn) => frameHooks.push(fn),
    onDrawn: (fn) => drawnHooks.push(fn),
    invalidate: () => frame.invalidate(),
  };
}

// With ?frames in the address, a counter and the console say what made each frame get drawn.
export const DEBUG_FRAMES = new URLSearchParams(location.search).has("frames");

function logCulprit(change: Change, o: Object3D | undefined): void {
  const path: string[] = [];
  for (let at: Object3D | null | undefined = o; at; at = at.parent)
    path.unshift(at.name || at.type);
  console.debug(change, path.join(" > "), o?.position.toArray());
}

/** Starts fading where the look says, and fades far land into the sky before the far plane. */
function setViewDistance(fog: Fog, camera: PerspectiveCamera, chunks: number, look: Look): void {
  const far = farDistance(chunks) * CHUNK_SIZE;
  fog.near = chunks * CHUNK_SIZE * look.fog.start;
  fog.far = far;
  camera.far = far + CHUNK_SIZE;
  camera.updateProjectionMatrix();
}
