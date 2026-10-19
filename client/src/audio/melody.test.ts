import { describe, expect, it } from "vitest";
import { chordAt } from "./chords";
import { melody } from "./melody";

/** A repeatable stand-in for Math.random. */
function seeded(seed: number): () => number {
  return () => {
    seed = (seed * 1664525 + 1013904223) % 2 ** 32;
    return seed / 2 ** 32;
  };
}

const midi = (hz: number) => Math.round(69 + 12 * Math.log2(hz / 440));

describe("melody", () => {
  it("plays only tones of the chord sounding, in order, within the phrase", () => {
    const notes = melody(seeded(7), 24);
    expect(notes.length).toBeGreaterThan(6);
    for (const n of notes) {
      expect(chordAt(n.beat).tones).toContain(midi(n.freq) % 12);
      expect(n.beat).toBeGreaterThanOrEqual(0);
      expect(n.beat + n.beats).toBeLessThanOrEqual(24);
    }
    const beats = notes.map((n) => n.beat);
    expect(beats).toEqual([...beats].sort((a, b) => a - b));
  });

  it("never holds a note across a chord change", () => {
    for (const n of melody(seeded(5), 24)) {
      expect(chordAt(n.beat + n.beats - 0.5).name).toBe(chordAt(n.beat).name);
    }
  });

  it("moves by small steps so it stays gentle", () => {
    const notes = melody(seeded(3), 48);
    for (let i = 1; i < notes.length; i++) {
      const jump = Math.abs(midi(notes[i]?.freq ?? 0) - midi(notes[i - 1]?.freq ?? 0));
      expect(jump).toBeLessThanOrEqual(10);
    }
  });

  it("ends on a long note", () => {
    for (const seed of [1, 2, 11]) {
      const last = melody(seeded(seed), 24).at(-1);
      expect((last?.beat ?? 0) + (last?.beats ?? 0)).toBe(24);
      expect(last?.beats).toBe(2);
    }
  });
});
