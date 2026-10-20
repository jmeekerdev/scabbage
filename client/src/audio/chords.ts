/** Equal-tempered pitch of a MIDI note number in Hz. */
export const midiHz = (note: number): number => 440 * 2 ** ((note - 69) / 12);

export interface Chord {
  name: string;
  bass: number; // MIDI note
  voicing: readonly number[]; // MIDI notes, voice-led so each change moves a step at most
  tones: readonly number[]; // pitch classes (0 = C) the melody may use
}

export const BAR_BEATS = 4;

/** G♭maj9 → D♭maj7 → A♭/B♭, one bar each, looping. */
export const PROGRESSION: readonly Chord[] = [
  { name: "Gbmaj9", bass: 42, voicing: [58, 61, 65, 68], tones: [6, 10, 1, 5, 8] },
  { name: "Dbmaj7", bass: 37, voicing: [56, 60, 65, 68], tones: [1, 5, 8, 0] },
  { name: "Ab/Bb", bass: 46, voicing: [56, 60, 63, 68], tones: [10, 0, 3, 5, 8] },
];

export function chordAt(beat: number): Chord {
  return PROGRESSION[Math.floor(beat / BAR_BEATS) % PROGRESSION.length] as Chord;
}
