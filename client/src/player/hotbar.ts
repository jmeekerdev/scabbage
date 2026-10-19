import type { Stack } from "./inventory";

export const HOTBAR_SLOTS = 9;

const STARTER_BLOCKS = [
  "grass",
  "dirt",
  "stone",
  "cobblestone",
  "planks",
  "log",
  "brick",
  "glass",
  "torch",
];

/** IDs of the starter blocks the server knows, in hotbar order. */
export function hotbarBlocks(types: readonly { id: number; name: string }[]): number[] {
  return STARTER_BLOCKS.flatMap((name) => types.find((t) => t.name === name)?.id ?? []);
}

/** The row of blocks the player builds with, one of them selected. */
export class Hotbar {
  selected = 0;
  onChange = () => {};
  counts: number[] | undefined; // survival: how many of each slot's block are carried
  private creative?: number[]; // put aside while carrying an inventory

  constructor(public slots: number[]) {}

  /** Shows a survival inventory's first slots, or the creative blocks again when undefined. */
  carry(stacks: readonly Stack[] | undefined): void {
    if (stacks) {
      this.creative ??= this.slots;
      this.slots = stacks.slice(0, HOTBAR_SLOTS).map((s) => (s.count > 0 ? s.block : 0));
      this.counts = stacks.slice(0, HOTBAR_SLOTS).map((s) => s.count);
    } else if (this.creative) {
      [this.slots, this.creative, this.counts] = [this.creative, undefined, undefined];
    }
    this.select(Math.min(this.selected, this.slots.length - 1));
  }

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

  /** Selects the slot holding `block`, or in creative swaps it into the selected slot. */
  pick(block: number): void {
    const i = this.slots.indexOf(block);
    if (i < 0 && this.counts) return;
    if (i < 0) this.slots[this.selected] = block;
    this.select(i < 0 ? this.selected : i);
  }
}
