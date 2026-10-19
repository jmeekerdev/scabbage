import { GameMode, Harm, type Vitals } from "../pb/scabbage/v1/net_pb";
import { faintText, hearts } from "../player/health";
import { el } from "./dom";

const FAINT_NOTE_MS = 3500;

/** Survival hearts above the hotbar, a red flash when hurt and a short note after fainting. */
export class HealthView {
  private noteTimer?: number;

  constructor(
    private readonly hearts: HTMLElement,
    private readonly flash: HTMLElement,
    private readonly note: HTMLElement,
  ) {}

  show(v: Vitals): void {
    this.hearts.hidden = v.mode !== GameMode.SURVIVAL;
    this.hearts.replaceChildren(...hearts(v.health).map((h) => el("span", { class: h }, "♥")));
    if (v.hurt !== Harm.UNSPECIFIED) this.flashRed();
    if (v.health === 0) this.faint(v.hurt);
  }

  private flashRed(): void {
    this.flash.classList.remove("on");
    void this.flash.offsetWidth; // restarts the animation
    this.flash.classList.add("on");
  }

  private faint(cause: Harm): void {
    this.note.textContent = faintText(cause);
    this.note.hidden = false;
    clearTimeout(this.noteTimer);
    this.noteTimer = window.setTimeout(() => (this.note.hidden = true), FAINT_NOTE_MS);
  }
}
