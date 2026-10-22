import { swell, type SwellSpec } from "./envelope";

/** Tiny sound makers built from oscillators and noise, so the game ships no audio files. */

export interface ToneSpec {
  freq: number;
  start: number; // AudioContext seconds
  dur: number; // seconds
  gain: number;
  type?: OscillatorType;
  endFreq?: number; // slides the pitch here by the end
}

/** A note with a quick attack and a smooth fade so it never clicks. */
export function tone(ctx: AudioContext, out: AudioNode, t: ToneSpec): void {
  const osc = ctx.createOscillator();
  const env = ctx.createGain();
  osc.type = t.type ?? "sine";
  osc.frequency.setValueAtTime(t.freq, t.start);
  if (t.endFreq) osc.frequency.exponentialRampToValueAtTime(t.endFreq, t.start + t.dur);
  env.gain.setValueAtTime(0, t.start);
  env.gain.linearRampToValueAtTime(t.gain, t.start + Math.min(0.02, t.dur / 4));
  env.gain.exponentialRampToValueAtTime(0.0001, t.start + t.dur);
  osc.connect(env).connect(out);
  osc.start(t.start);
  osc.stop(t.start + t.dur + 0.05);
}

export interface VoiceSpec extends SwellSpec {
  freq: number;
  type?: OscillatorType;
}

/** A sustained music note that swells in and fades slowly, overlapping its neighbors. */
export function voice(ctx: AudioContext, out: AudioNode, v: VoiceSpec): void {
  const osc = ctx.createOscillator();
  const env = ctx.createGain();
  osc.type = v.type ?? "sine";
  osc.frequency.value = v.freq;
  const [first, ...rest] = swell(v);
  env.gain.setValueAtTime(first![1], first![0]);
  for (const [t, g] of rest) env.gain.linearRampToValueAtTime(g, t);
  osc.connect(env).connect(out);
  osc.start(v.start);
  osc.stop(v.start + v.dur + v.release + 0.05);
}

/** A soft echo that blurs notes together, built once per music bus. */
export function echo(ctx: AudioContext, out: AudioNode): AudioNode {
  const input = ctx.createGain();
  const delay = ctx.createDelay(1);
  const feedback = ctx.createGain();
  const damp = ctx.createBiquadFilter();
  delay.delayTime.value = 0.42;
  feedback.gain.value = 0.38;
  damp.type = "lowpass";
  damp.frequency.value = 1400;
  input.connect(out);
  input.connect(delay).connect(damp).connect(feedback).connect(delay);
  damp.connect(out);
  return input;
}

export interface NoiseSpec {
  start: number;
  dur: number;
  gain: number;
  filter: BiquadFilterType;
  freq: number;
}

/** A filtered burst of noise: thuds, crunches and scuffs. */
export function noise(ctx: AudioContext, out: AudioNode, buffer: AudioBuffer, n: NoiseSpec): void {
  const src = ctx.createBufferSource();
  const filter = ctx.createBiquadFilter();
  const env = ctx.createGain();
  src.buffer = buffer;
  filter.type = n.filter;
  filter.frequency.value = n.freq;
  env.gain.setValueAtTime(n.gain, n.start);
  env.gain.exponentialRampToValueAtTime(0.0001, n.start + n.dur);
  src.connect(filter).connect(env).connect(out);
  src.start(n.start, Math.random() * (buffer.duration - n.dur));
  src.stop(n.start + n.dur);
}

/** Half a second of white noise, made once and reused by every burst. */
export function noiseBuffer(ctx: AudioContext): AudioBuffer {
  const buffer = ctx.createBuffer(1, ctx.sampleRate / 2, ctx.sampleRate);
  const data = buffer.getChannelData(0);
  for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
  return buffer;
}
