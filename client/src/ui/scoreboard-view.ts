import type { Scoreboard } from "../pb/scabbage/v1/net_pb";
import { clock, scoreRows, teamTotals } from "../world/scoreboard";
import { el } from "./dom";

/** The minigame round's board in the corner, counting its clock down between updates. */
export class ScoreboardView {
  private endsAt = 0;
  private timer?: number;
  private readonly clockEl = el("span", { class: "clock" });

  constructor(private readonly root: HTMLElement) {}

  show(b: Scoreboard): void {
    clearInterval(this.timer);
    this.root.hidden = !b.zone;
    if (!b.zone) return;
    const totals = teamTotals(b.scores);
    this.root.replaceChildren(
      el("div", { class: "title" }, `${b.zone}: ${b.status} `, this.clockEl),
      totals && el("div", { class: "totals" }, totals),
      ...scoreRows(b.scores).map((r) =>
        el("div", { class: r.out ? "out" : undefined, "data-team": r.team }, r.text),
      ),
    );
    this.endsAt = performance.now() + b.seconds * 1000;
    this.clockEl.hidden = b.seconds === 0;
    this.tick();
    if (b.seconds > 0) this.timer = window.setInterval(() => this.tick(), 1000);
  }

  hide(): void {
    clearInterval(this.timer);
    this.root.hidden = true;
  }

  private tick(): void {
    this.clockEl.textContent = clock((this.endsAt - performance.now()) / 1000);
  }
}
