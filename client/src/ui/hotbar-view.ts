import type { Hotbar } from "../player/hotbar";
import { el } from "./dom";

/** Draws the hotbar into `root`, redrawing whenever it changes. */
export function mountHotbar(
  root: HTMLElement,
  hotbar: Hotbar,
  name: (block: number) => string,
  icon: (name: string) => HTMLCanvasElement,
): void {
  const draw = () =>
    root.replaceChildren(
      ...hotbar.slots.map((block, i) =>
        el(
          "div",
          { class: i === hotbar.selected ? "slot selected" : "slot", title: name(block) },
          icon(name(block)),
          el("span", {}, String(i + 1)),
        ),
      ),
    );
  hotbar.onChange = draw;
  draw();
}
