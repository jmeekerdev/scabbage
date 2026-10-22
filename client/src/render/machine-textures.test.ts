import { describe, expect, it } from "vitest";
import { digitPixel, labelPixel, pipAt } from "./machine-textures";

// Draws digit d as rows of # and ., top row first, from the 6×10 area it fills.
const draw = (d: number) =>
  Array.from({ length: 10 }, (_, r) =>
    Array.from({ length: 6 }, (_, c) => (digitPixel(d, 5 + c, 12 - r) ? "#" : ".")).join(""),
  ).filter((_, r) => r % 2 === 0);

describe("digitPixel", () => {
  it("draws counter digits upright at double size", () => {
    expect(draw(7)).toEqual(["######", "....##", "..##..", "..##..", "..##.."]);
    expect(draw(4)).toEqual(["##..##", "##..##", "######", "....##", "....##"]);
  });

  it("leaves the border dark", () => {
    expect(
      digitPixel(8, 4, 8) || digitPixel(8, 11, 8) || digitPixel(8, 8, 2) || digitPixel(8, 8, 13),
    ).toBe(false);
  });
});

// Draws a label as rows of # and ., top row first, across the 5 rows it fills.
const label = (text: string, from: number, to: number) =>
  Array.from({ length: 5 }, (_, r) =>
    Array.from({ length: to - from + 1 }, (_, c) =>
      labelPixel(text, from + c, 10 - r) ? "#" : ".",
    ).join(""),
  );

describe("labelPixel", () => {
  it("writes a word upright in small letters across the middle", () => {
    expect(label("NOT", 2, 12)).toEqual([
      "#.#.###.###",
      "###.#.#..#.",
      "###.#.#..#.",
      "#.#.#.#..#.",
      "#.#.###..#.",
    ]);
  });

  it("fits four letters and leaves the rows above and below clear", () => {
    expect(label("NAND", 0, 15).every((row) => row.length === 16)).toBe(true);
    expect(labelPixel("NAND", 0, 10) && labelPixel("NAND", 13, 6)).toBe(true);
    expect(labelPixel("NAND", 7, 11) || labelPixel("NAND", 7, 5)).toBe(false);
  });
});

describe("pipAt", () => {
  it("numbers eight pips in two rows of four, top row first, around the middle stripe", () => {
    expect([pipAt(3, 11), pipAt(4, 12), pipAt(12, 11), pipAt(3, 3), pipAt(13, 4)]).toEqual([
      0, 0, 3, 4, 7,
    ]);
  });

  it("leaves the gaps and the stripe without a pip", () => {
    expect([pipAt(5, 3), pipAt(3, 7), pipAt(14, 12), pipAt(3, 2)]).toEqual([-1, -1, -1, -1]);
  });

  it("fits ten pips in rows of five, for a capacitor", () => {
    expect([pipAt(1, 11, 5), pipAt(13, 12, 5), pipAt(1, 4, 5), pipAt(14, 3, 5)]).toEqual([
      0, 4, 5, 9,
    ]);
    expect([pipAt(3, 11, 5), pipAt(0, 11, 5), pipAt(15, 11, 5)]).toEqual([-1, -1, -1]);
  });
});
