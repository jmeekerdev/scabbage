import { describe, expect, it } from "vitest";
import { distanceGain, Footsteps } from "./cues";
import { METRE } from "../world/constants";

const ground = { onGround: true, flying: false };

describe("Footsteps", () => {
  it("steps once per stride walked on the ground", () => {
    const f = new Footsteps();
    const steps = Array.from({ length: 30 }, (_, i) =>
      f.update({ x: i * 0.2 * METRE, z: 0, ...ground }),
    );
    expect(steps.filter(Boolean)).toHaveLength(4); // 5.8 m at a 1.5 m stride
    expect(steps.slice(0, 2)).toEqual([false, true]); // the first as soon as they move
  });

  it("stays quiet in the air and while flying", () => {
    const f = new Footsteps();
    const air = Array.from({ length: 30 }, (_, i) =>
      f.update({ x: i * 0.2, z: 0, onGround: false, flying: false }),
    );
    const fly = Array.from({ length: 30 }, (_, i) =>
      f.update({ x: i * 0.2, z: 0, onGround: true, flying: true }),
    );
    expect([...air, ...fly].some(Boolean)).toBe(false);
  });

  it("does not step for a teleport", () => {
    const f = new Footsteps();
    f.update({ x: 0, z: 0, ...ground });
    expect(f.update({ x: 40 * METRE, z: 0, ...ground })).toBe(false);
  });
});

describe("distanceGain", () => {
  it("is full volume close by and silent far away", () => {
    expect(distanceGain(0)).toBe(1);
    expect(distanceGain(4 * METRE)).toBe(1);
    expect(distanceGain(14 * METRE)).toBeCloseTo(0.5);
    expect(distanceGain(24 * METRE)).toBe(0);
    expect(distanceGain(100 * METRE)).toBe(0);
  });
});
