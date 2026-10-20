/** Note blocks: which pitch to play when one changes. */

// MIDI notes of the C major scale from C5, one per note block pitch.
const SCALE = [72, 74, 76, 77, 79, 81, 83, 84];

interface NoteState {
  midi: number;
  sounding: boolean;
}

/** Each note block ID's pitch and whether it is the powered, sounding state. */
export function noteTable(blocks: readonly { id: number; name: string }[]): Map<number, NoteState> {
  const out = new Map<number, NoteState>();
  for (const { id, name } of blocks) {
    const m = name.match(/^note_block(?:_(\d))?(_on)?$/);
    if (m) out.set(id, { midi: SCALE[Number(m[1] ?? 0)] ?? SCALE[0]!, sounding: !!m[2] });
  }
  return out;
}

/** The MIDI note to play when a block changes from `was` to `now`: on power or a new pitch. */
export function noteToPlay(
  table: ReadonlyMap<number, NoteState>,
  was: number | undefined,
  now: number,
): number | undefined {
  const [before, after] = [table.get(was ?? -1), table.get(now)];
  if (!after || !before) return undefined;
  return (after.sounding && !before.sounding) || after.midi !== before.midi
    ? after.midi
    : undefined;
}
