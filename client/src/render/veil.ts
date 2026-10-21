/** A cover that fades the world in and out over a fixed time, e.g. while chunks rebuild. */
export class Veil {
  private from = 0;
  private to = 0;
  private since = 0;

  constructor(private readonly ms = 200) {}

  close(now: number): void {
    this.head(now, 1);
  }

  open(now: number): void {
    this.head(now, 0);
  }

  /** How covered the world is at time now, 0 (clear) to 1. */
  amount(now: number): number {
    const t = Math.min(1, (now - this.since) / this.ms);
    return this.from + (this.to - this.from) * t;
  }

  /** Whether it has finished closing. */
  closed(now: number): boolean {
    return this.to === 1 && this.amount(now) === 1;
  }

  private head(now: number, to: number): void {
    this.from = this.amount(now);
    this.to = to;
    this.since = now;
  }
}
