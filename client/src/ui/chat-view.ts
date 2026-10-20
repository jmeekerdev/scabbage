import { colorCss } from "../join/rules";
import type { Chat } from "../pb/scabbage/v1/net_pb";
import { el } from "./dom";

const MAX_LINES = 50;
const MAX_CHARS = 200; // matches the server's chat.MaxRunes
const OPEN_KEYS = new Set(["Enter", "t", "T", "/"]);

/** The chat log, whose lines fade after a while, and the box to type in (Enter, T or /). */
export class ChatView {
  /** Off until the player is in the world. */
  enabled = false;
  private readonly log = el("div", { class: "log", "aria-live": "polite" });
  private readonly input = el("input", {
    maxlength: MAX_CHARS,
    autocomplete: "off",
    enterkeyhint: "send",
    placeholder: "Say something…",
    "aria-label": "Chat message",
  });
  private readonly form = el("form", { hidden: true }, this.input);

  constructor(
    private readonly root: HTMLElement,
    private readonly send: (text: string) => void,
    private readonly onClose: (sent: boolean) => void,
  ) {
    root.append(this.log, this.form);
    this.form.addEventListener("submit", (e) => {
      e.preventDefault();
      this.submit();
    });
    this.input.addEventListener("keydown", (e) => {
      if (e.key === "Escape") this.close(false);
      e.stopPropagation(); // typing never reaches the game controls
    });
    document.addEventListener("keydown", (e) => this.openKey(e));
  }

  get isOpen(): boolean {
    return !this.form.hidden;
  }

  open(prefill = ""): void {
    if (!this.enabled) return;
    if (document.pointerLockElement) document.exitPointerLock(); // iPhones have no pointer lock
    this.root.classList.add("open");
    this.form.hidden = false;
    this.input.value = prefill;
    this.input.focus();
    this.log.scrollTop = this.log.scrollHeight;
  }

  add(c: Chat): void {
    if (c.notice) return this.note(c.text);
    this.append(
      el(
        "div",
        { class: "line" },
        el("span", { class: "dot", style: `background: ${colorCss(c.color)}` }),
        el("strong", {}, c.name),
        " ",
        c.text,
      ),
    );
  }

  /** Shows a line from the game itself. */
  note(text: string): void {
    this.append(el("div", { class: "line notice" }, text));
  }

  private append(line: HTMLElement): void {
    this.log.append(line);
    while (this.log.childElementCount > MAX_LINES) this.log.firstElementChild?.remove();
    this.log.scrollTop = this.log.scrollHeight;
  }

  private submit(): void {
    const text = this.input.value.trim();
    if (text) this.send(text);
    this.input.value = "";
    this.close(text !== "");
  }

  private close(sent: boolean): void {
    this.root.classList.remove("open");
    this.form.hidden = true;
    this.input.blur();
    this.onClose(sent);
  }

  private openKey(e: KeyboardEvent): void {
    const typing = e.target instanceof HTMLElement && e.target.closest("input, select, textarea");
    if (this.isOpen || typing || e.repeat || !OPEN_KEYS.has(e.key)) return;
    if (!this.enabled || document.querySelector("dialog[open]")) return;
    e.preventDefault(); // keeps the key itself out of the box
    this.open(e.key === "/" ? "/" : "");
  }
}
