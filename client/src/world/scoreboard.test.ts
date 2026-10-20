import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { ScoreSchema } from "../pb/scabbage/v1/net_pb";
import { clock, scoreRows, teamTotals } from "./scoreboard";

const score = (name: string, team = "", points = 0, extra = {}) =>
  create(ScoreSchema, { name, team, points, ...extra });

describe("clock", () => {
  it("shows minutes and seconds", () => {
    expect(clock(245)).toBe("4:05");
    expect(clock(9)).toBe("0:09");
    expect(clock(-3)).toBe("0:00");
  });
});

describe("scoreRows", () => {
  it("lists players with points, flags and who is out", () => {
    const rows = scoreRows([
      score("Ann", "red", 2, { flag: "blue" }),
      score("Bo", "", 0, { out: true }),
    ]);
    expect(rows).toEqual([
      { text: "Ann 2 ⚑", team: "red", out: false },
      { text: "Bo 0", team: "", out: true },
    ]);
  });
});

describe("teamTotals", () => {
  it("adds up each team's points in the order teams appear", () => {
    expect(teamTotals([score("A", "red", 1), score("B", "blue", 2), score("C", "red", 3)])).toBe(
      "red 4 · blue 2",
    );
  });

  it("is empty without teams", () => {
    expect(teamTotals([score("A")])).toBe("");
  });
});
