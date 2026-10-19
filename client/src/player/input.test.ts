import { describe, expect, it } from "vitest";
import { DoubleTap, HoldRepeat, Look, moveInputFromKeys } from "./input";

describe("moveInputFromKeys", () => {
  it("maps WASD, arrows, space and shift", () => {
    expect(moveInputFromKeys(new Set(["KeyW", "KeyD", "Space"]))).toEqual({
      forward: 1,
      right: 1,
      jump: true,
      descend: false,
    });
    expect(moveInputFromKeys(new Set(["ArrowDown", "ArrowLeft", "ShiftLeft"]))).toEqual({
      forward: -1,
      right: -1,
      jump: false,
      descend: true,
    });
  });

  it("cancels opposite keys", () => {
    expect(moveInputFromKeys(new Set(["KeyW", "KeyS"])).forward).toBe(0);
  });
});

describe("DoubleTap", () => {
  it("fires on a second press soon after the first, then starts over", () => {
    const t = new DoubleTap(300);
    expect(t.press(0)).toBe(false);
    expect(t.press(200)).toBe(true);
    expect(t.press(250)).toBe(false);
    expect(t.press(1000)).toBe(false);
    expect(t.press(1400)).toBe(false);
  });
});

describe("Look", () => {
  it("turns left and right, wrapping yaw", () => {
    const l = new Look(0.01);
    l.turn(-100, 0);
    expect(l.yaw).toBeCloseTo(1);
    l.turn(1000, 0);
    expect(l.yaw).toBeGreaterThan(-Math.PI);
    expect(l.yaw).toBeLessThanOrEqual(Math.PI);
  });

  it("stops just short of looking straight up or down", () => {
    const l = new Look(0.01);
    l.turn(0, -10000);
    expect(l.pitch).toBeLessThan(Math.PI / 2);
    expect(l.pitch).toBeGreaterThan(1.5);
    l.turn(0, 20000);
    expect(l.pitch).toBeGreaterThan(-Math.PI / 2);
  });
});

describe("HoldRepeat", () => {
  it("fires on press, then every interval while held, and stops on release", () => {
    const r = new HoldRepeat(250);
    expect(r.due(0)).toBe(false);
    r.press(100);
    expect(r.due(100)).toBe(true);
    expect(r.due(200)).toBe(false);
    expect(r.due(360)).toBe(true);
    expect(r.due(600)).toBe(true);
    r.release();
    expect(r.due(2000)).toBe(false);
  });
});
