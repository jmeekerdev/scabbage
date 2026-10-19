import { describe, expect, it } from "vitest";
import { Inventory, SlotMover } from "./inventory";

const stacks = (...s: [number, number][]) => s.map(([block, count]) => ({ block, count }));

describe("Inventory", () => {
  it("reports when items were picked up", () => {
    const inv = new Inventory();
    expect(inv.set(stacks([3, 1]))).toBe(true);
    expect(inv.set(stacks([3, 1]))).toBe(false);
    expect(inv.set(stacks([3, 0], [3, 1]))).toBe(false); // moved, not gained
    expect(inv.set(stacks([3, 2]))).toBe(true);
  });

  it("spends a block from a slot until it is empty", () => {
    const inv = new Inventory();
    inv.set(stacks([3, 1]));
    expect(inv.spend(0)).toBe(true);
    expect(inv.slots[0]).toEqual({ block: 0, count: 0 });
    expect(inv.spend(0)).toBe(false);
    expect(inv.spend(5)).toBe(false);
  });

  it("tells listeners about changes", () => {
    const inv = new Inventory();
    let calls = 0;
    inv.onChange = () => calls++;
    inv.set(stacks([3, 2]));
    inv.spend(0);
    expect(calls).toBe(2);
  });
});

describe("SlotMover", () => {
  it("moves from the first slot clicked to the second", () => {
    const m = new SlotMover();
    expect(m.click(4)).toBeUndefined();
    expect(m.held).toBe(4);
    expect(m.click(7)).toEqual([4, 7]);
    expect(m.held).toBeUndefined();
  });

  it("clicking the same slot again lets go", () => {
    const m = new SlotMover();
    m.click(2);
    expect(m.click(2)).toBeUndefined();
    expect(m.held).toBeUndefined();
  });
});
