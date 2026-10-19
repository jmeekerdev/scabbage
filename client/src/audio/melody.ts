/** C major pentatonic from C4 to C6 in Hz; any run of these sounds pleasant together. */
export const SCALE: readonly number[] = [
  261.63, 293.66, 329.63, 392.0, 440.0, 523.25, 587.33, 659.25, 783.99, 880.0, 1046.5,
];
const HOME_EVERY = 5; // a C every five notes of the scale
const REST_CHANCE = 0.15;

export interface Note {
  freq: number;
  beat: number;
  beats: number;
}

/** A wandering phrase `total` beats long that moves in small steps and ends on a C. */
export function melody(random: () => number, total: number): Note[] {
  const notes: Note[] = [];
  let idx = HOME_EVERY;
  let beat = 0;
  while (beat < total - 2) {
    const beats = Math.min(random() < 0.25 ? 2 : 1, total - 2 - beat);
    if (random() >= REST_CHANCE) {
      notes.push({ freq: SCALE[idx] as number, beat, beats });
      idx = Math.min(SCALE.length - 1, Math.max(0, idx + Math.floor(random() * 5) - 2));
    }
    beat += beats;
  }
  const last = SCALE.indexOf(notes.at(-1)?.freq ?? (SCALE[HOME_EVERY] as number));
  const home = Math.round(last / HOME_EVERY) * HOME_EVERY;
  notes.push({ freq: SCALE[home] as number, beat, beats: total - beat });
  return notes;
}
