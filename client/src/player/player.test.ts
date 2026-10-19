import { describe, expect, it } from "vitest";
import { EYE_HEIGHT, WALK_SPEED, type MoveInput, type BoxesAt } from "./physics";
import { FULL_BLOCK } from "../world/shapes";
import { Player } from "./player";

const floor: BoxesAt = (_x, y) => (y < 10 ? FULL_BLOCK : []);
const walk: MoveInput = { forward: 1, right: 0, jump: false, descend: false };

describe("Player", () => {
  it("moves the same however the frames are sliced", () => {
    const a = new Player();
    const b = new Player();
    a.place(0.5, 10, 0.5);
    b.place(0.5, 10, 0.5);
    for (let i = 0; i < 60; i++) a.update(1 / 60, walk, floor);
    for (let i = 0; i < 24; i++) b.update(1 / 24, walk, floor);
    expect(b.body.z).toBeCloseTo(a.body.z, 6);
    expect(a.body.z).toBeCloseTo(0.5 - WALK_SPEED, 1);
  });

  it("ignores the time lost in a long stall", () => {
    const p = new Player();
    p.place(0.5, 10, 0.5);
    p.update(10, walk, floor);
    expect(p.body.z).toBeGreaterThan(0.5 - WALK_SPEED * 0.3);
  });

  it("puts the eye between the last two steps for smooth frames", () => {
    const p = new Player();
    p.place(0.5, 10, 0.5);
    p.update(1 / 60, walk, floor);
    const z1 = p.body.z;
    p.update(1 / 120, walk, floor);
    const eye = p.eye();
    expect(p.body.z).toBe(z1);
    expect(eye.y).toBeCloseTo(10 + EYE_HEIGHT);
    expect(eye.z).toBeCloseTo((0.5 + z1) / 2);
  });

  it("stops and lands nowhere special when placed", () => {
    const p = new Player();
    p.place(0.5, 20, 0.5);
    p.toggleFly();
    p.place(3, 30, 3);
    expect(p.eye()).toEqual({ x: 3, y: 30 + EYE_HEIGHT, z: 3 });
    expect(p.body.vy).toBe(0);
  });

  it("only flies when allowed", () => {
    const p = new Player();
    p.toggleFly();
    expect(p.body.flying).toBe(true);
    p.allowFlying(false);
    expect(p.body.flying).toBe(false);
    p.toggleFly();
    expect(p.body.flying).toBe(false);
    p.allowFlying(true);
    p.toggleFly();
    expect(p.body.flying).toBe(true);
  });
});
