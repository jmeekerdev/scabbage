/** Tracks round-trip time as an exponentially weighted moving average. */
export class LatencyTracker {
  rttMs: number | undefined;

  constructor(private readonly weight = 0.2) {}

  record(sentMs: number, receivedMs: number): void {
    const sample = receivedMs - sentMs;
    if (sample < 0) return;
    this.rttMs =
      this.rttMs === undefined ? sample : this.rttMs + this.weight * (sample - this.rttMs);
  }
}
