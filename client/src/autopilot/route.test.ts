import { describe, expect, it } from "vitest";
import { newBody } from "../player/physics";
import { Steering } from "./route";

const forwardOf = (yaw: number) => ({ x: -Math.sin(yaw), z: -Math.cos(yaw) });

describe("Steering", () => {
  it("faces the target and flies toward it", () => {
    const s = new Steering();
    const step = s.steer(0.1, newBody(0, 10, 0), { x: 20, y: 10, z: 0 });
    expect(forwardOf(step.yaw).x).toBeCloseTo(1);
    expect(forwardOf(step.yaw).z).toBeCloseTo(0);
    expect(step.move).toMatchObject({ forward: 1, jump: false, descend: false });
  });

  it("rises and sinks toward the target's height", () => {
    const s = new Steering();
    expect(s.steer(0.1, newBody(0, 0, 0), { x: 0, y: 10, z: 0 }).move.jump).toBe(true);
    expect(s.steer(0.1, newBody(0, 20, 0), { x: 0, y: 10, z: 0 }).move.descend).toBe(true);
  });

  it("sinks on the way when told the way is clear", () => {
    const s = new Steering();
    expect(s.steer(0.1, newBody(0, 20, 0), { x: 20, y: 10, z: 0 }).move.descend).toBe(false);
    expect(s.steer(0.1, newBody(0, 20, 0), { x: 20, y: 10, z: 0 }, false).move.descend).toBe(true);
  });

  it("slows down close to the target instead of overshooting", () => {
    const step = new Steering().steer(0.1, newBody(0, 10, 0), { x: 0.5, y: 10, z: 0 });
    expect(step.move.forward).toBeLessThan(0.5);
  });

  it("knows when it has arrived", () => {
    const s = new Steering();
    expect(s.arrived(newBody(0, 10, 0), { x: 0.3, y: 10.2, z: 0 })).toBe(true);
    expect(s.arrived(newBody(0, 10, 0), { x: 2, y: 10, z: 0 })).toBe(false);
  });

  it("tries going over, around either side, then back when stuck again and again", () => {
    const s = new Steering();
    const body = newBody(0, 10, 0);
    const target = { x: 20, y: 10, z: 0 };
    const ways = [];
    for (let i = 0; i < 21; i++) {
      const step = s.steer(0.125, body, target);
      if (step.stuck) ways.push(step.move);
    }
    expect(ways.map((m) => [m.jump, m.right, m.forward])).toEqual([
      [true, 0, 0],
      [false, 1, 0],
      [false, -1, 0],
      [false, 0, -1],
      [true, 0, 0],
    ]);
  });

  it("climbs straight up for a while when stuck", () => {
    const s = new Steering();
    const body = newBody(0, 10, 0);
    const target = { x: 20, y: 10, z: 0 };
    const steps = (n: number) => {
      for (let i = 0; i < n; i++) s.steer(0.125, body, target);
    };
    steps(4); // half a second without moving
    const stuck = s.steer(0.125, body, target);
    expect(stuck).toMatchObject({ stuck: true, move: { forward: 0, jump: true } });
    expect(s.steer(0.125, body, target).stuck).toBe(false);
    steps(2);
    body.y += 5; // up and over, then on without sinking back into the wall
    expect(s.steer(0.125, body, target).move).toMatchObject({ forward: 1, descend: false });
  });
});
