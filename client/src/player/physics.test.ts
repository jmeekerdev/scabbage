import { describe, expect, it } from "vitest";
import { FULL_BLOCK, type Box } from "../world/shapes";
import {
  FLY_SPEED,
  HALF_WIDTH,
  HEIGHT,
  launch,
  newBody,
  stepBody,
  WALK_SPEED,
  type Body,
  type MoveInput,
  type BoxesAt,
} from "./physics";
import { METER } from "../world/constants";

const DT = 1 / 60;
const idle: MoveInput = { forward: 0, right: 0, jump: false, descend: false };
const cubes =
  (solid: (x: number, y: number, z: number) => boolean): BoxesAt =>
  (x, y, z) =>
    solid(x, y, z) ? FULL_BLOCK : [];
const below10 = (_x: number, y: number) => y < 10;
const floor = cubes(below10);
const wallAtX5 = cubes((x, y) => below10(x, y) || x === 5);
const SLAB: Box[] = [[0, 0, 0, 1, 0.5, 1]];
const slabAtX5: BoxesAt = (x, y, z) => (x === 5 && y === 10 ? SLAB : floor(x, y, z));

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
    const b = run(newBody(3.5, 10, 0.5), { right: 1, forward: 1 }, 1, wallAtX5);
    expect(b.z).toBeLessThan(-2 * METER);
  });

  it("jumps a little over a meter high", () => {
    const b = newBody(0.5, 10, 0.5);
    run(b, {}, 0.1);
    stepBody(b, { ...idle, jump: true }, 0, DT, floor);
    let top = b.y;
    for (let i = 0; i < 120; i++) {
      stepBody(b, idle, 0, DT, floor);
      top = Math.max(top, b.y);
    }
    expect(top - 10).toBeGreaterThan(1.1 * METER);
    expect(top - 10).toBeLessThan(1.5 * METER);
    expect(b.y).toBeCloseTo(10, 5);
  });

  it("cannot jump in mid-air", () => {
    const b = newBody(0.5, 20, 0.5);
    stepBody(b, { ...idle, jump: true }, 0, DT, floor);
    expect(b.vy).toBeLessThanOrEqual(0);
  });

  it("bumps its head on a ceiling", () => {
    const ceiling = cubes((x, y) => below10(x, y) || y === 16);
    const b = run(newBody(0.5, 10, 0.5), {}, 0.1, ceiling);
    stepBody(b, { ...idle, jump: true }, 0, DT, ceiling);
    run(b, {}, 0.3, ceiling);
    expect(b.y + HEIGHT).toBeLessThanOrEqual(16);
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
    const b = run(newBody(0.5, 10, 0.5), { forward: 1, right: 1 }, 1, () => FULL_BLOCK);
    expect([b.x, b.y, b.z]).toEqual([0.5, 10, 0.5]);
  });

  it("stands on slabs", () => {
    const b = run(newBody(5.5, 12, 0.5), {}, 2, slabAtX5);
    expect(b.y).toBeCloseTo(10.5, 5);
    expect(b.onGround).toBe(true);
  });

  it("steps up onto slabs and single blocks while walking, but not up walls", () => {
    const up = run(newBody(3.5, 10, 0.5), { right: 1 }, 0.25, slabAtX5);
    expect(up.x).toBeGreaterThan(5);
    expect(up.y).toBeCloseTo(10.5, 5);
    const ledge = cubes((x, y) => below10(x, y) || (x >= 5 && y === 10));
    const onto = run(newBody(3.5, 10, 0.5), { right: 1 }, 0.25, ledge);
    expect(onto.x).toBeGreaterThan(5);
    expect(onto.y).toBeCloseTo(11, 5);
    const blocked = run(newBody(3.5, 10, 0.5), { right: 1 }, 1, wallAtX5);
    expect(blocked.x).toBeCloseTo(5 - HALF_WIDTH, 5);
    expect(blocked.y).toBeCloseTo(10, 5);
  });
});

describe("launch", () => {
  const settled = () => run(newBody(0.5, 10, 0.5), {}, 0.1);

  it("throws a body high straight up and lets it fall back", () => {
    const b = settled();
    launch(b, { x: 0, y: 13 * METER, z: 0 });
    let top = b.y;
    for (let t = 0; t < 2; t += DT) {
      stepBody(b, idle, 0, DT, floor);
      top = Math.max(top, b.y);
    }
    expect(top - 10).toBeGreaterThan(2.5 * METER);
    expect([b.y, b.x]).toEqual([10, 0.5]);
  });

  it("carries a body along through the air and stops it on landing", () => {
    const b = settled();
    launch(b, { x: 9 * METER, y: 9 * METER, z: 0 });
    run(b, {}, 2);
    expect(b.x - 0.5).toBeGreaterThan(5 * METER);
    expect(b.x - 0.5).toBeLessThan(7 * METER);
    expect([b.vx, b.vz, b.onGround]).toEqual([0, 0, true]);
  });
});
