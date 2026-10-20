import { describe, expect, it } from "vitest";
import { digitPixel } from "./machine-textures";

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
