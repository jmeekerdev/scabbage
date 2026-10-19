import { describe, expect, it } from "vitest";
import { melody, SCALE } from "./melody";

/** A repeatable stand-in for Math.random. */
function seeded(seed: number): () => number {
  return () => {
    seed = (seed * 1664525 + 1013904223) % 2 ** 32;
    return seed / 2 ** 32;
  };
}

describe("melody", () => {
  it("plays only notes from the scale, in order, within the phrase", () => {
    const notes = melody(seeded(7), 16);
    expect(notes.length).toBeGreaterThan(4);
    for (const n of notes) {
      expect(SCALE).toContain(n.freq);
      expect(n.beat).toBeGreaterThanOrEqual(0);
      expect(n.beat + n.beats).toBeLessThanOrEqual(16);
    }
    const beats = notes.map((n) => n.beat);
    expect(beats).toEqual([...beats].sort((a, b) => a - b));
  });

  it("moves by small steps so it stays gentle", () => {
    const notes = melody(seeded(3), 32);
    for (let i = 1; i < notes.length; i++) {
      const jump = Math.abs(
        SCALE.indexOf(notes[i]?.freq ?? 0) - SCALE.indexOf(notes[i - 1]?.freq ?? 0),
      );
      expect(jump).toBeLessThanOrEqual(2);
    }
  });

  it("ends on a home note", () => {
    for (const seed of [1, 2, 11]) {
      expect([SCALE[0], SCALE[5], SCALE[10]]).toContain(melody(seeded(seed), 16).at(-1)?.freq);
    }
  });
});
