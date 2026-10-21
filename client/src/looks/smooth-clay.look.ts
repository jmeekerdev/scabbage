import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// Light numbers are the look book's sRGB ones raised to 2.2, to match in linear light.
export default {
  name: "Smooth Clay",
  about: "Rolling hills with crisp built blocks",
  surface: {
    textured: false,
    tint: 0.03,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "9ccc6b",
      dirt: "b98a64",
      stone: "b9b0a8",
      sand: "f0dcae",
      water: "5fb8cc",
      log: "8c6248",
      leaves: "7fbf5e",
      pine: "4f8f6a",
      planks: "e6c08f",
      brick: "e07a5f",
      glass: "cfeef2",
      redFlower: "f06b6b",
      yellowFlower: "ffd65c",
      gravel: "c9bfb2",
    }),
  },
  light: {
    faces: { top: 1, bottom: 0.75, x: 0.85, z: 0.85 },
    ao: { r: 0.24, g: 0.19, b: 0.55 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
    sun: {
      dir: { x: 0.45, y: 0.75, z: 0.45 },
      lit: { r: 1.18, g: 1, b: 0.75 },
      shade: { r: 0.4, g: 0.32, b: 0.75 },
      wrap: 0.35,
      bands: 0,
      shine: 0.06,
    },
  },
  shape: { edges: true, softCorners: 0.5 },
  sky: {
    day: hex("bcd3e6"),
    night: hex("1e2238"),
    sunset: hex("fbe0c4"),
    sun: hex("fff4d6"),
    moon: hex("e9ecf4"),
  },
  fog: { start: 0.5 },
  water: { opacity: 0.72 },
  after: {
    outline: { width: 1, strength: 0.55 },
    tiltShift: 1,
    vignette: 0.2,
    saturation: 1.04,
  },
} satisfies Look;
