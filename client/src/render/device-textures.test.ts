import { describe, expect, it } from "vitest";
import { arrowPixel, DEVICE_PAINTERS } from "./device-textures";

describe("arrowPixel", () => {
  it("draws a wide head at the end it points to and a thin shaft behind", () => {
    expect([arrowPixel(0, 1, 7, 13), arrowPixel(0, 1, 3, 8), arrowPixel(0, 1, 3, 6)]).toEqual([
      true,
      true,
      false,
    ]);
    expect([arrowPixel(0, 1, 7, 3), arrowPixel(0, 1, 7, 15)]).toEqual([true, false]);
  });

  it("turns the same arrow each way", () => {
    for (let x = 0; x < 16; x++)
      for (let y = 0; y < 16; y++) {
        expect(arrowPixel(0, -1, x, y)).toBe(arrowPixel(0, 1, x, 15 - y));
        expect(arrowPixel(1, 0, x, y)).toBe(arrowPixel(0, 1, y, x));
        expect(arrowPixel(-1, 0, x, y)).toBe(arrowPixel(1, 0, 15 - x, y));
      }
  });
});

describe("timer faces", () => {
  it("brighten while on and differ by speed", () => {
    const at = (name: string) => {
      let s = 1;
      return DEVICE_PAINTERS[name]!(7, 4, () => (s = (s * 7) % 11) / 11);
    };
    const sum = (p: number[]) => p[0]! + p[1]! + p[2]!;
    expect(sum(at("timer_1_on"))).toBeGreaterThan(sum(at("timer_1")));
    expect(at("timer_1_on")).not.toEqual(at("timer_16_on"));
  });
});
