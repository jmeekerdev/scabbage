import { el } from "./dom";

export interface PickableBlock {
  id: number;
  name: string;
  item?: number; // set when players hold another block for this one
  turned?: boolean; // its item turned another way, which may be placed as it is
}

// Breaking-only or invisible blocks never go in the hotbar.
const UNPLACEABLE = new Set(["air", "bedrock"]);

/** Blocks players may hold; turned stairs and powered logic blocks are held as another block. */
export function placeable<T extends PickableBlock>(blocks: readonly T[]): T[] {
  return blocks.filter((b) => !UNPLACEABLE.has(b.name) && !b.item);
}

/** Blocks that may be placed as they are, like held blocks plus stairs turned each way. */
export function buildable<T extends PickableBlock>(blocks: readonly T[]): T[] {
  return blocks.filter((b) => !UNPLACEABLE.has(b.name) && (!b.item || b.turned));
}

/** Every block that can be placed, newest custom blocks last; picking one puts it in the hotbar. */
export class BlockPicker {
  readonly node = el("div", { class: "picker" });

  constructor(private readonly pick: (block: number) => void) {}

  set(blocks: readonly PickableBlock[], icon: (block: number) => HTMLCanvasElement): void {
    const choices = placeable(blocks);
    this.node.replaceChildren(
      el("h1", {}, "Blocks"),
      el(
        "div",
        { class: "grid" },
        ...choices.map((b) =>
          el(
            "button",
            { class: "slot", title: b.name, onclick: () => this.pick(b.id) },
            icon(b.id),
            el("span", {}, b.name),
          ),
        ),
      ),
    );
  }
}
