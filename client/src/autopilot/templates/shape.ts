/** What a template cell is made of: its main material, a floor, roof, window glass or air. */
export type Part = "main" | "floor" | "roof" | "window" | "air";

/** A template's own cells: x across its front, y up, z back from its front. */
export interface LocalCell {
  x: number;
  y: number;
  z: number;
  part: Part;
}

/** Width across the front, depth from front to back, and height, in blocks. */
export interface Size {
  w: number;
  d: number;
  h: number;
}

export interface Template {
  about: string;
  size: Size; // the size it's built when none is asked for
  min: Size;
  material: string; // the main material when none is asked for
  floor?: string;
  roof?: string;
  /** Why it can't be made this size, beyond the minimum, or undefined if it can. */
  check?(size: Size): string | undefined;
  cells(size: Size): LocalCell[];
}

/** Cells by position, where a later part replaces an earlier one, as a doorway cuts a wall. */
export class Cells {
  private readonly byKey = new Map<string, LocalCell>();

  set(x: number, y: number, z: number, part: Part): void {
    this.byKey.set(`${x},${y},${z}`, { x, y, z, part });
  }

  /** Sets every cell of the box from `lo` to `hi`, both included. */
  box(lo: [number, number, number], hi: [number, number, number], part: Part): void {
    for (let y = lo[1]; y <= hi[1]; y++)
      for (let z = lo[2]; z <= hi[2]; z++)
        for (let x = lo[0]; x <= hi[0]; x++) this.set(x, y, z, part);
  }

  list(): LocalCell[] {
    return [...this.byKey.values()];
  }
}
