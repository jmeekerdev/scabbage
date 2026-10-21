import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// Light numbers are the look book's sRGB ones raised to 2.2, to match in linear light.
export default {
  name: "Clay Diorama",
  about: "Soft moulded colours with rounded edges",
  surface: {
    textured: false,
    tint: 0.03,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "9cc96b",
      dirt: "b98a64",
      stone: "b7b2a8",
      sand: "f0dcae",
      water: "6fc3d6",
      log: "8c6248",
      leaves: "7fbf5e",
      pine: "5f9e6e",
      planks: "e6c08f",
      brick: "e07a5f",
      glass: "cfeef2",
      redFlower: "f06b6b",
      yellowFlower: "ffd65c",
      gravel: "c9bfb2",
    }),
  },
  light: {
    faces: { top: 1, bottom: 0.55, x: 1, z: 1 },
    ao: { r: 0.2, g: 0.2, b: 0.2 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
    sun: {
      dir: { x: 0.45, y: 0.8, z: 0.4 },
      lit: { r: 1.14, g: 1, b: 0.98 },
      shade: { r: 0.22, g: 0.2, b: 0.24 },
      wrap: 0.25,
      bands: 0,
      shine: 0.12,
    },
  },
  shape: { edges: true, softCorners: 0 },
  sky: {
    day: hex("bcdbe8"),
    night: hex("1c2433"),
    sunset: hex("f4b38a"),
    sun: hex("fff4d6"),
    moon: hex("e9ecf4"),
  },
  fog: { start: 0.5 },
  water: { opacity: 0.8 },
  after: { tiltShift: 1, vignette: 0.25, saturation: 1.05 },
} satisfies Look;
