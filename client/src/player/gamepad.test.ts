import { describe, expect, it } from "vitest";
import { GamepadControls, type PadState } from "./gamepad";
import { Hotbar } from "./hotbar";
import { Player } from "./player";

const BUTTONS = { a: 0, x: 2, y: 3, lb: 4, rb: 5, lt: 6, rt: 7 };

function pad(axes: number[] = [0, 0, 0, 0], held: (keyof typeof BUTTONS)[] = []): PadState {
  const pressed = new Set(held.map((b) => BUTTONS[b]));
  return { axes, buttons: Array.from({ length: 16 }, (_, i) => ({ pressed: pressed.has(i) })) };
}

/** Gamepad controls reading whatever pad `current` holds. */
function setup() {
  const player = new Player();
  const hotbar = new Hotbar([1, 2, 3]);
  const state = { current: pad() as PadState | null };
  const controls = new GamepadControls(player, hotbar, () => [state.current]);
  return { player, hotbar, state, controls };
}

describe("GamepadControls", () => {
  it("does nothing without a pad", () => {
    const { state, controls } = setup();
    state.current = null;
    expect(controls.poll(0).move.forward).toBe(0);
    expect(controls.active).toBe(false);
  });

  it("walks with the left stick, pushed up meaning forward", () => {
    const { state, controls } = setup();
    state.current = pad([1, -1, 0, 0]);
    const { move } = controls.poll(0);
    expect(move.forward).toBeGreaterThan(0.7);
    expect(move.right).toBeGreaterThan(0.7);
    expect(controls.active).toBe(true);
  });

  it("looks with the right stick at a rate, not a jump", () => {
    const { player, state, controls } = setup();
    controls.poll(0);
    state.current = pad([0, 0, 1, 0]);
    controls.poll(100);
    const after100 = player.look.yaw;
    controls.poll(200);
    expect(after100).toBeLessThan(0);
    expect(player.look.yaw).toBeCloseTo(2 * after100);
  });

  it("jumps with A and flies on a double tap", () => {
    const { player, state, controls } = setup();
    state.current = pad(undefined, ["a"]);
    expect(controls.poll(0).move.jump).toBe(true);
    state.current = pad();
    controls.poll(100);
    state.current = pad(undefined, ["a"]);
    controls.poll(200);
    expect(player.body.flying).toBe(true);
  });

  it("breaks with the right trigger, repeating while held", () => {
    const { state, controls } = setup();
    state.current = pad(undefined, ["rt"]);
    expect(controls.poll(0).build.breaking).toBe(true);
    expect(controls.poll(100).build.breaking).toBe(false);
    expect(controls.poll(300).build.breaking).toBe(true);
    state.current = pad(undefined, ["lt", "x"]);
    expect(controls.poll(400).build).toEqual({ breaking: false, placing: true, picking: true });
    expect(controls.poll(450).build.picking).toBe(false);
  });

  it("steps through the hotbar once per bumper press", () => {
    const { hotbar, state, controls } = setup();
    state.current = pad(undefined, ["rb"]);
    controls.poll(0);
    controls.poll(50);
    expect(hotbar.selected).toBe(1);
    state.current = pad(undefined, ["lb"]);
    controls.poll(100);
    expect(hotbar.selected).toBe(0);
  });
});
