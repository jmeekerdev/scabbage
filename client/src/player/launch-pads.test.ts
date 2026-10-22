import { describe, expect, it } from "vitest";
import { launchPads } from "./launch-pads";
import { METRE } from "../world/constants";

const blocks = [
  { id: 1, name: "launch_pad" },
  { id: 2, name: "launch_pad_on" },
  { id: 3, name: "launch_pad_east_on" },
  { id: 4, name: "launch_pad_up_on" },
  { id: 5, name: "stone" },
];

describe("launchPads", () => {
  const pads = launchPads(blocks);

  it("throws only from powered pads", () => {
    expect([pads.get(1), pads.get(5)]).toEqual([undefined, undefined]);
  });

  it("throws along a pad's facing and a little up, or high straight up", () => {
    const [north, east, up] = [pads.get(2)!, pads.get(3)!, pads.get(4)!];
    expect(north.z).toBeLessThan(0);
    expect([north.x, east.z]).toEqual([0, 0]);
    expect(east.x).toBe(-north.z);
    expect([up.x, up.z]).toEqual([0, 0]);
    expect(up.y).toBeGreaterThan(north.y);
  });

  it("stays within the speeds the server accepts", () => {
    for (const v of pads.values()) {
      expect(Math.hypot(v.x, v.z)).toBeLessThan(10 * METRE); // walking adds 4.5 m/s; limit is 14
      expect(v.y).toBeLessThan(14 * METRE);
    }
  });
});
