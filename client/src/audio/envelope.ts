/** Gain shapes for music notes, kept apart from the Web Audio calls so they can be tested. */

export interface SwellSpec {
  start: number;
  dur: number;
  gain: number;
  attack: number; // seconds to rise
  release: number; // seconds to fade after dur
}

/** [time, gain] points: a soft rise, a hold, then a long fade that blends into the next note. */
export function swell(s: SwellSpec): [number, number][] {
  const rise = Math.min(s.attack, s.dur / 2);
  const end = s.start + s.dur;
  return [
    [s.start, 0],
    [s.start + rise, s.gain],
    [end, s.gain],
    [end + s.release, 0],
  ];
}
