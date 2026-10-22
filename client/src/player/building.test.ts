import { describe, expect, it } from "vitest";
import {
  aim,
  breakEdit,
  buildWorldFor,
  digMs,
  placeEdit,
  PressFilter,
  UseHold,
  usesBlock,
  type BuildWorld,
} from "./building";
import { newBody } from "./physics";
import { Shape } from "../world/shapes";

const [AIR, STONE, WATER, BEDROCK, TORCH, STAIRS, PISTON, PAD] = [0, 1, 5, 12, 30, 40, 70, 80];
const solid = Array.from({ length: 44 }, (_, id) => id === STONE || id === BEDROCK || id >= STAIRS);
const targetable = solid.map((s, id) => s || id === TORCH);
const shape = solid.map((_, id) => (id >= STAIRS ? Shape.STAIRS_NORTH + id - STAIRS : Shape.CUBE));

/** Stone floor below y=10, bedrock at x=9, water at 3,10,3; unloaded past x=20. */
function world(): BuildWorld {
  return {
    solid,
    targetable,
    shape,
    usable: new Set(),
    liquid: new Set([WATER]),
    sixWay: new Set([PISTON]),
    padWay: new Set([PAD]),
    item: [],
    unbreakable: new Set([BEDROCK]),
    mineMs: [],
    inside: (_x, y) => y >= 0 && y < 64,
    blockAt: (x, y, z) => {
      if (x > 20) return undefined;
      if (x === 9 && y === 10) return BEDROCK;
      if (x === 3 && y === 10 && z === 3) return WATER;
      if (x === 4 && y === 10 && z === 4) return TORCH;
      return y < 10 ? STONE : AIR;
    },
  };
}

const down = -Math.PI / 2 + 0.01;

describe("buildWorldFor", () => {
  it("keeps each block's mining time", () => {
    const blocks = [
      { id: 0, name: "air", solid: false, shape: 0, mineMs: 0 },
      { id: 1, name: "stone", solid: true, shape: 0, mineMs: 1500 },
    ];
    const w = buildWorldFor({ radius: -1, height: 64, blocks }, () => AIR);
    expect(w.mineMs[STONE]).toBe(1500);
  });

  it("knows water as a liquid", () => {
    const blocks = [{ id: WATER, name: "water", solid: false, shape: 0, mineMs: 0 }];
    const w = buildWorldFor({ radius: -1, height: 64, blocks }, () => AIR);
    expect([...w.liquid]).toEqual([WATER]);
  });

  it("aims at flat blocks, presses usable ones and holds powered ones as off", () => {
    const blocks = [
      { id: 60, name: "wire", solid: false, shape: Shape.FLAT, mineMs: 0 },
      { id: 61, name: "wire_on", solid: false, shape: Shape.FLAT, mineMs: 0, item: 60 },
      { id: 62, name: "switch", solid: true, shape: 0, mineMs: 300, usable: true },
      { id: 63, name: "door_open", solid: false, shape: 0, mineMs: 600, usable: true, item: 62 },
      { id: 64, name: "piston", solid: true, shape: 0, mineMs: 600 },
      { id: 65, name: "diode", solid: true, shape: 0, mineMs: 300 },
      { id: 66, name: "gate_and", solid: true, shape: 0, mineMs: 300, usable: true },
      { id: 67, name: "delay_1", solid: true, shape: 0, mineMs: 300, usable: true },
      { id: 68, name: "relay", solid: true, shape: 0, mineMs: 300 },
      { id: 69, name: "capacitor_1", solid: true, shape: 0, mineMs: 300, usable: true },
      { id: 70, name: "block_detector", solid: true, shape: 0, mineMs: 300 },
      { id: 71, name: "sticky_piston", solid: true, shape: 0, mineMs: 600 },
      { id: 72, name: "launch_pad", solid: false, shape: Shape.FLAT, mineMs: 300 },
    ];
    const w = buildWorldFor({ radius: -1, height: 64, blocks }, () => AIR);
    expect(w.targetable[61] && w.targetable[63]).toBe(true);
    expect([...w.sixWay]).toEqual([64, 65, 66, 67, 68, 69, 70, 71]);
    expect([...w.padWay]).toEqual([72]);
    expect([...w.usable]).toEqual([62, 63, 66, 67, 69]);
    expect([w.item[61], w.item[62]]).toEqual([60, 62]);
  });
});

describe("aim", () => {
  it("targets the solid block in view within reach", () => {
    expect(aim([0.5, 11.62, 0.5], 0, down, world())).toMatchObject({ x: 0, y: 9, z: 0, ny: 1 });
  });

  it("targets water from above, but looks through it from underwater", () => {
    expect(aim([3.5, 11, 3.5], 0, down, world())).toMatchObject({ x: 3, y: 10, z: 3 });
    expect(aim([3.5, 10.5, 3.5], 0, down, world())).toMatchObject({ y: 9 });
  });

  it("ignores far or unloaded blocks", () => {
    expect(aim([0.5, 30, 0.5], 0, down, world())).toBeUndefined();
    expect(aim([20.5, 9.5, 0.5], -Math.PI / 2, 0, world())).toBeUndefined();
  });
});

describe("breakEdit", () => {
  it("breaks to air, except unbreakable blocks", () => {
    const w = world();
    const hit = { x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    expect(breakEdit(hit, w)).toEqual({
      x: 2,
      y: 9,
      z: 0,
      size: 1,
      block: AIR,
      cells: [{ x: 2, y: 9, z: 0 }],
    });
    expect(breakEdit({ ...hit, x: 9, y: 10 }, w)).toBeUndefined();
  });

  it("breaks the water it aims at, but leaves water in a brush aimed at other blocks", () => {
    const water = { x: 3, y: 10, z: 3, nx: 0, ny: 1, nz: 0, distance: 1 };
    expect(breakEdit(water, world())?.cells).toEqual([{ x: 3, y: 10, z: 3 }]);
    const side = { x: 3, y: 9, z: 3, nx: 0, ny: 1, nz: 0, distance: 1 };
    expect(breakEdit(side, world(), 3)?.cells).not.toContainEqual({ x: 3, y: 10, z: 3 });
  });

  it("breaks only the breakable blocks of a brush", () => {
    const side = { x: 9, y: 10, z: 1, nx: -1, ny: 0, nz: 0, distance: 2 };
    const edit = breakEdit(side, world(), 3);
    expect(edit).toMatchObject({ x: 9, y: 9, z: 0, size: 3 });
    expect(edit?.cells).toEqual([
      { x: 9, y: 9, z: 0 },
      { x: 9, y: 9, z: 1 },
      { x: 9, y: 9, z: 2 },
      { x: 10, y: 9, z: 0 },
      { x: 10, y: 9, z: 1 },
      { x: 10, y: 9, z: 2 },
      { x: 11, y: 9, z: 0 },
      { x: 11, y: 9, z: 1 },
      { x: 11, y: 9, z: 2 },
    ]); // stone below y=10; bedrock and air above
  });
});

describe("digMs", () => {
  it("is the slowest block's time for each block of the brush's width", () => {
    const w = { ...world(), mineMs: [0, 500] };
    const edit = breakEdit({ x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 }, w, 3)!;
    expect(digMs(edit, w)).toBe(1500);
  });
});

describe("placeEdit", () => {
  const body = newBody(0.5, 10, 0.5);
  const onTopOf = (x: number, z: number) => ({ x, y: 9, z, nx: 0, ny: 1, nz: 0, distance: 2 });

  it("places against the face that was hit", () => {
    expect(placeEdit(onTopOf(2, 0), STONE, body, { yaw: 0, pitch: 0 }, world())).toEqual({
      x: 2,
      y: 10,
      z: 0,
      size: 1,
      block: STONE,
      cells: [{ x: 2, y: 10, z: 0 }],
    });
  });

  it("fills a brush around the player's body and anything already there", () => {
    const edit = placeEdit(onTopOf(1, 1), STONE, body, { yaw: 0, pitch: 0 }, world(), 3);
    expect(edit).toMatchObject({ x: 0, y: 10, z: 0, size: 3 });
    expect(edit?.cells).toHaveLength(27 - 12); // the body fills two by two columns, three high
    expect(edit?.cells).not.toContainEqual({ x: 1, y: 12, z: 1 });
    const torch = placeEdit(onTopOf(4, 4), STONE, body, { yaw: 0, pitch: 0 }, world(), 3);
    expect(torch?.cells).toHaveLength(26);
  });

  it("replaces water", () => {
    expect(placeEdit(onTopOf(3, 3), STONE, body, { yaw: 0, pitch: 0 }, world())).toMatchObject({
      x: 3,
      y: 10,
      z: 3,
    });
  });

  it("won't place a solid block inside the player, but will under their feet", () => {
    expect(placeEdit(onTopOf(0, 0), STONE, body, { yaw: 0, pitch: 0 }, world())).toBeUndefined();
    const jumping = { ...body, y: 11.05 };
    expect(placeEdit(onTopOf(0, 0), STONE, jumping, { yaw: 0, pitch: 0 }, world())).toMatchObject({
      y: 10,
    });
  });

  it("won't place outside the world, into solid blocks, or unplaceable blocks", () => {
    const w = world();
    const top = { x: 2, y: 63, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    expect(placeEdit(top, STONE, body, { yaw: 0, pitch: 0 }, w)).toBeUndefined();
    expect(
      placeEdit({ ...onTopOf(2, 0), ny: -1, y: 10 }, STONE, body, { yaw: 0, pitch: 0 }, w),
    ).toBeUndefined();
    expect(placeEdit(onTopOf(2, 0), BEDROCK, body, { yaw: 0, pitch: 0 }, w)).toBeUndefined();
    expect(placeEdit(onTopOf(2, 0), AIR, body, { yaw: 0, pitch: 0 }, w)).toBeUndefined();
  });
});

describe("torches and stairs", () => {
  it("aims at torches and won't place into them", () => {
    expect(aim([4.5, 11.62, 4.5], 0, down, world())).toMatchObject({ x: 4, y: 10, z: 4 });
    const beside = { x: 5, y: 10, z: 4, nx: -1, ny: 0, nz: 0, distance: 2 };
    expect(
      placeEdit(beside, STONE, newBody(8.5, 10, 8.5), { yaw: 0, pitch: 0 }, world()),
    ).toBeUndefined();
  });

  it("turns stairs to rise away from the player", () => {
    const hit = { x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    const body = newBody(0.5, 10, 0.5);
    expect(placeEdit(hit, STAIRS, body, { yaw: Math.PI, pitch: 0 }, world())?.block).toBe(
      STAIRS + 2,
    );
  });

  it("turns pistons to push where the player looks", () => {
    const hit = { x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    const body = newBody(0.5, 10, 0.5);
    expect(placeEdit(hit, PISTON, body, { yaw: -Math.PI / 2, pitch: 0 }, world())?.block).toBe(
      PISTON + 2,
    );
    expect(placeEdit(hit, PISTON, body, { yaw: 0, pitch: down }, world())?.block).toBe(PISTON + 10);
  });

  it("turns launch pads to throw where the player looks, or up when they look down", () => {
    const hit = { x: 2, y: 9, z: 0, nx: 0, ny: 1, nz: 0, distance: 2 };
    const body = newBody(0.5, 10, 0.5);
    const place = (yaw: number, pitch: number) =>
      placeEdit(hit, PAD, body, { yaw, pitch }, world());
    expect(place(-Math.PI / 2, -0.3)?.block).toBe(PAD + 2);
    expect(place(0, down)?.block).toBe(PAD + 8);
  });
});

describe("usesBlock", () => {
  it("presses usable blocks unless sneaking, so doors can be stacked", () => {
    const w = { ...world(), usable: new Set([TORCH]) };
    expect(usesBlock(TORCH, false, w)).toBe(true);
    expect(usesBlock(TORCH, true, w)).toBe(false);
    expect(usesBlock(STONE, false, w)).toBe(false);
  });
});

describe("PressFilter", () => {
  it("lets a held button through once", () => {
    const presses = new PressFilter(250);
    expect([0, 250, 500, 1000, 1500].map((t) => presses.fresh(t))).toEqual([
      true,
      false,
      false,
      true,
      true,
    ]);
  });
});

describe("UseHold", () => {
  it("keeps a held press that used a block from placing until it is let go", () => {
    const hold = new UseHold(250);
    expect(hold.blocks(0)).toBe(false);
    hold.used(0);
    expect([250, 500, 750].map((t) => hold.blocks(t))).toEqual([true, true, true]);
    expect(hold.blocks(1500)).toBe(false);
  });
});
