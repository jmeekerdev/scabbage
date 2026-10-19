interface Cell {
  x: number;
  y: number;
  z: number;
}

/** Times digging one block in survival; looking away or letting go starts over. */
export class Miner {
  private target?: Cell;
  private elapsedMs = 0;

  /** Digs for dt seconds and returns progress from 0 to 1; at 1 the block breaks. */
  dig(dt: number, target: Cell, mineMs: number): number {
    if (!this.target || !same(this.target, target)) this.stop();
    this.target = target;
    this.elapsedMs += dt * 1000;
    if (this.elapsedMs < mineMs) return this.elapsedMs / mineMs;
    this.stop();
    return 1;
  }

  stop(): void {
    this.target = undefined;
    this.elapsedMs = 0;
  }
}

const same = (a: Cell, b: Cell) => a.x === b.x && a.y === b.y && a.z === b.z;
