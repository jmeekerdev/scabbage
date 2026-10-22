import { describe, expect, it } from "vitest";
import { MobKind } from "../pb/scabbage/v1/net_pb";
import type { MobState } from "../player/mobs";
import { bossNear } from "./boss-bar";
import { METER } from "../world/constants";

const mob = (kind: MobKind, x: number, health = 0, maxHealth = 0): MobState => ({
  id: x,
  kind,
  x,
  y: 10,
  z: 0,
  yaw: 0,
  sheared: false,
  hurt: false,
  health,
  maxHealth,
});

describe("bossNear", () => {
  const me = { x: 0, y: 10, z: 0 };

  it("names the nearest boss in range with its health", () => {
    const mobs = [
      mob(MobKind.SLIME, 1),
      mob(MobKind.GOLEM, 20, 30, 40),
      mob(MobKind.KING_SLIME, 8, 6, 24),
    ];
    expect(bossNear(mobs, me)).toEqual({ name: "King Slime", health: 6, maxHealth: 24 });
  });

  it("is nothing when bosses are far away or missing", () => {
    expect(bossNear([mob(MobKind.GOLEM, 30 * METER, 30, 40)], me)).toBeUndefined();
    expect(bossNear([mob(MobKind.SHEEP, 2)], me)).toBeUndefined();
  });
});
