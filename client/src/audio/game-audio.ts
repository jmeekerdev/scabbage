import { melody } from "./melody";
import { noise, noiseBuffer, tone } from "./synth";

export type Cue = "step" | "break" | "place" | "pop";

const BEAT = 0.45; // seconds
const PHRASE_BEATS = 16;
const BASS = [130.81, 110, 98, 130.81]; // C3 A2 G2 C3, one per bar of four beats
const GAP_MS = { min: 6000, max: 16000 }; // quiet between phrases
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
};

/** Synthesized sound effects and gentle music; audio only starts after the first tap or key. */
export class GameAudio {
  private ctx?: AudioContext;
  private sfx?: GainNode;
  private music?: GainNode;
  private noise?: AudioBuffer;
  private nextPhrase?: number;
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
    this.music.gain.value = 0.12;
    this.sfx.connect(ctx.destination);
    this.music.connect(ctx.destination);
    return ctx;
  }

  /** Starts or stops music and lets the context sleep whenever nothing needs it. */
  private sync(): void {
    const ctx = this.ctx;
    if (!ctx) return;
    const playing = this.musicOn && !document.hidden;
    if (playing && this.nextPhrase === undefined) this.phrase();
    if (!playing) {
      clearTimeout(this.nextPhrase);
      this.nextPhrase = undefined;
      this.music?.gain.setTargetAtTime(0, ctx.currentTime, 0.2); // fades out notes already queued
    }
    if (playing) this.wake();
    else this.sleepSoon();
  }

  private phrase(): void {
    const { ctx, music } = this;
    if (!ctx || !music) return;
    this.wake();
    const start = ctx.currentTime + 0.1;
    music.gain.setTargetAtTime(0.12, ctx.currentTime, 0.1);
    for (const n of melody(Math.random, PHRASE_BEATS))
      tone(ctx, music, {
        freq: n.freq,
        start: start + n.beat * BEAT,
        dur: n.beats * BEAT,
        gain: 0.8,
        type: "triangle",
      });
    BASS.forEach((freq, bar) =>
      tone(ctx, music, { freq, start: start + bar * 4 * BEAT, dur: 4 * BEAT, gain: 0.6 }),
    );
    const gap = GAP_MS.min + Math.random() * (GAP_MS.max - GAP_MS.min);
    this.nextPhrase = window.setTimeout(() => this.phrase(), PHRASE_BEATS * BEAT * 1000 + gap);
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
