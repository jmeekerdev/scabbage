import type { Emote } from "../pb/scabbage/v1/net_pb";
import { EMOTES, WheelAim } from "../social/emotes";
import { el } from "./dom";

const MIN_AIM_PX = 30; // mouse travel before a slot lights up

/** A ring of emotes: hold E and point with the mouse, or tap one. */
export class EmoteWheel {
  private readonly aim = new WheelAim(EMOTES.length, MIN_AIM_PX);
  private readonly slots: HTMLButtonElement[];

  constructor(
    private readonly root: HTMLElement,
    private readonly onPick: (e: Emote) => void,
  ) {
    this.slots = EMOTES.map(({ emote, emoji, label }, i) =>
      el(
        "button",
        {
          "aria-label": label,
          style: `--turn: ${i / EMOTES.length}turn`,
          onclick: () => this.pick(emote),
        },
        emoji,
      ),
    );
    root.append(el("div", { class: "ring" }, ...this.slots));
    root.addEventListener("click", (e) => e.target === root && this.close());
  }

  get isOpen(): boolean {
    return !this.root.hidden;
  }

  open(): void {
    this.aim.reset();
    this.highlight();
    this.root.hidden = false;
  }

  /** Points from the middle by a relative movement, e.g. a locked mouse. */
  move(dx: number, dy: number): void {
    this.aim.move(dx, dy);
    this.highlight();
  }

  /** Sends whatever is pointed at, if anything, and closes. */
  release(): void {
    const slot = this.aim.slot();
    if (slot === undefined) this.close();
    else this.pick(EMOTES[slot].emote);
  }

  close(): void {
    this.root.hidden = true;
  }

  private pick(e: Emote): void {
    this.close();
    this.onPick(e);
  }

  private highlight(): void {
    const slot = this.aim.slot();
    this.slots.forEach((b, i) => b.classList.toggle("aimed", i === slot));
  }
}

/** Briefly shows your own emote in the middle of the screen. */
export function popEmoji(root: HTMLElement, emoji: string): void {
  const pop = el("div", { class: "emote-pop" }, emoji);
  pop.addEventListener("animationend", () => pop.remove());
  root.append(pop);
}
