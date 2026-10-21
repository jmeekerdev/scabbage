import type { Look } from "./look";

/** The pixel-textured blocks the game started with. */
export default {
  name: "Today",
  about: "Crisp pixel blocks",
  surface: { textured: true, tint: 0.06 },
  light: {
    faces: { top: 1, bottom: 0.5, x: 0.8, z: 0.65 },
    ao: { r: 0.5, g: 0.5, b: 0.5 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
  },
  shape: { edges: false, softCorners: 0 },
  sky: {
    day: { r: 0.53, g: 0.81, b: 0.92 },
    night: { r: 0.04, g: 0.06, b: 0.14 },
    sunset: { r: 0.98, g: 0.6, b: 0.35 },
    sun: { r: 1, g: 0.949, b: 0.627 },
    moon: { r: 0.91, g: 0.925, b: 0.957 },
  },
  fog: { start: 0.6 },
  water: { opacity: 1 },
} satisfies Look;
