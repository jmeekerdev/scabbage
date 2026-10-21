import { describe, expect, it } from "vitest";
import { AVATAR_COLORS } from "../join/rules";
import { Npcs } from "./npcs";

const npc = (id: number, name: string, x = 0, y = 10, z = 0) => ({ id, name, pos: { x, y, z } });

describe("characters", () => {
  it("appear, get renamed and go", () => {
    const n = new Npcs();
    n.change([npc(1, "Ivy"), npc(2, "Bo", 5)], []);
    n.change([npc(1, "Granny Ivy")], [2]);
    expect(n.figures({ x: 0, z: 0 }).map((f) => f.name)).toEqual(["Granny Ivy"]);
    n.clear();
    expect(n.figures({ x: 0, z: 0 })).toEqual([]);
  });

  it("stand in the middle of their block in a color from their name", () => {
    const n = new Npcs();
    n.change([npc(1, "Ivy", 3, 10, -4)], []);
    const [f] = n.figures({ x: 100, z: 100 });
    expect(f.pose).toMatchObject({ x: 3.5, y: 10, z: -3.5 });
    expect(AVATAR_COLORS).toContain(f.color);
    expect(new Npcs().figures({ x: 0, z: 0 })).toEqual([]);
  });

  it("turn to face someone nearby", () => {
    const n = new Npcs();
    n.change([npc(1, "Ivy")], []);
    const yaw = n.figures({ x: 0.5, z: 5 })[0].pose.yaw;
    // An avatar at yaw faces (-sin yaw, -cos yaw); the visitor is straight down +z.
    expect(-Math.sin(yaw)).toBeCloseTo(0);
    expect(-Math.cos(yaw)).toBeCloseTo(1);
  });

  it("are picked along a look", () => {
    const n = new Npcs();
    n.change([npc(1, "Ivy", 0, 10, -3)], []);
    expect(n.pick([0.5, 11.5, 0.5], [0, 0, -1], 5)?.id).toBe(1);
    expect(n.pick([0.5, 11.5, 0.5], [0, 0, 1], 5)).toBeUndefined();
  });
});
