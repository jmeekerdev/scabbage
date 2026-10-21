import { blockPalette, hex } from "./colors";
import type { Look } from "./look";

// Light numbers are the look book's sRGB ones raised to 2.2, to match in linear light.
export default {
  name: "Ink & Cel",
  about: "Bright cartoon colours with ink lines",
  surface: {
    textured: false,
    tint: 0,
    solid: ["leaves", "pine_leaves"],
    palette: blockPalette({
      grass: "5fcf4a",
      dirt: "a8693b",
      stone: "9aa3ad",
      sand: "f6d98b",
      water: "2fa8e0",
      log: "8a5530",
      leaves: "3fbf5a",
      pine: "239a73",
      planks: "f0b25a",
      brick: "e2483d",
      glass: "a6e8ff",
      redFlower: "ff3d6e",
      yellowFlower: "ffd400",
      gravel: "b8aa98",
    }),
  },
  light: {
    faces: { top: 1, bottom: 1, x: 1, z: 1 },
    ao: { r: 0.61, g: 0.61, b: 0.61 },
    lamp: { r: 1, g: 0.86, b: 0.66 },
    lampFade: 0.82,
    sun: {
      dir: { x: 0.55, y: 0.75, z: 0.35 },
      lit: { r: 1, g: 1, b: 1 },
      shade: { r: 0.29, g: 0.31, b: 0.46 },
      wrap: 0,
      bands: 3,
      shine: 0,
    },
  },
  shape: { edges: false, softCorners: 0 },
  sky: {
    day: hex("8ccaff"),
    night: hex("101838"),
    sunset: hex("ff9a6a"),
    sun: hex("fff27a"),
    moon: hex("eef2ff"),
  },
  fog: { start: 0.6 },
  water: { opacity: 0.85 },
  after: {
    outline: { width: 1.3, strength: 1, color: hex("1d2340"), creases: true },
    saturation: 1.08,
  },
} satisfies Look;
