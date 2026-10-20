import { describe, expect, it } from "vitest";
import { CALM_STEP_MS, IDLE_AFTER_MS, IDLE_STEP_MS, Pacer } from "./pacer";

describe("pacer", () => {
  it("runs at full speed while the view keeps changing", () => {
    const p = new Pacer();
    for (let t = 0; t < 5000; t += 16) p.frame(t, "view");
    expect(p.step()).toBeUndefined();
  });

  it("slows down once the view has been still for a while", () => {
    const p = new Pacer();
    p.frame(0, "view");
    p.frame(IDLE_AFTER_MS - 1, "none");
    expect(p.step()).toBeUndefined();
    p.frame(IDLE_AFTER_MS, "none");
    expect(p.step()).toBe(IDLE_STEP_MS);
  });

  it("keeps up with things moving in a still view, without full speed", () => {
    const p = new Pacer();
    p.frame(0, "view");
    for (let t = 0; t <= 3 * IDLE_AFTER_MS; t += 300) p.frame(t, "objects");
    expect(p.step()).toBe(CALM_STEP_MS);
    p.frame(5 * IDLE_AFTER_MS, "none");
    expect(p.step()).toBe(IDLE_STEP_MS);
  });

  it("wakes on a view change or input", () => {
    const p = new Pacer();
    p.frame(0, "none");
    p.frame(IDLE_AFTER_MS, "none");
    p.frame(IDLE_AFTER_MS + 100, "view");
    expect(p.step()).toBeUndefined();
    p.frame(5 * IDLE_AFTER_MS, "none");
    p.wake(5 * IDLE_AFTER_MS + 1);
    expect(p.step()).toBeUndefined();
    p.frame(5 * IDLE_AFTER_MS + 2, "none");
    expect(p.step()).toBeUndefined();
  });
});
