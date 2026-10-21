import type { Dialogue } from "../pb/scabbage/v1/net_pb";
import { el } from "./dom";

/** What a script's character says, with a button for each answer it waits for. */
export class DialogueView {
  private readonly speaker = el("h1", {});
  private readonly text = el("p", { class: "line" });
  private readonly choices = el("div", { class: "choices" });
  private readonly dialog = el(
    "dialog",
    { class: "card dialogue" },
    this.speaker,
    this.text,
    this.choices,
  );

  constructor(private readonly onAnswer: (dialogue: number, choice: number) => void) {
    document.body.append(this.dialog);
    this.dialog.addEventListener("keydown", (e) => this.key(e));
  }

  show(d: Dialogue): void {
    this.speaker.textContent = `💬 ${d.speaker}`;
    this.text.textContent = d.text;
    const buttons = d.choices.length
      ? d.choices.map((c, i) => this.button(`${i + 1}. ${c}`, () => this.onAnswer(d.id, i)))
      : [this.button("👋 Bye", () => undefined)];
    this.choices.replaceChildren(...buttons);
    if (document.pointerLockElement) document.exitPointerLock();
    if (!this.dialog.open) this.dialog.showModal();
    buttons[0].focus();
  }

  private button(label: string, onclick: () => void): HTMLButtonElement {
    return el(
      "button",
      {
        type: "button",
        onclick: () => {
          this.dialog.close();
          onclick();
        },
      },
      label,
    );
  }

  /** Number keys pick answers; no key reaches the game while talking. */
  private key(e: KeyboardEvent): void {
    e.stopPropagation();
    const pick = /^Digit[1-4]$/.test(e.code) ? Number(e.code.slice(5)) - 1 : -1;
    const button = this.choices.children[pick];
    if (button instanceof HTMLButtonElement) button.click();
  }
}
