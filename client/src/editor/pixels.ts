import type { Rect } from "../render/skin";

export type RGBA = readonly [number, number, number, number];
export const CLEAR: RGBA = [0, 0, 0, 0];

/** Bright, friendly colors first, then skin tones, browns and grays. */
export const SWATCHES = [
  "#e53935", "#fb8c00", "#fdd835", "#43a047", "#1e88e5", "#3949ab", "#8e24aa", "#ec407a",
  "#80deea", "#a5d6a7", "#f8bbd0", "#fff59d", "#ffe0bd", "#f0c8a0", "#c68642", "#8d5524",
  "#5d4037", "#3e2723", "#ffffff", "#bdbdbd", "#757575", "#424242", "#212121", "#000000",
]; // prettier-ignore

/** An RGBA picture, rows top-down, edited one pixel at a time. */
export class PixelImage {
  readonly data: Uint8Array;

  constructor(
    readonly width: number,
    readonly height: number,
    data?: Uint8Array,
  ) {
    this.data = data ? data.slice() : new Uint8Array(width * height * 4);
  }

  private index(x: number, y: number): number {
    const inside = x >= 0 && y >= 0 && x < this.width && y < this.height;
    return inside ? (y * this.width + x) * 4 : -1;
  }

  get(x: number, y: number): RGBA {
    const i = this.index(x, y);
    if (i < 0) return CLEAR;
    const d = this.data;
    return [d[i] ?? 0, d[i + 1] ?? 0, d[i + 2] ?? 0, d[i + 3] ?? 0];
  }

  set(x: number, y: number, c: RGBA): boolean {
    const i = this.index(x, y);
    if (i < 0 || same(this.get(x, y), c)) return false;
    this.data.set(c, i);
    return true;
  }

  /** Repaints the area of one color touching x,y, staying inside `bounds`. */
  fill(x: number, y: number, c: RGBA, bounds: Rect): boolean {
    const from = this.get(x, y);
    if (same(from, c)) return false;
    const inBounds = (px: number, py: number) =>
      px >= bounds.x && py >= bounds.y && px < bounds.x + bounds.w && py < bounds.y + bounds.h;
    const todo: [number, number][] = [[x, y]];
    for (let p = todo.pop(); p; p = todo.pop()) {
      const [px, py] = p;
      if (!inBounds(px, py) || !same(this.get(px, py), from)) continue;
      this.set(px, py, c);
      todo.push([px + 1, py], [px - 1, py], [px, py + 1], [px, py - 1]);
    }
    return true;
  }
}

const same = (a: RGBA, b: RGBA) => a.every((v, i) => v === b[i]);

/** Snapshots of a picture so mistakes can be undone stroke by stroke. */
export class Undo {
  private readonly steps: Uint8Array[] = [];

  constructor(
    private readonly image: PixelImage,
    private readonly limit = 30,
  ) {}

  /** Call before each stroke. */
  save(): void {
    this.steps.push(this.image.data.slice());
    if (this.steps.length > this.limit) this.steps.shift();
  }

  back(): boolean {
    const step = this.steps.pop();
    if (step) this.image.data.set(step);
    return step !== undefined;
  }
}

export function parseHex(hex: string): RGBA {
  const n = parseInt(hex.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255, 255];
}

export function hexColor([r, g, b]: RGBA): string {
  return "#" + [r, g, b].map((v) => v.toString(16).padStart(2, "0")).join("");
}
