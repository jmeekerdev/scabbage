import { describe, expect, it } from "vitest";
import { coordsText } from "./coords";

describe("coordsText", () => {
  it("shows the block you stand in and the way you face", () => {
    expect(coordsText({ x: 96.7, y: 40.2, z: -86.5 }, 0)).toBe("📍 96, 40, -87 · facing north");
    expect(coordsText({ x: 0, y: 0, z: 0 }, Math.PI / 2)).toMatch(/west$/);
    expect(coordsText({ x: 0, y: 0, z: 0 }, -Math.PI / 2)).toMatch(/east$/);
    expect(coordsText({ x: 0, y: 0, z: 0 }, Math.PI)).toMatch(/south$/);
    expect(coordsText({ x: 0, y: 0, z: 0 }, 2 * Math.PI - 0.1)).toMatch(/north$/);
  });
});
