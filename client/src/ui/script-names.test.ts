import { describe, expect, it } from "vitest";
import { scriptBlockNames } from "./script-names";

describe("scriptBlockNames", () => {
  it("offers built-in blocks scripts may place, with air last", () => {
    const blocks = [
      { id: 0, name: "air" },
      { id: 1, name: "bedrock" },
      { id: 2, name: "stone" },
      { id: 3, name: "lamp" },
      { id: 4, name: "lamp_on", item: 3 },
      { id: 5, name: "script_block" },
      { id: 6, name: "my_flower", pixels: new Uint8Array(4) },
    ];
    expect(scriptBlockNames(blocks)).toEqual(["stone", "lamp", "air"]);
  });
});
