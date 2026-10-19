import { BAR_BEATS, chordAt, midiHz } from "./chords";

const LOW = 60; // C4
const HIGH = 84; // C6
const START = 72;
const REST_CHANCE = 0.15;

export interface Note {
  freq: number;
  beat: number;
  beats: number;
}

/** A wandering phrase `total` beats long that steps between tones of whichever chord is sounding. */
export function melody(random: () => number, total: number): Note[] {
  const notes: Note[] = [];
  let pitch = START;
  let beat = 0;
  while (beat < total - 2) {
    const toBarEnd = BAR_BEATS - (beat % BAR_BEATS);
    const beats = Math.min(random() < 0.25 ? 2 : 1, toBarEnd, total - 2 - beat);
    if (random() >= REST_CHANCE) {
      pitch = step(chordTones(beat), pitch, Math.floor(random() * 5) - 2);
      notes.push({ freq: midiHz(pitch), beat, beats });
    }
    beat += beats;
  }
  pitch = step(chordTones(beat), pitch, 0);
  notes.push({ freq: midiHz(pitch), beat, beats: total - beat });
  return notes;
}

/** Every MIDI note in the melody's range that belongs to the chord at `beat`. */
function chordTones(beat: number): number[] {
  const { tones } = chordAt(beat);
  const pool: number[] = [];
  for (let n = LOW; n <= HIGH; n++) if (tones.includes(n % 12)) pool.push(n);
  return pool;
}

/** Moves `by` places through `pool`, starting from the tone nearest `pitch`. */
function step(pool: number[], pitch: number, by: number): number {
  let nearest = 0;
  pool.forEach((n, i) => {
    if (Math.abs(n - pitch) < Math.abs((pool[nearest] ?? 0) - pitch)) nearest = i;
  });
  return pool[Math.min(pool.length - 1, Math.max(0, nearest + by))] as number;
}
