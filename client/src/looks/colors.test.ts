import { describe, expect, it } from "vitest";
import { TEXTURE_NAMES } from "../render/textures";
import { blockPalette, hex } from "./colors";

describe("hex", () => {
  it("reads a six-digit sRGB colour as 0–1 channels", () => {
    expect(hex("ff8000")).toEqual({ r: 1, g: 128 / 255, b: 0 });
  });
});

describe("blockPalette", () => {
  const pal = blockPalette({ grass: "00ff00", dirt: "ff0000", log: "0000ff" });

  it("colours every texture a block shows", () => {
    expect(pal.grass_top).toEqual(hex("00ff00"));
    expect(pal.log_top).toEqual(hex("0000ff"));
    expect(pal.log_side).toEqual(hex("0000ff"));
  });

  it("gives grass sides mostly soil with a little grass", () => {
    expect(pal.grass_side).toEqual({ r: 0.7, g: 0.3, b: 0 });
  });

  it("names only textures that exist", () => {
    const all = blockPalette({
      grass: "000000",
      dirt: "000000",
      stone: "000000",
      sand: "000000",
      water: "000000",
      log: "000000",
      leaves: "000000",
      pine: "000000",
      planks: "000000",
      brick: "000000",
      glass: "000000",
      redFlower: "000000",
      yellowFlower: "000000",
      gravel: "000000",
    });
    for (const name of Object.keys(all)) expect(TEXTURE_NAMES).toContain(name);
  });
});
