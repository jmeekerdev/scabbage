import { describe, expect, it } from "vitest";
import { STARTER, compact, luaFor, setBlockNames } from "./script-blocks";

const num = (n: number) => ({ shadow: { type: "math_number", fields: { NUM: n } } });
const workspace = (...blocks: object[]) => ({ blocks: { languageVersion: 0, blocks } });

describe("script blocks", () => {
  it("start with a greeting when used", () => {
    const lua = luaFor(STARTER);
    expect(lua).toContain("function on_touch(player)");
    expect(lua).toContain("say(tostring('Hello, ' .. player))");
  });

  it("place blocks next to the script block", () => {
    setBlockNames(["stone", "lamp"]);
    const place = {
      type: "sc_set_block",
      fields: { BLOCK: "lamp" },
      inputs: { X: num(0), Y: num(1), Z: num(-2) },
    };
    const lua = luaFor(workspace({ type: "sc_on_power", inputs: { DO: { block: place } } }));
    expect(lua).toContain("function on_power(on)");
    expect(lua).toContain("set_block(here.x + 0, here.y + 1, here.z + -2, 'lamp')");
  });

  it("wait before doing things", () => {
    const say = {
      type: "sc_say",
      inputs: { TEXT: { shadow: { type: "text", fields: { TEXT: "ding" } } } },
    };
    const after = { type: "sc_after", inputs: { SECONDS: num(2), DO: { block: say } } };
    const lua = luaFor(workspace({ type: "sc_on_touch", inputs: { DO: { block: after } } }));
    expect(lua).toMatch(/after\(2, function\(\)\n\s+say\(tostring\('ding'\)\)\n\s+end\)/);
  });

  it("keep players' own variables away from the script's names", () => {
    const state = {
      ...workspace({
        type: "variables_set",
        fields: { VAR: { id: "v" } },
        inputs: { VALUE: num(1) },
      }),
      variables: [{ name: "player", id: "v" }],
    };
    expect(luaFor(state)).not.toMatch(/^player = 1/m);
  });

  it("score points in rounds", () => {
    const points = {
      type: "sc_add_points",
      inputs: { PLAYER: { block: { type: "sc_player" } }, POINTS: num(3) },
    };
    const lua = luaFor(workspace({ type: "sc_on_touch", inputs: { DO: { block: points } } }));
    expect(lua).toContain("add_points(player, 3)");
  });
});

describe("compact", () => {
  it("drops block ids but keeps variables'", () => {
    const state = {
      ...workspace({ type: "variables_set", id: "abc", fields: { VAR: { id: "v" } } }),
      variables: [{ name: "count", id: "v", type: "" }],
    };
    const small = compact(state) as typeof state;
    expect(JSON.stringify(small)).not.toContain('"abc"');
    expect(small.variables[0].id).toBe("v");
    expect(luaFor(small)).toBe(luaFor(state));
  });
});
