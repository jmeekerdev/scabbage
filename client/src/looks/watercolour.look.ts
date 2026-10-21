import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// Light numbers are the look book's sRGB ones raised to 2.2, to match in linear light.
export default {
  name: "Watercolour",
  about: "Drifting washes on paper with loose pencil lines",
  surface: {
    textured: false,
    tint: 0.02,
    wash: 1,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "a7c58a",
      dirt: "c4a07e",
      stone: "b9b6ad",
      sand: "ecdcb4",
      water: "8fbfd0",
      log: "98785c",
      leaves: "8fb57a",
      pine: "76a08a",
      planks: "e0c49c",
      brick: "cf8a74",
      glass: "d8ecef",
      redFlower: "e48a8a",
      yellowFlower: "ecd27a",
      gravel: "cbc2b4",
    }),
  },
  light: {
    faces: { top: 1, bottom: 1, x: 1, z: 1 },
    ao: { r: 0.49, g: 0.49, b: 0.49 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
    sun: {
      dir: { x: 0.4, y: 0.8, z: 0.45 },
      lit: { r: 1.09, g: 1.09, b: 1.09 },
      shade: { r: 0.52, g: 0.52, b: 0.52 },
      wrap: 0,
      bands: 0,
      shine: 0,
    },
  },
  shape: { edges: true, softCorners: 0, pool: 0.2 },
  sky: {
    day: hex("e6eae4"),
    night: hex("2a2e38"),
    sunset: hex("f0cdb4"),
    sun: hex("fff6dc"),
    moon: hex("eef0f2"),
  },
  fog: { start: 0.4 },
  water: { opacity: 0.8 },
  after: {
    outline: { width: 1, strength: 0.55, color: hex("5a4d44"), creases: true },
    paper: 1,
    saturation: 0.9,
  },
} satisfies Look;
