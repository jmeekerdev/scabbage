import { describe, expect, it } from "vitest";
import { MoveSync, type Pose } from "./move-sync";

const pose = (x: number, yaw = 0): Pose => ({ x, y: 10, z: 0, yaw, pitch: 0 });

describe("MoveSync", () => {
  it("sends changes at most once per interval", () => {
    const s = new MoveSync(50);
    expect(s.due(0, pose(1))).toEqual(pose(1));
    expect(s.due(20, pose(2))).toBeUndefined();
    expect(s.due(50, pose(3))).toEqual(pose(3));
  });

  it("stays quiet while nothing changes but sends where the player stopped", () => {
    const s = new MoveSync(50);
    s.due(0, pose(1));
    s.due(10, pose(1.5));
    expect(s.due(60, pose(1.5))).toEqual(pose(1.5));
    expect(s.due(200, pose(1.5))).toBeUndefined();
  });

  it("sends turning on the spot", () => {
    const s = new MoveSync(50);
    s.due(0, pose(1));
    expect(s.due(100, pose(1, 0.5))).toEqual(pose(1, 0.5));
  });

  it("sends taking off on the spot", () => {
    const s = new MoveSync(50);
    s.due(0, pose(1));
    expect(s.due(100, { ...pose(1), flying: true })).toEqual({ ...pose(1), flying: true });
  });

  it("remembers the latest teleport and forgets what it sent before", () => {
    const s = new MoveSync(50);
    s.due(0, pose(1));
    s.teleported(3);
    expect(s.teleport).toBe(3);
    expect(s.due(100, pose(1))).toEqual(pose(1));
  });

  it("flushes a changed pose right away, e.g. just before a block edit", () => {
    const sync = new MoveSync(50);
    sync.due(0, pose(0));
    expect(sync.flush(pose(1))).toEqual(pose(1));
    expect(sync.flush(pose(1))).toBeUndefined();
    expect(sync.due(60, pose(1))).toBeUndefined();
  });
});
