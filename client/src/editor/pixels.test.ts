import { describe, expect, it } from "vitest";
import { hexColor, parseHex, PixelImage, SWATCHES, Undo } from "./pixels";

const RED = [255, 0, 0, 255] as const;
const BLUE = [0, 0, 255, 255] as const;

describe("PixelImage", () => {
  it("sets and gets pixels, reporting whether anything changed", () => {
    const img = new PixelImage(4, 4);
    expect(img.set(1, 2, RED)).toBe(true);
    expect(img.set(1, 2, RED)).toBe(false);
    expect(img.get(1, 2)).toEqual(RED);
    expect(img.get(0, 0)).toEqual([0, 0, 0, 0]);
  });

  it("ignores pixels outside the picture", () => {
    const img = new PixelImage(2, 2);
    expect(img.set(2, 0, RED)).toBe(false);
    expect(img.get(-1, 0)).toEqual([0, 0, 0, 0]);
  });

  it("flood-fills only the touching area of one color, inside the bounds", () => {
    const img = new PixelImage(4, 4);
    for (let y = 0; y < 4; y++) img.set(2, y, BLUE); // a wall at x=2
    img.fill(0, 0, RED, { x: 0, y: 0, w: 4, h: 3 });
    expect(img.get(1, 2)).toEqual(RED);
    expect(img.get(0, 3)).toEqual([0, 0, 0, 0]); // outside the bounds
    expect(img.get(3, 0)).toEqual([0, 0, 0, 0]); // behind the wall
    expect(img.get(2, 0)).toEqual(BLUE);
  });

  it("filling with the same color does nothing", () => {
    const img = new PixelImage(2, 2);
    img.set(0, 0, RED);
    expect(img.fill(0, 0, RED, { x: 0, y: 0, w: 2, h: 2 })).toBe(false);
  });
});

describe("Undo", () => {
  it("brings back earlier pictures, newest first", () => {
    const img = new PixelImage(1, 1);
    const undo = new Undo(img);
    undo.save();
    img.set(0, 0, RED);
    undo.save();
    img.set(0, 0, BLUE);
    expect(undo.back()).toBe(true);
    expect(img.get(0, 0)).toEqual(RED);
    expect(undo.back()).toBe(true);
    expect(img.get(0, 0)).toEqual([0, 0, 0, 0]);
    expect(undo.back()).toBe(false);
  });

  it("keeps a limited number of steps", () => {
    const img = new PixelImage(1, 1);
    const undo = new Undo(img, 2);
    for (let i = 0; i < 5; i++) undo.save();
    expect([undo.back(), undo.back(), undo.back()]).toEqual([true, true, false]);
  });
});

describe("colors", () => {
  it("round-trips hex colors", () => {
    expect(parseHex("#ff8000")).toEqual([255, 128, 0, 255]);
    expect(hexColor([255, 128, 0, 255])).toBe("#ff8000");
  });

  it("offers a handful of distinct swatches", () => {
    expect(new Set(SWATCHES).size).toBe(SWATCHES.length);
    expect(SWATCHES.length).toBeGreaterThanOrEqual(16);
  });
});
