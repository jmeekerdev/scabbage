import { describe, expect, it } from "vitest";
import { aim, breakEdit, placeEdit, type BuildWorld } from "./building";
import { newBody } from "./physics";

const [AIR, STONE, WATER, BEDROCK] = [0, 1, 5, 12];
const solid = Array.from({ length: 13 }, (_, id) => id === STONE || id === BEDROCK);

/** Stone floor below y=10, bedrock at x=9, water at 3,10,3; unloaded past x=20. */
function world(): BuildWorld {
  return {
    solid,
    unbreakable: new Set([BEDROCK]),
    inside: (_x, y) => y >= 0 && y < 64,
    blockAt: (x, y, z) => {
      if (x > 20) return undefined;
      if (x === 9 && y === 10) return BEDROCK;
      if (x === 3 && y === 10 && z === 3) return WATER;
      return y < 10 ? STONE : AIR;
    },
  };
}

const down = -Math.PI / 2 + 0.01;

describe("aim", () => {
  it("targets the solid block in view within reach", () => {
    expect(aim([0.5, 11.62, 0.5], 0, down, world())).toMatchObject({ x: 0, y: 9, z: 0, ny: 1 });
  });

  it("looks through water and ignores far or unloaded blocks", () => {
    expect(aim([3.5, 11, 3.5], 0, down, world())).toMatchObject({ y: 9 });
    expect(aim([0.5, 30, 0.5], 0, down, world())).toBeUndefined();
    expect(aim([20.5, 9.5, 0.5], -Math.PI / 2, 0, world())).toBeUndefined();
  });
});

describe("breakEdit", () => {
  it("breaks to air, except unbreakable blocks", () => {
    const w = world();
    const hit = { x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    expect(breakEdit(hit, w)).toEqual({ x: 2, y: 9, z: 0, block: AIR });
    expect(breakEdit({ ...hit, x: 9, y: 10 }, w)).toBeUndefined();
  });
});

describe("placeEdit", () => {
  const body = newBody(0.5, 10, 0.5);
  const onTopOf = (x: number, z: number) => ({ x, y: 9, z, nx: 0, ny: 1, nz: 0, distance: 2 });

  it("places against the face that was hit", () => {
    expect(placeEdit(onTopOf(2, 0), STONE, body, world())).toEqual({
      x: 2,
      y: 10,
      z: 0,
      block: STONE,
    });
  });

  it("replaces water", () => {
    expect(placeEdit(onTopOf(3, 3), STONE, body, world())).toMatchObject({ x: 3, y: 10, z: 3 });
  });

  it("won't place a solid block inside the player, but will under their feet", () => {
    expect(placeEdit(onTopOf(0, 0), STONE, body, world())).toBeUndefined();
    const jumping = { ...body, y: 11.05 };
    expect(placeEdit(onTopOf(0, 0), STONE, jumping, world())).toMatchObject({ y: 10 });
  });

  it("won't place outside the world, into solid blocks, or unplaceable blocks", () => {
    const w = world();
    const top = { x: 2, y: 63, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    expect(placeEdit(top, STONE, body, w)).toBeUndefined();
    expect(placeEdit({ ...onTopOf(2, 0), ny: -1, y: 10 }, STONE, body, w)).toBeUndefined();
    expect(placeEdit(onTopOf(2, 0), BEDROCK, body, w)).toBeUndefined();
    expect(placeEdit(onTopOf(2, 0), AIR, body, w)).toBeUndefined();
  });
});
