import { BAR_BEATS, chordAt } from "./chords";

/** A [MIDI note, beats] pair; note 0 is a rest. */
type Sung = readonly [number, number];

export interface Section {
  name: string;
  bars: number;
  level: number; // 0..1, how loud the whole band plays
  melody?: readonly Sung[];
  arpeggio?: boolean; // a flowing chord figure under the tune
}

export interface Note {
  midi: number;
  beat: number; // from the start of the section
  beats: number;
}

// Each tune is six bars: G♭maj9 → D♭maj7 → A♭/B♭ twice, written one bar per line.
// prettier-ignore
const VERSE_A: readonly Sung[] = [
  [70, 2], [68, 1], [65, 1],
  [68, 3], [65, 1],
  [63, 2], [65, 1], [68, 1],
  [70, 2], [73, 2],
  [72, 2], [68, 2],
  [70, 1], [68, 3],
];

// prettier-ignore
const VERSE_B: readonly Sung[] = [
  [70, 2], [68, 1], [65, 1],
  [68, 2], [72, 2],
  [75, 3], [72, 1],
  [70, 2], [68, 1], [66, 1],
  [65, 3], [63, 1],
  [63, 2], [0, 2],
];

// prettier-ignore
const CHORUS_RISE: readonly Sung[] = [
  [73, 1], [77, 3],
  [80, 2], [77, 1], [75, 1],
  [75, 2], [72, 2],
  [73, 1], [77, 2], [78, 1],
  [80, 3], [77, 1],
  [75, 2], [72, 2],
];

// prettier-ignore
const CHORUS_HOME: readonly Sung[] = [
  ...CHORUS_RISE.slice(0, 7), // its first three bars
  [82, 2], [80, 1], [77, 1],
  [77, 2], [75, 1], [73, 1],
  [72, 4],
];

const verse = (name: string, melody: readonly Sung[]): Section => ({
  name,
  bars: 6,
  level: 0.75,
  melody,
});
const chorus = (name: string, melody: readonly Sung[]): Section => ({
  name,
  bars: 6,
  level: 1,
  melody,
  arpeggio: true,
});

/** Intro, two verses, the chorus, a verse, the chorus again, then a soft outro. */
export const SONG: readonly Section[] = [
  { name: "intro", bars: 3, level: 0.5 },
  verse("verse a", VERSE_A),
  verse("verse b", VERSE_B),
  chorus("chorus rise", CHORUS_RISE),
  chorus("chorus home", CHORUS_HOME),
  verse("verse a again", VERSE_A),
  chorus("chorus rise again", CHORUS_RISE),
  chorus("chorus home again", CHORUS_HOME),
  { name: "outro", bars: 3, level: 0.4 },
];

/** A section's tune as timed notes, skipping rests. */
export function sectionNotes(s: Section): Note[] {
  const notes: Note[] = [];
  let beat = 0;
  for (const [midi, beats] of s.melody ?? []) {
    if (midi) notes.push({ midi, beat, beats });
    beat += beats;
  }
  return notes;
}

const ARPEGGIO = [0, 1, 2, 3, 2, 1, 2, 3]; // pad voices, one per half beat

/** The pad's chord tones rippling up and down in half beats for `bars` bars. */
export function arpeggioNotes(bars: number): Note[] {
  const notes: Note[] = [];
  for (let beat = 0; beat < bars * BAR_BEATS; beat += 0.5) {
    const { voicing } = chordAt(beat);
    const step = ARPEGGIO[(beat * 2) % ARPEGGIO.length] ?? 0;
    notes.push({ midi: voicing[step] ?? voicing[0]!, beat, beats: 0.5 });
  }
  return notes;
}

/** The tune and arpeggio notes that start in one bar of a section, timed from that bar. */
export function barNotes(s: Section, bar: number): { tune: Note[]; arpeggio: Note[] } {
  const from = bar * BAR_BEATS;
  const inBar = (notes: Note[]) =>
    notes
      .filter((n) => n.beat >= from && n.beat < from + BAR_BEATS)
      .map((n) => ({ ...n, beat: n.beat - from }));
  return { tune: inBar(sectionNotes(s)), arpeggio: s.arpeggio ? inBar(arpeggioNotes(s.bars)) : [] };
}
