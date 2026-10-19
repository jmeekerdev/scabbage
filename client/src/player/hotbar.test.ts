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
