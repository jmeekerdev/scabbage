import { describe, expect, it } from "vitest";
import { BAR_BEATS, chordAt, PROGRESSION } from "./chords";
import { arpeggioNotes, barNotes, sectionNotes, SONG } from "./song";

const D_FLAT_MAJOR = [1, 3, 5, 6, 8, 10, 0];
const sectionBeats = (bars: number) => bars * BAR_BEATS;

describe("song", () => {
  it("starts every section at the top of the progression", () => {
    for (const s of SONG) expect(s.bars % PROGRESSION.length).toBe(0);
  });

  it("fills each sung section exactly, in D♭ major and a singable range", () => {
    for (const s of SONG.filter((s) => s.melody)) {
      const sung = s.melody!.reduce((sum, [, beats]) => sum + beats, 0);
      expect(sung, s.name).toBe(sectionBeats(s.bars));
      for (const n of sectionNotes(s)) {
        expect(D_FLAT_MAJOR, s.name).toContain(n.midi % 12);
        expect(n.midi).toBeGreaterThanOrEqual(60);
        expect(n.midi).toBeLessThanOrEqual(84);
      }
    }
  });

  it("lands on a tone of the chord at every bar line", () => {
    for (const s of SONG) {
      for (const n of sectionNotes(s).filter((n) => n.beat % BAR_BEATS === 0))
        expect(chordAt(n.beat).tones, `${s.name} beat ${n.beat}`).toContain(n.midi % 12);
    }
  });

  it("swells into a louder chorus and frames it with quiet melody-free ends", () => {
    const chorus = SONG.filter((s) => s.name.startsWith("chorus"));
    const verse = SONG.filter((s) => s.name.startsWith("verse"));
    expect(chorus.length).toBeGreaterThan(0);
    for (const c of chorus) {
      expect(c.arpeggio).toBe(true);
      for (const v of verse) expect(c.level).toBeGreaterThan(v.level);
    }
    for (const end of [SONG[0], SONG.at(-1)]) {
      expect(end?.melody).toBeUndefined();
      for (const v of verse) expect(end?.level ?? 1).toBeLessThan(v.level);
    }
  });

  it("arpeggiates the sounding chord in steady half beats", () => {
    const notes = arpeggioNotes(6);
    expect(notes.length).toBe(6 * BAR_BEATS * 2);
    notes.forEach((n, i) => {
      expect(n.beat).toBe(i / 2);
      expect(chordAt(n.beat).voicing).toContain(n.midi);
    });
  });

  it("splits each section into bars without changing a note", () => {
    for (const s of SONG) {
      const bars = Array.from({ length: s.bars }, (_, bar) => barNotes(s, bar));
      const fromBar = (bar: number) => (n: { beat: number }) => ({
        ...n,
        beat: n.beat + bar * BAR_BEATS,
      });
      expect(
        bars.flatMap((b, bar) => b.tune.map(fromBar(bar))),
        s.name,
      ).toEqual(sectionNotes(s));
      const arpeggio = s.arpeggio ? arpeggioNotes(s.bars) : [];
      expect(
        bars.flatMap((b, bar) => b.arpeggio.map(fromBar(bar))),
        s.name,
      ).toEqual(arpeggio);
    }
  });
});
