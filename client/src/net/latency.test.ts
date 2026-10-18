import { describe, expect, it } from "vitest";
import { LatencyTracker } from "./latency";

describe("LatencyTracker", () => {
  it("has no estimate before any sample", () => {
    expect(new LatencyTracker().rttMs).toBeUndefined();
  });

  it("uses the first sample directly", () => {
    const t = new LatencyTracker();
    t.record(100, 140);
    expect(t.rttMs).toBe(40);
  });

  it("smooths later samples", () => {
    const t = new LatencyTracker(0.5);
    t.record(0, 40);
    t.record(0, 80);
    expect(t.rttMs).toBe(60);
  });

  it("ignores negative samples from clock weirdness", () => {
    const t = new LatencyTracker();
    t.record(0, 40);
    t.record(100, 50);
    expect(t.rttMs).toBe(40);
  });
});
