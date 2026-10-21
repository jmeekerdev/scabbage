import { describe, expect, it } from "vitest";
import { questNews } from "./quest-news";

const q = (title: string, step = "", done = false) => ({ title, step, done });

describe("quest news", () => {
  it("announces new, moved-on and finished quests", () => {
    const before = [q("Lost sheep", "Find 3 wool"), q("Old well", "Look down the well")];
    const after = [
      q("Old well", "", true),
      q("Lost sheep", "Bring the wool to Ivy"),
      q("Bridge", "Ask Bo about the bridge"),
    ];
    expect(questNews(before, after)).toEqual([
      "🏆 Quest done: Old well",
      "📜 Lost sheep: Bring the wool to Ivy",
      "📜 New quest: Bridge. Ask Bo about the bridge",
    ]);
  });

  it("stays quiet when nothing changed", () => {
    const log = [q("Lost sheep", "Find 3 wool"), q("Old well", "", true)];
    expect(questNews(log, [...log])).toEqual([]);
  });

  it("names a quest finished before it was ever started", () => {
    expect(questNews([], [q("Secret cave", "", true)])).toEqual(["🏆 Quest done: Secret cave"]);
  });
});
