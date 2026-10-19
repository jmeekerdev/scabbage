import { describe, expect, it } from "vitest";
import { Miner } from "./mining";

const block = { x: 1, y: 2, z: 3 };

describe("Miner", () => {
  it("finishes after the block's mining time", () => {
    const m = new Miner();
    expect(m.dig(0.25, block, 1000)).toBeCloseTo(0.25);
    expect(m.dig(0.5, block, 1000)).toBeCloseTo(0.75);
    expect(m.dig(0.25, block, 1000)).toBe(1);
  });

  it("starts over on a new block or after letting go", () => {
    const m = new Miner();
    m.dig(0.5, block, 1000);
    expect(m.dig(0.25, { ...block, x: 9 }, 1000)).toBeCloseTo(0.25);
    m.stop();
    expect(m.dig(0.25, { ...block, x: 9 }, 1000)).toBeCloseTo(0.25);
  });

  it("starts the next dig from zero once a block breaks", () => {
    const m = new Miner();
    m.dig(1, block, 500);
    expect(m.dig(0.1, block, 1000)).toBeCloseTo(0.1);
  });

  it("breaks instant blocks at once", () => {
    expect(new Miner().dig(0, block, 0)).toBe(1);
  });
});
