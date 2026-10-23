import { BAR_BEATS, chordAt, midiHz } from "./chords";
import { barNotes, type Section, SONG } from "./song";
import { echo, noise, noiseBuffer, tone, voice } from "./synth";

export type Cue = "step" | "break" | "place" | "pop" | "hurt" | "bonk";

const BEAT = 0.65; // seconds; slow and calm
const PAD_GAIN = 0.1; // per voice, under the melody
const MUSIC_GAIN = 0.12;
const LOOKAHEAD = 1; // seconds a bar is queued before it plays, so few notes wait at once
const GAP_MS = { min: 10000, max: 20000 }; // quiet between plays of the song
// Touch only counts as a gesture on pointerup, a mouse on pointerdown.
const GESTURES = ["pointerdown", "pointerup", "keydown"];
const IDLE_SUSPEND_MS = 3000; // lets the audio thread sleep when nothing is playing

type Recipe = (ctx: AudioContext, out: AudioNode, buf: AudioBuffer, t: number, g: number) => void;

const CUES: Record<Cue, Recipe> = {
  step: (ctx, out, buf, t, g) =>
    noise(ctx, out, buf, { start: t, dur: 0.07, gain: 0.25 * g, filter: "lowpass", freq: 500 }),
  break: (ctx, out, buf, t, g) =>
    noise(ctx, out, buf, { start: t, dur: 0.18, gain: 0.6 * g, filter: "bandpass", freq: 900 }),
  place: (ctx, out, buf, t, g) => {
    tone(ctx, out, { freq: 190, endFreq: 120, start: t, dur: 0.09, gain: 0.5 * g });
    noise(ctx, out, buf, { start: t, dur: 0.04, gain: 0.3 * g, filter: "lowpass", freq: 1500 });
  },
  pop: (ctx, out, _buf, t, g) =>
    tone(ctx, out, { freq: 660, endFreq: 990, start: t, dur: 0.12, gain: 0.25 * g }),
  bonk: (ctx, out, buf, t, g) => {
    tone(ctx, out, { freq: 320, endFreq: 200, start: t, dur: 0.08, gain: 0.4 * g });
    noise(ctx, out, buf, { start: t, dur: 0.05, gain: 0.25 * g, filter: "lowpass", freq: 800 });
  },
  hurt: (ctx, out, _buf, t, g) =>
    tone(ctx, out, { freq: 260, endFreq: 140, start: t, dur: 0.2, gain: 0.45 * g }),
};

/** The tab is hidden or another window has focus. */
function away(): boolean {
  return document.hidden || !document.hasFocus();
}

/** Synthesized sound effects and gentle music; audio only starts after the first tap or key. */
export class GameAudio {
  private ctx?: AudioContext;
  private sfx?: GainNode;
  private music?: GainNode;
  private noise?: AudioBuffer;
  private nextBar?: number;
  private run?: GainNode; // this play of the song, so stopping silences its queued notes
  private idle?: number;

  constructor(
    private soundOn: boolean,
    private musicOn: boolean,
  ) {
    for (const type of GESTURES) addEventListener(type, this.unlock, true);
    document.addEventListener("visibilitychange", () => this.sync());
    for (const type of ["focus", "blur"]) addEventListener(type, () => this.sync());
  }

  set(sound: boolean, music: boolean): void {
    this.soundOn = sound;
    this.musicOn = music;
    this.sync();
  }

  /** Plays a cue at `gain` (0..1), e.g. quieter for things far away. */
  play(cue: Cue, gain = 1): void {
    if (!this.soundOn || away() || !this.ctx || !this.sfx || !this.noise || gain <= 0) return;
    this.wake();
    CUES[cue](this.ctx, this.sfx, this.noise, this.ctx.currentTime, gain);
  }

  /** Plays a note block's MIDI note, like a soft xylophone. */
  playNote(midi: number, gain = 1): void {
    if (!this.soundOn || away() || !this.ctx || !this.sfx || gain <= 0) return;
    this.wake();
    const t = { freq: midiHz(midi), start: this.ctx.currentTime, dur: 0.6, gain: 0.35 * gain };
    tone(this.ctx, this.sfx, { ...t, type: "triangle" });
  }

  /** Browsers only allow audio after a user gesture, so listening stops once it is running. */
  private readonly unlock = (): void => {
    this.ctx ??= this.createContext();
    this.ctx
      .resume()
      .then(() => {
        for (const type of GESTURES) removeEventListener(type, this.unlock, true);
        this.sync();
      })
      .catch(() => undefined);
  };

  private createContext(): AudioContext {
    const ctx = new AudioContext();
    this.noise = noiseBuffer(ctx);
    this.sfx = ctx.createGain();
    this.music = ctx.createGain();
    this.music.gain.value = 1;
    this.sfx.connect(ctx.destination);
    this.music.connect(echo(ctx, ctx.destination));
    return ctx;
  }

  /** Starts or stops music and lets the context sleep whenever nothing needs it; silent while away. */
  private sync(): void {
    const ctx = this.ctx;
    if (!ctx) return;
    const playing = this.musicOn && !away();
    if (playing && this.nextBar === undefined) this.playBar(0, 0, ctx.currentTime + 0.1);
    if (!playing) this.stopSong(ctx);
    if (playing) this.wake();
    else if (away()) void ctx.suspend();
    else this.sleepSoon();
  }

  /** Queues bar `bar` of section `i` at `start`, then the next bar just before this one ends. */
  private playBar(i: number, bar: number, start: number): void {
    const { ctx, music } = this;
    if (!ctx || !music) return;
    this.wake();
    if (i === 0 && bar === 0) {
      this.run = ctx.createGain();
      this.run.gain.value = MUSIC_GAIN;
      this.run.connect(music);
    }
    const section = SONG[i];
    if (!section || !this.run) {
      const gap = GAP_MS.min + Math.random() * (GAP_MS.max - GAP_MS.min);
      this.nextBar = window.setTimeout(() => this.playBar(0, 0, ctx.currentTime + 0.1), gap);
      return;
    }
    if (bar >= section.bars) return this.playBar(i + 1, 0, start);
    start = Math.max(start, ctx.currentTime + 0.05); // a throttled timer may run late
    this.bar(ctx, this.run, section, bar, start);
    const end = start + BAR_BEATS * BEAT;
    const wait = (end - ctx.currentTime - LOOKAHEAD) * 1000;
    this.nextBar = window.setTimeout(() => this.playBar(i, bar + 1, end), wait);
  }

  /** Fades out whatever is queued and forgets the song's place. */
  private stopSong(ctx: AudioContext): void {
    clearTimeout(this.nextBar);
    this.nextBar = undefined;
    const run = this.run;
    this.run = undefined;
    run?.gain.setTargetAtTime(0, ctx.currentTime, 0.2);
    window.setTimeout(() => run?.disconnect(), 2000);
  }

  /** One bar of a section: its chord, then the tune and, in choruses, the arpeggio. */
  private bar(ctx: AudioContext, out: AudioNode, s: Section, bar: number, start: number): void {
    const at = (beat: number) => start + beat * BEAT;
    this.chord(ctx, out, bar * BAR_BEATS, start, s.level);
    const { tune, arpeggio } = barNotes(s, bar);
    for (const n of tune)
      voice(ctx, out, {
        freq: midiHz(n.midi),
        start: at(n.beat),
        dur: n.beats * BEAT,
        gain: 0.55 * s.level,
        attack: 0.12,
        release: 0.9,
        type: "triangle",
      });
    for (const n of arpeggio)
      voice(ctx, out, {
        freq: midiHz(n.midi),
        start: at(n.beat),
        dur: n.beats * BEAT,
        gain: 0.16 * s.level,
        attack: 0.03,
        release: 0.7,
      });
  }

  /** One bar of the progression: a bass note under a soft pad, crossfading into the next bar. */
  private chord(
    ctx: AudioContext,
    out: AudioNode,
    beat: number,
    start: number,
    level: number,
  ): void {
    const { bass, voicing } = chordAt(beat);
    const bar = { start, dur: BAR_BEATS * BEAT, attack: 1.2, release: 1.6 };
    voice(ctx, out, { ...bar, freq: midiHz(bass), gain: 0.45 * level });
    for (const note of voicing)
      voice(ctx, out, { ...bar, freq: midiHz(note), gain: PAD_GAIN * level });
  }

  private wake(): void {
    clearTimeout(this.idle);
    this.idle = undefined;
    if (this.ctx?.state === "suspended") void this.ctx.resume();
    if (!this.musicOn) this.sleepSoon();
  }

  private sleepSoon(): void {
    clearTimeout(this.idle);
    this.idle = window.setTimeout(() => void this.ctx?.suspend(), IDLE_SUSPEND_MS);
  }
}
