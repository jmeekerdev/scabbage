import { describe, expect, it } from "vitest";
import { lookFacing, parseBuild, planNeeded } from "./command";

const names = new Map([
  ["planks", 8],
  ["stone", 1],
  ["glass", 10],
]);
const feet = { x: 10, y: 20, z: 30 };

describe("lookFacing", () => {
  it("names the way a yaw looks; yaw 0 looks north and turning left raises it", () => {
    expect(lookFacing(0)).toBe("north");
    expect(lookFacing(Math.PI / 2)).toBe("west");
    expect(lookFacing(Math.PI)).toBe("south");
    expect(lookFacing(-Math.PI / 2 + 0.3)).toBe("east");
  });
});

describe("parseBuild", () => {
  it("builds just ahead of the player, its front toward them", () => {
    const job = parseBuild("floor 3 2 1", feet, 0, names);
    if (typeof job === "string") throw new Error(job);
    expect(job.template).toBe("floor");
    const zs = job.cells.map((c) => c.z);
    expect([Math.min(...zs), Math.max(...zs)]).toEqual([26, 27]); // looking north, 3 ahead
    expect(job.cells.every((c) => c.y === 20)).toBe(true);
  });

  it("takes a material, a facing and a spot in any order after the size", () => {
    const job = parseBuild("wall stone east at 0 ~ 0", feet, 0, names);
    if (typeof job === "string") throw new Error(job);
    expect(job.cells.every((c) => c.block === 1)).toBe(true);
    expect(Math.max(...job.cells.map((c) => c.x))).toBe(0); // facing east, it runs to -x
    expect(Math.min(...job.cells.map((c) => c.y))).toBe(20);
  });

  it("names a saved plan it needs, and builds it once it comes", () => {
    expect(planNeeded("hut stone east")).toBe("hut");
    expect(planNeeded("cabin")).toBeUndefined();
    expect(planNeeded("")).toBeUndefined();
    const plan = { name: "hut", palette: { p: "planks" }, layers: [["ppp"]] };
    const job = parseBuild("hut stone", feet, 0, names, plan);
    if (typeof job === "string") throw new Error(job);
    expect(job.template).toBe("hut");
    expect(job.cells.map((c) => c.block)).toEqual([1, 1, 1]);
    expect(parseBuild("hut 3 3 3", feet, 0, names, plan)).toMatch(/can't be resized/);
  });

  it("explains what's wrong", () => {
    expect(parseBuild("", feet, 0, names)).toMatch(/Use \/build cabin\|tower/);
    expect(parseBuild("wall 3 3", feet, 0, names)).toMatch(/w d h/);
    expect(parseBuild("wall lava", feet, 0, names)).toMatch(/no block named lava/);
    expect(parseBuild("wall at 1 2", feet, 0, names)).toMatch(/at x y z/);
  });
});
