export interface Stack {
  block: number;
  count: number;
}

const EMPTY: Stack = { block: 0, count: 0 };

/** What a survival player carries, as the server last sent it; hotbar slots come first. */
export class Inventory {
  slots: Stack[] = [];
  onChange = () => {};

  /** Takes the server's copy; true when it holds more items than before, i.e. something was picked up. */
  set(slots: readonly Stack[]): boolean {
    const gained = total(slots) > total(this.slots);
    this.slots = slots.map((s) => (s.count > 0 ? { block: s.block, count: s.count } : EMPTY));
    this.onChange();
    return gained;
  }

  /** Uses up one block from a slot straight away, ahead of the server's update. */
  spend(slot: number): boolean {
    const s = this.slots[slot];
    if (!s || s.count === 0) return false;
    this.slots[slot] = s.count > 1 ? { block: s.block, count: s.count - 1 } : EMPTY;
    this.onChange();
    return true;
  }
}

const total = (slots: readonly Stack[]) => slots.reduce((n, s) => n + s.count, 0);

/** Moving items by clicking one slot, then the slot to put them in. */
export class SlotMover {
  held: number | undefined;

  /** Returns [from, to] once a second, different slot is clicked. */
  click(slot: number): [number, number] | undefined {
    const from = this.held;
    this.held = from === undefined ? slot : undefined;
    return from === undefined || from === slot ? undefined : [from, slot];
  }
}
