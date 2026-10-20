import { BAR_BEATS, chordAt, midiHz } from "./chords";
import { arpeggioNotes, type Section, sectionNotes, SONG } from "./song";
import { echo, noise, noiseBuffer, tone, voice } from "./synth";

export type Cue = "step" | "break" | "place" | "pop" | "hurt" | "bonk";

const BEAT = 0.65; // seconds; slow and calm
const PAD_GAIN = 0.1; // per voice, under the melody
const MUSIC_GAIN = 0.12;
const LOOKAHEAD = 0.5; // seconds a section is queued before it plays
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

/** Synthesized sound effects and gentle music; audio only starts after the first tap or key. */
export class GameAudio {
  private ctx?: AudioContext;
  private sfx?: GainNode;
  private music?: GainNode;
  private noise?: AudioBuffer;
  private nextSection?: number;
  private run?: GainNode; // this play of the song, so stopping silences its queued notes
  private idle?: number;

  constructor(
    private soundOn: boolean,
    private musicOn: boolean,
  ) {
    for (const type of GESTURES) addEventListener(type, this.unlock, true);
    document.addEventListener("visibilitychange", () => this.sync());
  }

  set(sound: boolean, music: boolean): void {
    this.soundOn = sound;
    this.musicOn = music;
    this.sync();
  }

  /** Plays a cue at `gain` (0..1), e.g. quieter for things far away. */
  play(cue: Cue, gain = 1): void {
    if (!this.soundOn || !this.ctx || !this.sfx || !this.noise || gain <= 0) return;
    this.wake();
    CUES[cue](this.ctx, this.sfx, this.noise, this.ctx.currentTime, gain);
  }

  /** Plays a note block's MIDI note, like a soft xylophone. */
  playNote(midi: number, gain = 1): void {
    if (!this.soundOn || !this.ctx || !this.sfx || gain <= 0) return;
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

  /** Starts or stops music and lets the context sleep whenever nothing needs it. */
  private sync(): void {
    const ctx = this.ctx;
    if (!ctx) return;
    const playing = this.musicOn && !document.hidden;
    if (playing && this.nextSection === undefined) this.playSection(0, ctx.currentTime + 0.1);
    if (!playing) this.stopSong(ctx);
    if (playing) this.wake();
    else this.sleepSoon();
  }

  /** Queues section `i` of the song at `start`, then the next one just before this ends. */
  private playSection(i: number, start: number): void {
    const { ctx, music } = this;
    if (!ctx || !music) return;
    this.wake();
    if (i === 0) {
      this.run = ctx.createGain();
      this.run.gain.value = MUSIC_GAIN;
      this.run.connect(music);
    }
    const section = SONG[i];
    if (!section || !this.run) {
      const gap = GAP_MS.min + Math.random() * (GAP_MS.max - GAP_MS.min);
      this.nextSection = window.setTimeout(() => this.playSection(0, ctx.currentTime + 0.1), gap);
      return;
    }
    start = Math.max(start, ctx.currentTime + 0.05); // a throttled timer may run late
    this.section(ctx, this.run, section, start);
    const end = start + section.bars * BAR_BEATS * BEAT;
    const wait = (end - ctx.currentTime - LOOKAHEAD) * 1000;
    this.nextSection = window.setTimeout(() => this.playSection(i + 1, end), wait);
  }

  /** Fades out whatever is queued and forgets the song's place. */
  private stopSong(ctx: AudioContext): void {
    clearTimeout(this.nextSection);
    this.nextSection = undefined;
    const run = this.run;
    this.run = undefined;
    run?.gain.setTargetAtTime(0, ctx.currentTime, 0.2);
    window.setTimeout(() => run?.disconnect(), 2000);
  }

  /** One section: chords every bar, then the tune and, in choruses, the arpeggio. */
  private section(ctx: AudioContext, out: AudioNode, s: Section, start: number): void {
    const at = (beat: number) => start + beat * BEAT;
    for (let bar = 0; bar < s.bars; bar++)
      this.chord(ctx, out, bar * BAR_BEATS, at(bar * BAR_BEATS), s.level);
    for (const n of sectionNotes(s))
      voice(ctx, out, {
        freq: midiHz(n.midi),
        start: at(n.beat),
        dur: n.beats * BEAT,
        gain: 0.55 * s.level,
        attack: 0.12,
        release: 0.9,
        type: "triangle",
      });
    if (!s.arpeggio) return;
    for (const n of arpeggioNotes(s.bars))
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
