import { describe, expect, it } from "vitest";
import { MobKind } from "../pb/scabbage/v1/net_pb";
import { type MobInfo, Mobs, pickMob } from "./mobs";
import { METER } from "../world/constants";

const sheep = (x: number, more: Partial<MobInfo> = {}): MobInfo => ({
  id: 7,
  kind: MobKind.SHEEP,
  pos: { x, y: 10, z: 0 },
  yaw: 0,
  sheared: false,
  hurt: false,
  ...more,
});

describe("Mobs", () => {
  it("interpolates moves a little in the past", () => {
    const m = new Mobs(100);
    m.change([sheep(0)], [], 0);
    m.change([sheep(1)], [], 100);
    expect(m.sample(100)[0]?.x).toBe(0);
    expect(m.sample(250)[0]?.x).toBeCloseTo(0.5);
    expect(m.sample(400)[0]?.x).toBe(1);
  });

  it("keeps wool and hurt flashes up to date", () => {
    const m = new Mobs(100);
    m.change([sheep(0)], [], 0);
    m.change([sheep(0, { sheared: true, hurt: true })], [], 50);
    const s = m.sample(60)[0];
    expect(s?.sheared).toBe(true);
    expect(s?.hurt).toBe(true);
    expect(m.sample(1000)[0]?.hurt).toBe(false);
  });

  it("keeps a boss's health up to date", () => {
    const m = new Mobs(100);
    const king = { kind: MobKind.KING_SLIME, health: 24, maxHealth: 24 };
    m.change([sheep(0, king)], [], 0);
    m.change([sheep(0, { ...king, health: 17 })], [], 50);
    expect(m.sample(60)[0]).toMatchObject({ health: 17, maxHealth: 24 });
  });

  it("forgets mobs that are gone", () => {
    const m = new Mobs(100);
    m.change([sheep(0)], [], 0);
    m.change([], [7], 10);
    expect(m.sample(20)).toEqual([]);
  });
});

describe("pickMob", () => {
  const m = new Mobs(100);
  m.change([sheep(0, { pos: { x: 0, y: 10, z: -3 * METER } })], [], 0);
  const mobs = m.sample(0);
  const eye = 10 + 0.5 * METER;

  it("finds the mob the ray points at, with its distance", () => {
    const hit = pickMob([0, eye, 0], [0, 0, -1], mobs, 4 * METER);
    expect(hit?.id).toBe(7);
    expect(hit?.distance).toBeCloseTo((3 - 0.45) * METER);
  });

  it("hits a golem's tall body", () => {
    const g = new Mobs(100);
    g.change([sheep(0, { kind: MobKind.GOLEM, pos: { x: 0, y: 10, z: -3 * METER } })], [], 0);
    expect(pickMob([0, 10 + 2.3 * METER, 0], [0, 0, -1], g.sample(0), 4 * METER)?.id).toBe(7);
  });

  it("misses mobs off to the side or out of reach", () => {
    expect(pickMob([2 * METER, eye, 0], [0, 0, -1], mobs, 4 * METER)).toBeUndefined();
    expect(pickMob([0, eye, 0], [0, 0, -1], mobs, 2 * METER)).toBeUndefined();
    expect(pickMob([0, eye, 0], [0, 0, 1], mobs, 4 * METER)).toBeUndefined();
  });
});
