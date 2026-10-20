import { describe, expect, it } from "vitest";
import { swell } from "./envelope";

describe("swell", () => {
  it("rises slowly, holds, then fades past the note's end so notes overlap", () => {
    expect(swell({ start: 1, dur: 2, gain: 0.5, attack: 0.4, release: 1 })).toEqual([
      [1, 0],
      [1.4, 0.5],
      [3, 0.5],
      [4, 0],
    ]);
  });

  it("shortens the rise of notes shorter than it", () => {
    const [, peak] = swell({ start: 0, dur: 0.2, gain: 1, attack: 0.4, release: 0.5 });
    expect(peak).toEqual([0.1, 1]);
  });
});
