import { PRESETS, VIEW_DISTANCE, type Settings } from "./settings";

const WARMUP_MS = 10_000; // loading the world is slow everywhere
const WINDOW_MS = 5_000;
const PAUSE_MS = 500; // a longer gap means a hidden tab, not a slow device
const SLOW = 0.7; // fraction of the target frame rate that counts as keeping up

/** One step lighter: the preset first, then view distance, stopping just above the minimum. */
export function lighter(s: Settings): Settings | undefined {
  const i = PRESETS.indexOf(s.preset);
  if (i > 0) return { ...s, preset: PRESETS[i - 1] };
  if (s.viewDistance > VIEW_DISTANCE.min + 1) return { ...s, viewDistance: s.viewDistance - 1 };
  return undefined;
}

/** Watches drawn frames and suggests lighter settings while the device can't keep up. */
export class AutoQuality {
  private start = -Infinity;
  private last = -Infinity;
  private frames = 0;
  private warmup = true;

  /**
   * Call once per drawn frame, with steady false when the frame before it went undrawn because
   * nothing changed; returns new settings when it is time to lighten.
   */
  frame(nowMs: number, s: Settings, steady = true): Settings | undefined {
    if (!s.auto) return undefined;
    const paused = nowMs - this.last > PAUSE_MS;
    this.last = nowMs;
    if (paused || !steady) {
      this.restart(nowMs, paused || this.warmup);
      return undefined;
    }
    this.frames++;
    const elapsed = nowMs - this.start;
    if (elapsed < (this.warmup ? WARMUP_MS : WINDOW_MS)) return undefined;
    const fps = (this.frames * 1000) / elapsed;
    const wasWarmup = this.warmup;
    this.restart(nowMs, false);
    if (wasWarmup || fps >= (s.fpsCap || 60) * SLOW) return undefined;
    const next = lighter(s);
    if (next) this.restart(nowMs, true); // let the new settings settle before judging again
    return next;
  }

  private restart(nowMs: number, warmup: boolean): void {
    this.start = nowMs;
    this.frames = 0;
    this.warmup = warmup;
  }
}
