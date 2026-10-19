import { describe, expect, it } from "vitest";
import type { Pose } from "./move-sync";
import { RemotePlayers } from "./remote-players";

const pose = (x: number, yaw = 0): Pose => ({ x, y: 10, z: 0, yaw, pitch: 0 });
const mom = { entity: 1, name: "Mom", color: 0xff0000 };

/** Remote players delayed by two 50 ms ticks, with Mom joining at x=0 at time 0. */
function withMom(): RemotePlayers {
  const r = new RemotePlayers(50);
  r.join(mom, pose(0), 0);
  return r;
}

const xAt = (r: RemotePlayers, nowMs: number) => r.sample(nowMs)[0]?.pose.x;

describe("RemotePlayers", () => {
  it("shows a player where they joined", () => {
    expect(withMom().sample(0)).toEqual([{ ...mom, pose: pose(0) }]);
  });

  it("interpolates between moves, a little in the past", () => {
    const r = withMom();
    r.move(1, pose(1), 50);
    r.move(1, pose(2), 100);
    expect(xAt(r, 125)).toBeCloseTo(0.5);
    expect(xAt(r, 175)).toBeCloseTo(1.5);
  });

  it("holds the last pose once moves stop", () => {
    const r = withMom();
    r.move(1, pose(2), 50);
    expect(xAt(r, 1000)).toBe(2);
  });

  it("starts moving from where the player stood after a pause", () => {
    const r = withMom();
    r.move(1, pose(1), 1000);
    expect(xAt(r, 1000)).toBe(0);
    expect(xAt(r, 1075)).toBeCloseTo(0.5);
    expect(xAt(r, 1100)).toBe(1);
  });

  it("turns the short way round", () => {
    const r = new RemotePlayers(50);
    r.join(mom, pose(0, 3), 0);
    r.move(1, pose(0, -3), 50);
    const yaw = r.sample(125)[0]?.pose.yaw ?? 0;
    expect(Math.abs(yaw)).toBeGreaterThan(3);
  });

  it("forgets players who leave and ignores moves for strangers", () => {
    const r = withMom();
    r.join({ ...mom, entity: 2, name: "Kid" }, pose(5), 0);
    r.move(9, pose(1), 50);
    r.leave(1);
    expect(r.sample(100).map((p) => p.name)).toEqual(["Kid"]);
  });
});
