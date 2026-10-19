import { describe, expect, it } from "vitest";
import { canCraft } from "./crafting";

const stacks = (...s: [number, number][]) => s.map(([block, count]) => ({ block, count }));
const brick = { make: { block: 11, count: 2 }, from: stacks([2, 1], [4, 1]) };

describe("canCraft", () => {
  it("needs every input", () => {
    expect(canCraft(brick, stacks([2, 1]))).toBe(false);
    expect(canCraft(brick, stacks([2, 1], [4, 1]))).toBe(true);
  });

  it("adds up inputs spread over several slots", () => {
    const wool = { make: { block: 13, count: 1 }, from: stacks([7, 4]) };
    expect(canCraft(wool, stacks([7, 3]))).toBe(false);
    expect(canCraft(wool, stacks([7, 3], [0, 0], [7, 1]))).toBe(true);
  });
});
