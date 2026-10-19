import { describe, expect, it } from "vitest";
import { Gait } from "./gait";

/** Feeds the gait one 50 ms frame per x position and returns the limb swings seen. */
function walk(xs: number[]): number[] {
  const g = new Gait();
  return xs.map((x) => g.update({ x, y: 10, z: 0 }, 50).swing);
}

describe("Gait", () => {
  it("keeps limbs still while standing", () => {
    expect(walk([0, 0, 0, 0])).toEqual([0, 0, 0, 0]);
  });

  it("swings limbs back and forth while walking", () => {
    const xs = Array.from({ length: 40 }, (_, i) => i * 0.2); // 4 blocks/s
    const swings = walk(xs);
    expect(Math.max(...swings)).toBeGreaterThan(0.4);
    expect(Math.min(...swings)).toBeLessThan(-0.4);
  });

  it("settles back to still after stopping", () => {
    const xs = [...Array.from({ length: 20 }, (_, i) => i * 0.2), ...Array<number>(20).fill(3.8)];
    expect(Math.abs(walk(xs).at(-1) ?? 1)).toBeLessThan(0.05);
  });

  it("ignores teleport-sized jumps", () => {
    expect(walk([0, 50, 50])).toEqual([0, 0, 0]);
  });

  it("knows when the figure is off the ground", () => {
    const g = new Gait();
    g.update({ x: 0, y: 10, z: 0 }, 50);
    expect(g.update({ x: 0, y: 10.4, z: 0 }, 50).airborne).toBe(true);
    expect(g.update({ x: 0, y: 10.4, z: 0 }, 50).airborne).toBe(false);
  });
});
