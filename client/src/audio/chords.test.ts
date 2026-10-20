import { describe, expect, it } from "vitest";
import { BAR_BEATS, chordAt, midiHz, PROGRESSION } from "./chords";

const pitchClasses = (notes: readonly number[]) => [...new Set(notes.map((n) => n % 12))].sort();

describe("chords", () => {
  it("converts MIDI notes to Hz", () => {
    expect(midiHz(69)).toBe(440);
    expect(midiHz(57)).toBeCloseTo(220);
  });

  it("plays G♭maj9, D♭maj7, A♭/B♭ a bar each, then loops", () => {
    const names = [0, 1, 2, 3].map((bar) => chordAt(bar * BAR_BEATS + 1).name);
    expect(names).toEqual(["Gbmaj9", "Dbmaj7", "Ab/Bb", "Gbmaj9"]);
  });

  it("spells each chord correctly over its bass", () => {
    const [gb, db, ab] = PROGRESSION;
    expect(gb?.bass).toBe(42); // G♭2
    expect(pitchClasses(gb?.voicing ?? [])).toEqual([1, 10, 5, 8].sort()); // B♭ D♭ F A♭
    expect(db?.bass).toBe(37); // D♭2
    expect(pitchClasses(db?.voicing ?? [])).toEqual([5, 8, 0].sort()); // F A♭ C
    expect(ab?.bass).toBe(46); // B♭2
    expect(pitchClasses(ab?.voicing ?? [])).toEqual([8, 0, 3].sort()); // A♭ C E♭
  });

  it("moves each pad voice by at most a step between chords", () => {
    PROGRESSION.forEach((chord, i) => {
      const next = PROGRESSION[(i + 1) % PROGRESSION.length];
      chord.voicing.forEach((n, v) =>
        expect(Math.abs((next?.voicing[v] ?? 0) - n)).toBeLessThanOrEqual(2),
      );
    });
  });
});
