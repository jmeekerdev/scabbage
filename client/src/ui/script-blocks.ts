// Visual blocks that write a script block's Lua; loaded only when someone opens the block editor.
import * as Blockly from "blockly/core";
import "blockly/blocks";
import { luaGenerator, Order } from "blockly/lua";
import * as En from "blockly/msg/en";

Blockly.setLocale(En as unknown as Record<string, string>);

const EVENTS = 45;
const WORLD = 200;
const PLAYERS = 290;

let blockNames = ["stone"];

/** Sets which blocks scripts may place or give, from the blocks this server has. */
export function setBlockNames(names: readonly string[]): void {
  if (names.length > 0) blockNames = [...names];
}

const placeMenu = (): [string, string][] =>
  blockNames.map((n) => [n === "air" ? "nothing (air)" : n.replaceAll("_", " "), n]);
const giveMenu = () => placeMenu().filter(([, n]) => n !== "air");

const statement = { previousStatement: null, nextStatement: null, inputsInline: true };
const offsets = () =>
  ["X", "Y", "Z"].map((name) => ({ type: "input_value", name, check: "Number" }));

const event = (type: string, text: string, tooltip: string) => ({
  type,
  message0: `${text} %1 %2`,
  args0: [{ type: "input_dummy" }, { type: "input_statement", name: "DO" }],
  colour: EVENTS,
  tooltip,
});

const value = (type: string, text: string, output: string, tooltip: string) => ({
  type,
  message0: text,
  output,
  colour: PLAYERS,
  tooltip,
});

Blockly.common.defineBlocksWithJsonArray([
  event("sc_on_touch", "when someone uses this block", "Runs when a player uses the script block."),
  event("sc_on_power", "when power changes", "Runs when power reaches the script block, or stops."),
  event(
    "sc_on_round_start",
    "when a round starts",
    "Runs when a minigame round starts in this arena.",
  ),
  event("sc_on_round_end", "when a round ends", "Runs when this arena's round ends."),
  value("sc_player", "player who used it", "String", "Who used the script block."),
  value(
    "sc_power_on",
    "power is on",
    "Boolean",
    "True when power just arrived, false when it stopped.",
  ),
  value("sc_players", "players in the round", "Array", "Everyone playing the round that started."),
  value("sc_winners", "winners of the round", "Array", "Whoever won the round that ended."),
  {
    type: "sc_say",
    message0: "say %1",
    args0: [{ type: "input_value", name: "TEXT" }],
    ...statement,
    colour: WORLD,
    tooltip: "Everyone nearby sees this in chat.",
  },
  {
    type: "sc_set_block",
    message0: "place %1 at %2 east %3 up %4 south of this block",
    args0: [{ type: "field_dropdown", name: "BLOCK", options: placeMenu }, ...offsets()],
    ...statement,
    colour: WORLD,
    tooltip: "Places a block near the script block. Use minus numbers for west, down and north.",
  },
  {
    type: "sc_give_item",
    message0: "give %1 %2 of %3",
    args0: [
      { type: "input_value", name: "PLAYER", check: "String" },
      { type: "input_value", name: "COUNT", check: "Number" },
      { type: "field_dropdown", name: "ITEM", options: giveMenu },
    ],
    ...statement,
    colour: PLAYERS,
    tooltip: "Hands a nearby player some blocks.",
  },
  {
    type: "sc_teleport",
    message0: "teleport %1 to %2 east %3 up %4 south of this block",
    args0: [{ type: "input_value", name: "PLAYER", check: "String" }, ...offsets()],
    ...statement,
    colour: PLAYERS,
    tooltip: "Moves a nearby player.",
  },
  {
    type: "sc_add_points",
    message0: "give %1 %2 points",
    args0: [
      { type: "input_value", name: "PLAYER", check: "String" },
      { type: "input_value", name: "POINTS", check: "Number" },
    ],
    ...statement,
    colour: PLAYERS,
    tooltip: "Scores points in this arena's round (from -10 to 10).",
  },
  {
    type: "sc_after",
    message0: "after %1 seconds %2 %3",
    args0: [
      { type: "input_value", name: "SECONDS", check: "Number" },
      { type: "input_dummy" },
      { type: "input_statement", name: "DO" },
    ],
    ...statement,
    colour: WORLD,
    tooltip: "Waits, then does the blocks inside.",
  },
]);

type Gen = typeof luaGenerator;
const arg = (b: Blockly.Block, g: Gen, name: string, order: Order, empty: string) =>
  g.valueToCode(b, name, order) || empty;
const spot = (b: Blockly.Block, g: Gen) =>
  ["X", "Y", "Z"].map((n) => arg(b, g, n, Order.ADDITIVE, "0"));

const handler = (fn: string, param: string) => (b: Blockly.Block, g: Gen) =>
  `function ${fn}(${param})\n${g.statementToCode(b, "DO")}end\n`;
const name =
  (param: string, order = Order.ATOMIC) =>
  (): [string, Order] => [param, order];

Object.assign(luaGenerator.forBlock, {
  sc_on_touch: handler("on_touch", "player"),
  sc_on_power: handler("on_power", "on"),
  sc_on_round_start: handler("on_round_start", "players"),
  sc_on_round_end: handler("on_round_end", "winners"),
  sc_player: name("player"),
  sc_power_on: name("on"),
  sc_players: name("players"),
  sc_winners: name("winners"),
  sc_say: (b: Blockly.Block, g: Gen) => `say(tostring(${arg(b, g, "TEXT", Order.NONE, "''")}))\n`,
  sc_set_block: (b: Blockly.Block, g: Gen) => {
    const [x, y, z] = spot(b, g);
    return `set_block(here.x + ${x}, here.y + ${y}, here.z + ${z}, ${g.quote_(b.getFieldValue("BLOCK"))})\n`;
  },
  sc_give_item: (b: Blockly.Block, g: Gen) =>
    `give_item(${arg(b, g, "PLAYER", Order.NONE, "player")}, ${g.quote_(b.getFieldValue("ITEM"))}, ${arg(b, g, "COUNT", Order.NONE, "1")})\n`,
  sc_teleport: (b: Blockly.Block, g: Gen) => {
    const [x, y, z] = spot(b, g);
    const who = arg(b, g, "PLAYER", Order.NONE, "player");
    return `teleport(${who}, here.x + 0.5 + ${x}, here.y + ${y}, here.z + 0.5 + ${z})\n`;
  },
  sc_add_points: (b: Blockly.Block, g: Gen) =>
    `add_points(${arg(b, g, "PLAYER", Order.NONE, "player")}, ${arg(b, g, "POINTS", Order.NONE, "1")})\n`,
  sc_after: (b: Blockly.Block, g: Gen) =>
    `after(${arg(b, g, "SECONDS", Order.NONE, "1")}, function()\n${g.statementToCode(b, "DO")}end)\n`,
});

// Players' variables and functions must not take the names scripts already use.
luaGenerator.addReservedWords(
  "player,on,players,winners,here,say,set_block,give_item,teleport,add_points,after," +
    "on_touch,on_power,on_round_start,on_round_end",
);

/** A new script: say hello to whoever uses the block. */
export const STARTER = {
  blocks: {
    languageVersion: 0,
    blocks: [
      {
        type: "sc_on_touch",
        x: 20,
        y: 20,
        inputs: {
          DO: {
            block: {
              type: "sc_say",
              inputs: {
                TEXT: {
                  block: {
                    type: "text_join",
                    extraState: { itemCount: 2 },
                    inputs: {
                      ADD0: { block: { type: "text", fields: { TEXT: "Hello, " } } },
                      ADD1: { block: { type: "sc_player" } },
                    },
                  },
                },
              },
            },
          },
        },
      },
    ],
  },
};

/** The Lua that a saved set of blocks makes, worked out without drawing them. */
export function luaFor(state: object): string {
  const ws = new Blockly.Workspace();
  try {
    Blockly.serialization.workspaces.load(state, ws);
    return luaGenerator.workspaceToCode(ws);
  } finally {
    ws.dispose();
  }
}

const num = (n: number) => ({ shadow: { type: "math_number", fields: { NUM: n } } });
const text = (t: string) => ({ shadow: { type: "text", fields: { TEXT: t } } });
const who = { shadow: { type: "sc_player" } };
const near = (x: number, y: number, z: number) => ({ X: num(x), Y: num(y), Z: num(z) });
const kinds = (...types: string[]) => types.map((type) => ({ kind: "block", type }));
const category = (name: string, colour: number | string, contents: object[]) => ({
  kind: "category",
  name,
  colour: String(colour),
  contents,
});

const TOOLBOX = {
  kind: "categoryToolbox",
  contents: [
    category(
      "When",
      EVENTS,
      kinds("sc_on_touch", "sc_on_power", "sc_on_round_start", "sc_on_round_end"),
    ),
    category("World", WORLD, [
      { kind: "block", type: "sc_say", inputs: { TEXT: text("Hello!") } },
      { kind: "block", type: "sc_set_block", inputs: near(0, 1, 0) },
      { kind: "block", type: "sc_after", inputs: { SECONDS: num(1) } },
    ]),
    category("Players", PLAYERS, [
      ...kinds("sc_player", "sc_power_on", "sc_players", "sc_winners"),
      { kind: "block", type: "sc_give_item", inputs: { PLAYER: who, COUNT: num(1) } },
      { kind: "block", type: "sc_teleport", inputs: { PLAYER: who, ...near(0, 1, 0) } },
      { kind: "block", type: "sc_add_points", inputs: { PLAYER: who, POINTS: num(1) } },
    ]),
    category(
      "Logic",
      "%{BKY_LOGIC_HUE}",
      kinds("controls_if", "logic_compare", "logic_operation", "logic_negate", "logic_boolean"),
    ),
    category("Loops", "%{BKY_LOOPS_HUE}", [
      { kind: "block", type: "controls_repeat_ext", inputs: { TIMES: num(3) } },
      { kind: "block", type: "controls_whileUntil" },
      { kind: "block", type: "controls_for", inputs: { FROM: num(1), TO: num(10), BY: num(1) } },
      { kind: "block", type: "controls_forEach" },
    ]),
    category("Math", "%{BKY_MATH_HUE}", [
      { kind: "block", type: "math_number" },
      { kind: "block", type: "math_arithmetic", inputs: { A: num(1), B: num(1) } },
      { kind: "block", type: "math_random_int", inputs: { FROM: num(1), TO: num(6) } },
      { kind: "block", type: "math_modulo", inputs: { DIVIDEND: num(10), DIVISOR: num(3) } },
    ]),
    category("Text", "%{BKY_TEXTS_HUE}", kinds("text", "text_join", "text_length")),
    category(
      "Lists",
      "%{BKY_LISTS_HUE}",
      kinds("lists_create_with", "lists_length", "lists_getIndex"),
    ),
    { kind: "category", name: "Variables", colour: "%{BKY_VARIABLES_HUE}", custom: "VARIABLE" },
    { kind: "category", name: "Functions", colour: "%{BKY_PROCEDURES_HUE}", custom: "PROCEDURE" },
  ],
};

const THEME = Blockly.Theme.defineTheme("scabbage", {
  name: "scabbage",
  base: Blockly.Themes.Classic,
  startHats: true,
  componentStyles: {
    workspaceBackgroundColour: "#0f1420",
    toolboxBackgroundColour: "#1a2133",
    toolboxForegroundColour: "#e8f0ff",
    flyoutBackgroundColour: "#232c42",
    flyoutForegroundColour: "#e8f0ff",
    flyoutOpacity: 1,
    scrollbarColour: "#5a6580",
    insertionMarkerColour: "#ffffff",
  },
});

// Vite serves the package in development; builds copy its pictures and sounds into assets.
const MEDIA = import.meta.env.DEV ? "/node_modules/blockly/media/" : "/assets/blockly/";

/** Blocks without their ids, which Blockly makes afresh, so saved scripts stay small. */
export function compact(state: object): object {
  return JSON.parse(
    JSON.stringify(state),
    function (this: Record<string, unknown>, key, v: unknown) {
      return key === "id" && typeof this.type === "string" && this.type !== "" ? undefined : v;
    },
  ) as object;
}

/** A block editor drawn into host, whose menus open inside it too, so they work in a modal dialog. */
export class BlockArea {
  private readonly ws: Blockly.WorkspaceSvg;

  constructor(host: HTMLElement, state: object | undefined, readOnly: boolean) {
    this.ws = Blockly.inject(host, {
      toolbox: readOnly ? undefined : TOOLBOX,
      readOnly,
      theme: THEME,
      media: MEDIA,
      renderer: "zelos",
      trashcan: !readOnly,
      zoom: {
        controls: true,
        wheel: true,
        startScale: matchMedia("(pointer: coarse)").matches ? 1 : 0.75,
      },
      move: { scrollbars: true, drag: true, wheel: true },
    });
    // Blocks outside any "when" block are greyed out rather than run once on saving.
    this.ws.addChangeListener(Blockly.Events.disableOrphans);
    Blockly.serialization.workspaces.load(state ?? STARTER, this.ws);
  }

  lua(): string {
    return luaGenerator.workspaceToCode(this.ws);
  }

  state(): object {
    return compact(Blockly.serialization.workspaces.save(this.ws));
  }

  /** Fits the editor to its box after the box is shown or resized. */
  resize(): void {
    Blockly.svgResize(this.ws);
  }

  dispose(): void {
    this.ws.dispose();
  }
}
