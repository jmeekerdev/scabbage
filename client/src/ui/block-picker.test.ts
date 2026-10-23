import { describe, expect, it } from "vitest";
import { buildable, placeable } from "./block-picker";

describe("block lists", () => {
  const blocks = [
    { id: 0, name: "air" },
    { id: 1, name: "stone" },
    { id: 2, name: "stone_stairs" },
    { id: 3, name: "stone_stairs_east", item: 2, turned: true },
    { id: 4, name: "lever_on", item: 5 },
    { id: 5, name: "lever" },
    { id: 6, name: "bedrock" },
  ];

  it("holds items only, but builds turned blocks as they are too", () => {
    expect(placeable(blocks).map((b) => b.name)).toEqual(["stone", "stone_stairs", "lever"]);
    expect(buildable(blocks).map((b) => b.name)).toEqual([
      "stone",
      "stone_stairs",
      "stone_stairs_east",
      "lever",
    ]);
  });
});
