import { describe, expect, it } from "vitest";
import { zoneGreeting, Zones, type ZoneInfo } from "./zones";

const zone = (name: string, half: number, build = false): ZoneInfo => ({
  name,
  x0: -half,
  z0: -half,
  x1: half,
  z1: half,
  build,
});

describe("Zones", () => {
  const zones = new Zones();
  zones.set([zone("Spawn", 16), zone("Home", 2, true)]);

  it("finds the smallest zone over a spot", () => {
    expect(zones.at(0, 0)?.name).toBe("Home");
    expect(zones.at(10, -16)?.name).toBe("Spawn");
    expect(zones.at(17, 0)).toBeUndefined();
  });

  it("says where blocks can be changed", () => {
    expect(zones.canBuild(1, 2)).toBe(true);
    expect(zones.canBuild(5, 5)).toBe(false);
    expect(zones.canBuild(40, 0)).toBe(true);
  });

  it("forgets old zones when they change", () => {
    const z = new Zones();
    z.set([zone("Spawn", 16)]);
    z.set([]);
    expect(z.canBuild(0, 0)).toBe(true);
  });
});

describe("zoneGreeting", () => {
  it("tells visitors they can't build", () => {
    expect(zoneGreeting(zone("Home", 2, true))).toBe("Welcome to Home!");
    expect(zoneGreeting(zone("Spawn", 2))).toBe(
      "Welcome to Spawn! Only its builders can change blocks here.",
    );
  });
});
