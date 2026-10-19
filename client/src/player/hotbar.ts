const STARTER_BLOCKS = [
  "grass",
  "dirt",
  "stone",
  "cobblestone",
  "planks",
  "log",
  "brick",
  "glass",
  "leaves",
];

/** IDs of the starter blocks the server knows, in hotbar order. */
export function hotbarBlocks(types: readonly { id: number; name: string }[]): number[] {
  return STARTER_BLOCKS.flatMap((name) => types.find((t) => t.name === name)?.id ?? []);
}

/** The row of blocks the player builds with, one of them selected. */
export class Hotbar {
  selected = 0;
  onChange = () => {};

  constructor(readonly slots: number[]) {}

  current(): number {
    return this.slots[this.selected] ?? 0; // air: nothing to place
  }

  select(i: number): void {
    if (i < 0 || i >= this.slots.length) return;
    this.selected = i;
    this.onChange();
  }

  scroll(steps: number): void {
    const n = this.slots.length;
    this.select((((this.selected + steps) % n) + n) % n);
  }

  /** Selects the slot holding `block`, or swaps it into the selected slot. */
  pick(block: number): void {
    const i = this.slots.indexOf(block);
    if (i < 0) this.slots[this.selected] = block;
    this.select(i < 0 ? this.selected : i);
  }
}
