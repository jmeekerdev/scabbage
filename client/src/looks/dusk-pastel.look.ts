import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// The look book shades in sRGB; its light multipliers are raised to 2.2 here to match in linear light.
export default {
  name: "Dusk Pastel",
  about: "Flat pastel faces, warm sun and lilac shade",
  surface: {
    textured: false,
    detail: 0.5,
    tint: 0.03,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "b7d98a",
      dirt: "e0a98b",
      stone: "c9bcd6",
      sand: "ffe1b8",
      water: "8fd3e0",
      log: "b27f8a",
      leaves: "9fd6a0",
      pine: "7fbfae",
      planks: "ffd2a1",
      brick: "ff8f86",
      glass: "e8f7ff",
      redFlower: "ff6f91",
      yellowFlower: "ffe08a",
      gravel: "e2cfd8",
    }),
  },
  light: {
    faces: { top: 1.09, bottom: 1, x: 1, z: 1 },
    ao: { r: 0.49, g: 0.49, b: 0.49 },
    lamp: { r: 1, g: 0.82, b: 0.7 },
    lampFade: 0.82,
    sun: {
      dir: { x: -0.8, y: 0.35, z: 0.45 },
      lit: { r: 1.23, g: 1, b: 0.79 },
      shade: { r: 0.35, g: 0.28, b: 0.72 },
      wrap: 0.1,
      bands: 0,
      shine: 0,
    },
    height: { low: 30, high: 80, lift: 0.1 }, // from the sea up to the high hills
  },
  shape: { edges: false, softCorners: 0 },
  sky: {
    day: hex("ffcaa8"),
    night: hex("2a2850"),
    sunset: hex("ff9f9a"),
    sun: hex("fff2c8"),
    moon: hex("e8e4f6"),
  },
  fog: { start: 0.3 },
  water: { opacity: 0.8 },
  after: { vignette: 0.15 },
} satisfies Look;
