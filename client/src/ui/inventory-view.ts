import { SlotMover, type Inventory } from "../player/inventory";
import { HOTBAR_SLOTS } from "../player/hotbar";
import { el } from "./dom";

/** The whole survival inventory; click a slot, then another, to move or swap items. */
export class InventoryView {
  readonly node = el("div", { class: "picker inventory" });
  private readonly mover = new SlotMover();
  private icon: (block: number) => HTMLCanvasElement = () => el("canvas");
  private name: (block: number) => string = String;

  constructor(
    private readonly inv: Inventory,
    private readonly move: (from: number, to: number) => void,
  ) {}

  setBlocks(icon: (block: number) => HTMLCanvasElement, name: (block: number) => string): void {
    [this.icon, this.name] = [icon, name];
    this.draw();
  }

  draw(): void {
    this.node.replaceChildren(
      el("h1", {}, "Backpack"),
      el("p", {}, "Tap an item, then the slot to put it in. The top row is your hotbar."),
      el("div", { class: "grid" }, ...this.inv.slots.map((s, i) => this.slot(s.block, s.count, i))),
    );
  }

  private slot(block: number, count: number, i: number): HTMLButtonElement {
    const classes = ["slot", i < HOTBAR_SLOTS && "hotbar", this.mover.held === i && "held"];
    return el(
      "button",
      {
        class: classes.filter(Boolean).join(" "),
        title: count ? this.name(block) : "empty",
        onclick: () => this.click(i),
      },
      count ? this.icon(block) : undefined,
      count ? el("span", {}, String(count)) : undefined,
    );
  }

  private click(i: number): void {
    const move = this.mover.click(i);
    if (move) this.move(...move);
    this.draw();
  }
}
