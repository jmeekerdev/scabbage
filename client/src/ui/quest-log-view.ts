import { el } from "./dom";
import { questNews, type QuestEntry } from "./quest-news";

/** The player's quests, newest first: what to do next, or done. */
export class QuestLogView {
  readonly node = el("div", { class: "quests" });
  private quests: readonly QuestEntry[] = [];

  constructor() {
    this.show();
  }

  /** Shows the log as it was when we joined; nothing in it is news. */
  start(quests: readonly QuestEntry[]): void {
    this.quests = quests;
    this.show();
  }

  /** Shows a changed log and returns what to tell the player about it. */
  update(quests: readonly QuestEntry[]): string[] {
    const news = questNews(this.quests, quests);
    this.quests = quests;
    this.show();
    return news;
  }

  private show(): void {
    this.node.replaceChildren(
      el("h1", {}, "📜 Quests"),
      ...(this.quests.length
        ? this.quests.map((q) =>
            el(
              "div",
              { class: q.done ? "quest done" : "quest" },
              el("strong", {}, q.done ? `✔ ${q.title}` : q.title),
              q.done ? undefined : el("div", {}, q.step),
            ),
          )
        : [el("p", {}, "No quests yet. Talk to the characters you meet!")]),
    );
  }
}
