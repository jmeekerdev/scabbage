import { describe, expect, it } from "vitest";
import { TouchSticks } from "./touch-sticks";

/** A 1000 px wide screen with a 50 px joystick. */
const sticks = () => new TouchSticks(1000, 50);

describe("TouchSticks", () => {
  it("walks with a joystick that starts wherever the left thumb lands", () => {
    const t = sticks();
    t.down(1, 100, 500);
    t.move(1, 100, 450);
    expect(t.walk()).toEqual({ forward: 1, right: 0 });
    expect(t.joystick()).toEqual({ x: 100, y: 500, dx: 0, dy: -50 });
  });

  it("stops walking when the thumb lifts", () => {
    const t = sticks();
    t.down(1, 100, 500);
    t.move(1, 130, 500);
    t.up(1);
    expect(t.walk()).toEqual({ forward: 0, right: 0 });
    expect(t.joystick()).toBeUndefined();
  });

  it("caps the knob at the joystick edge", () => {
    const t = sticks();
    t.down(1, 100, 500);
    t.move(1, 400, 500);
    expect(t.joystick()?.dx).toBe(50);
    expect(t.walk().right).toBeCloseTo(1);
  });

  it("looks by dragging on the right, reporting each drag once", () => {
    const t = sticks();
    t.down(2, 800, 300);
    t.move(2, 790, 310);
    t.move(2, 780, 330);
    expect(t.takeLook()).toEqual([-20, 30]);
    expect(t.takeLook()).toEqual([0, 0]);
  });

  it("tracks both thumbs at once", () => {
    const t = sticks();
    t.down(1, 100, 500);
    t.down(2, 800, 300);
    t.move(1, 150, 500);
    t.move(2, 810, 300);
    expect(t.walk().right).toBeCloseTo(1);
    expect(t.takeLook()).toEqual([10, 0]);
  });
});
