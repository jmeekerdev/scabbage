import { describe, expect, it } from "vitest";
import { Harm } from "../pb/scabbage/v1/net_pb";
import { faintText, hearts, MAX_HEALTH } from "./health";

describe("hearts", () => {
  it("shows ten full hearts at full health", () => {
    expect(hearts(MAX_HEALTH)).toEqual(Array(10).fill("full"));
  });

  it("shows half a heart for an odd point", () => {
    expect(hearts(5)).toEqual(["full", "full", "half", ...Array(7).fill("empty")]);
  });

  it("shows empty hearts when fainted", () => {
    expect(hearts(0)).toEqual(Array(10).fill("empty"));
  });
});

describe("faintText", () => {
  it("names the cause kindly", () => {
    expect(faintText(Harm.FALL)).toMatch(/fell/i);
    expect(faintText(Harm.PLAYER)).toMatch(/tagged/i);
    expect(faintText(Harm.UNSPECIFIED)).not.toBe("");
  });
});
