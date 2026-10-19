import { describe, expect, it } from "vitest";
import { Hotbar, hotbarBlocks } from "./hotbar";

describe("Hotbar", () => {
  it("selects slots by number and wraps when scrolling", () => {
    const h = new Hotbar([3, 4, 5]);
    expect(h.current()).toBe(3);
    h.select(2);
    expect(h.current()).toBe(5);
    h.select(7);
    expect(h.current()).toBe(5);
    h.scroll(1);
    expect(h.current()).toBe(3);
    h.scroll(-1);
    expect(h.current()).toBe(5);
  });

  it("picks a block by selecting its slot, or putting it in the current one", () => {
    const h = new Hotbar([3, 4, 5]);
    h.pick(4);
    expect(h.selected).toBe(1);
    h.pick(9);
    expect(h.slots).toEqual([3, 9, 5]);
    expect(h.current()).toBe(9);
  });

  it("tells listeners about every change", () => {
    const h = new Hotbar([3, 4]);
    let calls = 0;
    h.onChange = () => calls++;
    h.select(1);
    h.scroll(1);
    h.pick(7);
    expect(calls).toBe(3);
  });
});

describe("hotbarBlocks", () => {
  it("fills the bar with known building blocks in order", () => {
    const types = [
      { id: 0, name: "air" },
      { id: 1, name: "stone" },
      { id: 3, name: "grass" },
      { id: 8, name: "planks" },
    ];
    expect(hotbarBlocks(types)).toEqual([3, 1, 8]);
  });
});

describe("Hotbar.carry", () => {
  const stacks = (...s: [number, number][]) => s.map(([block, count]) => ({ block, count }));

  it("shows the inventory's first slots with counts, and empty slots as air", () => {
    const h = new Hotbar([3, 4, 5]);
    h.carry(stacks([7, 2], [0, 0], [8, 0], [9, 1]));
    expect(h.slots).toEqual([7, 0, 0, 9]);
    expect(h.counts).toEqual([2, 0, 0, 1]);
    h.select(1);
    expect(h.current()).toBe(0);
  });

  it("only selects blocks already carried when picking", () => {
    const h = new Hotbar([3, 4, 5]);
    h.carry(stacks([7, 2], [8, 1]));
    h.pick(8);
    expect(h.selected).toBe(1);
    h.pick(99);
    expect(h.slots).toEqual([7, 8]);
  });

  it("gets the creative blocks back afterwards", () => {
    const h = new Hotbar([3, 4, 5]);
    h.carry(stacks([7, 2]));
    h.carry(stacks([7, 1]));
    h.carry(undefined);
    expect(h.slots).toEqual([3, 4, 5]);
    expect(h.counts).toBeUndefined();
  });
});
