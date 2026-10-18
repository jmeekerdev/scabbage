import { describe, expect, it } from "vitest";
import {
  FLY_SPEED,
  HALF_WIDTH,
  newBody,
  stepBody,
  WALK_SPEED,
  type Body,
  type MoveInput,
  type SolidAt,
} from "./physics";

const DT = 1 / 60;
const idle: MoveInput = { forward: 0, right: 0, jump: false, descend: false };
const floor: SolidAt = (_x, y) => y < 10;
const wallAtX5: SolidAt = (x, y, z) => floor(x, y, z) || x === 5;

function run(b: Body, input: Partial<MoveInput>, seconds: number, solid = floor, yaw = 0): Body {
  for (let t = 0; t < seconds; t += DT) stepBody(b, { ...idle, ...input }, yaw, DT, solid);
  return b;
}

describe("stepBody", () => {
  it("falls onto the ground and stays there", () => {
    const b = run(newBody(0.5, 15, 0.5), {}, 2);
    expect(b.y).toBeCloseTo(10, 5);
    expect(b.onGround).toBe(true);
    expect(b.vy).toBe(0);
  });

  it("walks forward along -z at walking speed when yaw is 0", () => {
    const b = run(newBody(0.5, 10, 0.5), { forward: 1 }, 1);
    expect(b.z).toBeCloseTo(0.5 - WALK_SPEED, 1);
    expect(b.x).toBeCloseTo(0.5, 5);
  });

  it("turns movement with yaw", () => {
    const b = run(newBody(0.5, 10, 0.5), { forward: 1 }, 1, floor, Math.PI / 2);
    expect(b.x).toBeCloseTo(0.5 - WALK_SPEED, 1);
  });

  it("does not move faster diagonally", () => {
    const b = run(newBody(0, 10, 0), { forward: 1, right: 1 }, 1);
    expect(Math.hypot(b.x, b.z)).toBeCloseTo(WALK_SPEED, 1);
  });

  it("stops at a wall without overlapping it", () => {
    const b = run(newBody(0.5, 10, 0.5), { right: 1 }, 3, wallAtX5);
    expect(b.x).toBeCloseTo(5 - HALF_WIDTH, 5);
    expect(b.x + HALF_WIDTH).toBeLessThanOrEqual(5);
  });

  it("slides along a wall when moving diagonally into it", () => {
    const b = run(newBody(4.5, 10, 0.5), { right: 1, forward: 1 }, 1, wallAtX5);
    expect(b.z).toBeLessThan(-2);
  });

  it("jumps a little over one block high", () => {
    const b = newBody(0.5, 10, 0.5);
    run(b, {}, 0.1);
    stepBody(b, { ...idle, jump: true }, 0, DT, floor);
    let top = b.y;
    for (let i = 0; i < 120; i++) {
      stepBody(b, idle, 0, DT, floor);
      top = Math.max(top, b.y);
    }
    expect(top - 10).toBeGreaterThan(1.1);
    expect(top - 10).toBeLessThan(1.5);
    expect(b.y).toBeCloseTo(10, 5);
  });

  it("cannot jump in mid-air", () => {
    const b = newBody(0.5, 20, 0.5);
    stepBody(b, { ...idle, jump: true }, 0, DT, floor);
    expect(b.vy).toBeLessThanOrEqual(0);
  });

  it("bumps its head on a ceiling", () => {
    const ceiling: SolidAt = (x, y, z) => floor(x, y, z) || y === 12;
    const b = run(newBody(0.5, 10, 0.5), {}, 0.1, ceiling);
    stepBody(b, { ...idle, jump: true }, 0, DT, ceiling);
    run(b, {}, 0.3, ceiling);
    expect(b.y + 1.8).toBeLessThanOrEqual(12);
  });

  it("flies without gravity, rising and sinking on demand", () => {
    const b = newBody(0.5, 20, 0.5);
    b.flying = true;
    run(b, {}, 1);
    expect(b.y).toBeCloseTo(20, 5);
    run(b, { jump: true }, 0.5);
    expect(b.y).toBeGreaterThan(22);
    run(b, { forward: 1 }, 1);
    expect(b.z).toBeCloseTo(0.5 - FLY_SPEED, 0);
  });

  it("lands when flying down onto the ground", () => {
    const b = newBody(0.5, 11, 0.5);
    b.flying = true;
    run(b, { descend: true }, 1);
    expect(b.flying).toBe(false);
    expect(b.onGround).toBe(true);
  });

  it("stays put inside unloaded (solid) terrain instead of being pushed out", () => {
    const b = run(newBody(0.5, 10, 0.5), { forward: 1, right: 1 }, 1, () => true);
    expect([b.x, b.y, b.z]).toEqual([0.5, 10, 0.5]);
  });
});
