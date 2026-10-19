import type { Stack } from "./inventory";

export interface Recipe {
  make: Stack;
  from: readonly Stack[];
}

/** Whether the slots hold every input; the server has the final word, e.g. when there's no room. */
export function canCraft(r: Recipe, slots: readonly Stack[]): boolean {
  return r.from.every((need) => carried(slots, need.block) >= need.count);
}

const carried = (slots: readonly Stack[], block: number) =>
  slots.reduce((n, s) => n + (s.block === block ? s.count : 0), 0);
