import { describe, expect, it } from "vitest";
import {
  combineIntents,
  DoubleTap,
  HoldRepeat,
  Look,
  moveInputFromKeys,
  NO_BUILD,
  NO_INPUT,
  stickAxes,
} from "./input";

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
  it("knows while it is held", () => {
    const r = new HoldRepeat(250);
    expect(r.held).toBe(false);
    r.press(0);
    expect(r.held).toBe(true);
    r.release();
    expect(r.held).toBe(false);
  });

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

describe("Look.turnBy", () => {
  it("turns by radians, with the same signs as turn", () => {
    const l = new Look(0.01);
    l.turnBy(0.5, -0.25);
    expect(l.yaw).toBeCloseTo(-0.5);
    expect(l.pitch).toBeCloseTo(0.25);
  });
});

describe("stickAxes", () => {
  it("ignores small drift and reaches full speed at the edge", () => {
    expect(stickAxes(0.1, -0.05, 0.2)).toEqual([0, 0]);
    const [x, y] = stickAxes(1, 0, 0.2);
    expect(x).toBeCloseTo(1);
    expect(y).toBe(0);
  });

  it("ramps up smoothly from the dead zone", () => {
    const [x] = stickAxes(0.6, 0, 0.2);
    expect(x).toBeCloseTo(0.5);
  });

  it("keeps diagonals within the unit circle", () => {
    const [x, y] = stickAxes(1, 1, 0.2);
    expect(Math.hypot(x, y)).toBeCloseTo(1);
  });
});

describe("combineIntents", () => {
  const still = { move: NO_INPUT, build: NO_BUILD };

  it("adds movement up to full speed and ORs buttons", () => {
    const a = { move: { ...NO_INPUT, forward: 0.75, jump: true }, build: NO_BUILD };
    const b = {
      move: { ...NO_INPUT, forward: 0.5, right: -0.5 },
      build: { ...NO_BUILD, placing: true },
    };
    expect(combineIntents([a, b, still])).toEqual({
      move: { forward: 1, right: -0.5, jump: true, descend: false },
      build: { breaking: false, placing: true, picking: false, digging: false },
    });
  });

  it("is still with no sources", () => {
    expect(combineIntents([])).toEqual(still);
  });
});
