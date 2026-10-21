import { describe, expect, it } from "vitest";
import { Veil } from "./veil";

describe("Veil", () => {
  it("starts clear and closes over its time", () => {
    const v = new Veil(200);
    expect(v.amount(0)).toBe(0);
    v.close(1000);
    expect(v.amount(1100)).toBeCloseTo(0.5);
    expect(v.closed(1100)).toBe(false);
    expect(v.amount(1300)).toBe(1);
    expect(v.closed(1300)).toBe(true);
  });

  it("opens from wherever it got to", () => {
    const v = new Veil(200);
    v.close(0);
    v.open(100);
    expect(v.amount(100)).toBeCloseTo(0.5);
    expect(v.amount(300)).toBe(0);
    expect(v.closed(300)).toBe(false);
  });
});
