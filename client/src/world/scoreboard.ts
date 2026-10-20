import type { Score } from "../pb/scabbage/v1/net_pb";

export interface ScoreRow {
  text: string;
  team: string;
  out: boolean;
}

/** Seconds as m:ss. */
export function clock(seconds: number): string {
  const s = Math.max(0, Math.round(seconds));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

/** One row per player: their name, points and a flag if they carry one. */
export function scoreRows(scores: Score[]): ScoreRow[] {
  return scores.map((s) => ({
    text: `${s.name} ${s.points}${s.flag ? " ⚑" : ""}`,
    team: s.team,
    out: s.out,
  }));
}

/** Each team's points, e.g. "red 4 · blue 2"; empty when no one is on a team. */
export function teamTotals(scores: Score[]): string {
  const totals = new Map<string, number>();
  for (const s of scores) if (s.team) totals.set(s.team, (totals.get(s.team) ?? 0) + s.points);
  return [...totals].map(([team, points]) => `${team} ${points}`).join(" · ");
}
