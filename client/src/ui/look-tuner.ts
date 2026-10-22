import { checkLook } from "../looks/check";
import type { Look } from "../looks/look";
import {
  colorControls,
  fromHex,
  lookFile,
  numberControls,
  setAt,
  toHex,
  type ColorControl,
  type NumberControl,
} from "../looks/tuning";
import { el } from "./dom";

/** A developer panel of sliders over a look's live numbers, with an export to a new look file. */
export class LookTuner {
  readonly node = el("details", { class: "look-tuner", open: true });
  private readonly rows = el("div", { class: "rows" });
  private readonly problems = el("ul", { class: "problems" });
  private look: Look;

  constructor(
    private readonly start: Look,
    private readonly show: (look: Look) => void,
  ) {
    this.look = start;
    const name = el("input", { type: "text", value: start.name, "aria-label": "Look name" });
    name.addEventListener("input", () => this.change("name", name.value));
    this.node.append(
      el("summary", {}, "Look tuner"),
      el(
        "div",
        { class: "actions" },
        name,
        this.button("Export", () => this.export()),
        this.button("Reset", () => this.reset(name)),
      ),
      this.problems,
      this.rows,
    );
    // Typing here mustn't reach the game's key handlers, like chat's open key.
    this.node.addEventListener("keydown", (e) => e.stopPropagation());
    this.render();
  }

  private button(label: string, onclick: () => void): HTMLButtonElement {
    return el("button", { type: "button", onclick }, label);
  }

  private render(): void {
    this.rows.replaceChildren(
      ...colorControls(this.look).map((c) => this.colorRow(c)),
      ...numberControls(this.look).map((c) => this.numberRow(c)),
    );
    this.check();
  }

  private numberRow(c: NumberControl): HTMLElement {
    const shown = el("output", {}, String(c.value));
    const props = { type: "range", min: c.min, max: c.max, step: c.step, value: c.value };
    const input = el("input", { ...props, "aria-label": c.path });
    input.addEventListener("input", () => {
      shown.textContent = input.value;
      this.change(c.path, Number(input.value));
    });
    return el("label", {}, el("span", {}, c.path), input, shown);
  }

  private colorRow(c: ColorControl): HTMLElement {
    const input = el("input", { type: "color", value: toHex(c.value), "aria-label": c.path });
    input.addEventListener("input", () => this.change(c.path, fromHex(input.value)));
    return el("label", {}, el("span", {}, c.path), input);
  }

  private change(path: string, value: unknown): void {
    this.look = setAt(this.look, path, value);
    this.show(this.look);
    this.check();
  }

  private check(): void {
    this.problems.replaceChildren(...checkLook(this.look).map((p) => el("li", {}, p)));
  }

  private reset(name: HTMLInputElement): void {
    this.look = this.start;
    name.value = this.start.name;
    this.show(this.look);
    this.render();
  }

  private export(): void {
    const { name, text } = lookFile(this.look);
    const url = URL.createObjectURL(new Blob([text], { type: "text/typescript" }));
    el("a", { href: url, download: name }).click();
    URL.revokeObjectURL(url);
  }
}
