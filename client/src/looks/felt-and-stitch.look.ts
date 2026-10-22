import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// Light numbers are the look book's sRGB ones raised to 2.2, to match in linear light.
export default {
  name: "Felt & Stitch",
  about: "Fuzzy fabric with stitched edges",
  surface: {
    textured: false,
    tint: 0.03,
    fibre: 1,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "6fae5a",
      dirt: "9a6a4a",
      stone: "9d9a94",
      sand: "e8cf9a",
      water: "5aa4c9",
      log: "7b5236",
      leaves: "5e9e4c",
      pine: "3f7f62",
      planks: "d39b5e",
      brick: "c4473e",
      glass: "bfe3ea",
      redFlower: "e5484d",
      yellowFlower: "f4c430",
      gravel: "a8998a",
    }),
  },
  light: {
    faces: { top: 1, bottom: 1, x: 1, z: 1 },
    ao: { r: 0.32, g: 0.32, b: 0.32 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
    sun: {
      dir: { x: 0.5, y: 0.75, z: 0.4 },
      lit: { r: 1.23, g: 1.23, b: 1.23 },
      shade: { r: 0.22, g: 0.22, b: 0.22 },
      wrap: 1,
      bands: 0,
      shine: 0,
    },
  },
  shape: { edges: true, softCorners: 0, stitch: hex("fff5e0") },
  sky: {
    day: hex("c4e3ef"),
    night: hex("1a2230"),
    sunset: hex("f2bc94"),
    sun: hex("fff2c8"),
    moon: hex("e9ecf4"),
  },
  fog: { start: 0.5 },
  water: { opacity: 0.85 },
  after: { vignette: 0.3 },
} satisfies Look;
