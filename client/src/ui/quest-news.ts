export interface QuestEntry {
  title: string;
  step: string;
  done: boolean;
}

/** What to tell the player about how their quest log changed. */
export function questNews(before: readonly QuestEntry[], after: readonly QuestEntry[]): string[] {
  const was = new Map(before.map((q) => [q.title, q]));
  const news: string[] = [];
  for (const q of after) {
    const old = was.get(q.title);
    if (q.done) {
      if (!old?.done) news.push(`🏆 Quest done: ${q.title}`);
    } else if (!old) news.push(`📜 New quest: ${q.title}. ${q.step}`);
    else if (old.step !== q.step || old.done) news.push(`📜 ${q.title}: ${q.step}`);
  }
  return news;
}
