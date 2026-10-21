import { describe, expect, it } from "vitest";
import { joinBlocks, splitBlocks } from "./script-source";

describe("script source with blocks", () => {
  it("keeps the blocks after the Lua and reads them back", () => {
    const lua = 'say("hi")\n';
    const blocks = { blocks: { languageVersion: 0, blocks: [{ type: "say" }] } };
    const source = joinBlocks(lua, blocks);
    expect(source.startsWith(lua)).toBe(true);
    expect(splitBlocks(source)).toEqual({ lua, blocks });
  });

  it("leaves plain Lua alone", () => {
    expect(splitBlocks('say("hi")')).toEqual({ lua: 'say("hi")' });
  });

  it("picks a comment the blocks can't close early", () => {
    const blocks = { text: "]] ]=] ]==]" };
    expect(splitBlocks(joinBlocks("", blocks)).blocks).toEqual(blocks);
  });

  it("treats broken block data as plain Lua", () => {
    const source = "x = 1\n--[[scabbage-blocks\n{oops\n]]\n";
    expect(splitBlocks(source)).toEqual({ lua: source });
  });
});
