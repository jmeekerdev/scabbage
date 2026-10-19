import { describe, expect, it } from "vitest";
import { Emote } from "../pb/scabbage/v1/net_pb";
import { EMOTES, emojiFor, WheelAim } from "./emotes";

describe("EMOTES", () => {
  it("covers every emote in the protocol once", () => {
    const known = Object.values(Emote).filter((v) => typeof v === "number" && v !== 0);
    expect(EMOTES.map((e) => e.emote).sort()).toEqual(known.sort());
    expect(emojiFor(Emote.WAVE)).toBe("👋");
    expect(emojiFor(Emote.UNSPECIFIED)).toBeUndefined();
  });
});

describe("WheelAim", () => {
  it("picks nothing until the pointer moves far enough from the middle", () => {
    const aim = new WheelAim(8, 30);
    aim.move(0, -20);
    expect(aim.slot()).toBeUndefined();
  });

  it("numbers slots clockwise from the top", () => {
    const pick = (dx: number, dy: number) => {
      const aim = new WheelAim(8, 30);
      aim.move(dx, dy);
      return aim.slot();
    };
    expect(pick(0, -50)).toBe(0);
    expect(pick(50, 0)).toBe(2);
    expect(pick(0, 50)).toBe(4);
    expect(pick(-50, 0)).toBe(6);
    expect(pick(-40, -45)).toBe(7);
  });

  it("adds up movements and stays within reach so turning back is quick", () => {
    const aim = new WheelAim(8, 30);
    aim.move(500, 0);
    aim.move(-100, 0);
    expect(aim.slot()).toBe(6);
    aim.reset();
    expect(aim.slot()).toBeUndefined();
  });
});
