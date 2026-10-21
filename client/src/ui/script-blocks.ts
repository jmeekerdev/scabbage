// Visual blocks that write a script block's Lua; loaded only when someone opens the block editor.
import * as Blockly from "blockly/core";
import "blockly/blocks";
import { luaGenerator, Order } from "blockly/lua";
import * as En from "blockly/msg/en";

Blockly.setLocale(En as unknown as Record<string, string>);

const EVENTS = 45;
const WORLD = 200;
const PLAYERS = 290;
const STORY = 160;

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
  event(
    "sc_on_talk",
    "when someone talks to me",
    "Runs when a player talks to this block's character.",
  ),
  value("sc_player", "player", "String", "Who used the script block or talked to its character."),
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
  ...storyBlocks(),
]);

/** Blocks for characters and quests. */
function storyBlocks(): object[] {
  const player = { type: "input_value", name: "PLAYER", check: "String" };
  const words = (name: string, text: string) => ({ type: "field_input", name, text });
  const story = { ...statement, colour: STORY };
  return [
    {
      type: "sc_npc",
      message0: "🧑 I am a character called %1",
      args0: [words("NAME", "Granny Ivy")],
      colour: STORY,
      tooltip:
        "Stands a character on top of this block. Talking to it runs “when someone talks to me”.",
    },
    {
      type: "sc_talk",
      message0: "tell %1 %2",
      args0: [player, { type: "input_value", name: "TEXT" }],
      ...story,
      tooltip: "Shows a player what this block's character says.",
    },
    {
      type: "sc_ask",
      message0: "ask %1 %2 choices %3 %4 %5 %6 when they answer %7",
      args0: [
        player,
        { type: "input_value", name: "TEXT" },
        words("CHOICE1", "Yes"),
        words("CHOICE2", "No"),
        words("CHOICE3", ""),
        { type: "input_dummy" },
        { type: "input_statement", name: "DO" },
      ],
      ...story,
      tooltip:
        "Asks a player a question with up to three answers to pick from. Leave a choice empty to skip it.",
    },
    { ...value("sc_answer", "answer", "String", "The choice the player picked."), colour: STORY },
    {
      type: "sc_remember",
      message0: "remember about %1 that %2 is %3",
      args0: [player, words("KEY", "sheep found"), { type: "input_value", name: "VALUE" }],
      ...story,
      tooltip:
        "Keeps something about a player that any script can look up later, even after they leave.",
    },
    {
      type: "sc_recall",
      message0: "%1 about %2",
      args0: [words("KEY", "sheep found"), player],
      output: null,
      inputsInline: true,
      colour: STORY,
      tooltip: "What was remembered about a player, or nothing if it never was.",
    },
    {
      type: "sc_quest",
      message0: "give %1 the quest %2 next step %3",
      args0: [player, words("TITLE", "Lost sheep"), { type: "input_value", name: "STEP" }],
      ...story,
      tooltip: "Adds a quest to a player's quest log, or changes what it says to do next.",
    },
    {
      type: "sc_finish_quest",
      message0: "finish %1's quest %2",
      args0: [player, words("TITLE", "Lost sheep")],
      ...story,
      tooltip: "Ticks a quest off in a player's quest log.",
    },
    {
      type: "sc_take_item",
      message0: "take %1 %2 from %3",
      args0: [
        { type: "input_value", name: "COUNT", check: "Number" },
        { type: "field_dropdown", name: "ITEM", options: giveMenu },
        player,
      ],
      output: "Boolean",
      inputsInline: true,
      colour: STORY,
      tooltip: "Takes blocks from a player if they have enough, and says whether it did.",
    },
  ];
}

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
  ...storyLua(),
});

function storyLua(): Record<string, (b: Blockly.Block, g: Gen) => string | [string, Order]> {
  const who = (b: Blockly.Block, g: Gen) => arg(b, g, "PLAYER", Order.NONE, "player");
  const field = (b: Blockly.Block, g: Gen, name: string) =>
    g.quote_(b.getFieldValue(name) as string);
  const said = (b: Blockly.Block, g: Gen, name: string) =>
    `tostring(${arg(b, g, name, Order.NONE, "''")})`;
  return {
    sc_on_talk: handler("on_talk", "player"),
    sc_npc: (b, g) => `npc(${field(b, g, "NAME")})\n`,
    sc_talk: (b, g) => `talk(${who(b, g)}, ${said(b, g, "TEXT")})\n`,
    sc_ask: (b, g) => {
      const picked = ["CHOICE1", "CHOICE2", "CHOICE3"].map((n) =>
        (b.getFieldValue(n) as string).trim(),
      );
      const choices = picked.filter(Boolean).map((c) => g.quote_(c));
      const list = `{${(choices.length ? choices : ["'OK'"]).join(", ")}}`;
      return `ask(${who(b, g)}, ${said(b, g, "TEXT")}, ${list}, function(answer)\n${g.statementToCode(b, "DO")}end)\n`;
    },
    sc_answer: name("answer"),
    sc_remember: (b, g) =>
      `remember(${who(b, g)}, ${field(b, g, "KEY")}, ${arg(b, g, "VALUE", Order.NONE, "nil")})\n`,
    sc_recall: (b, g) => [`recall(${who(b, g)}, ${field(b, g, "KEY")})`, Order.HIGH],
    sc_quest: (b, g) => `quest(${who(b, g)}, ${field(b, g, "TITLE")}, ${said(b, g, "STEP")})\n`,
    sc_finish_quest: (b, g) => `finish_quest(${who(b, g)}, ${field(b, g, "TITLE")})\n`,
    sc_take_item: (b, g) => [
      `take_item(${who(b, g)}, ${field(b, g, "ITEM")}, ${arg(b, g, "COUNT", Order.NONE, "1")})`,
      Order.HIGH,
    ],
  };
}

// Players' variables and functions must not take the names scripts already use.
luaGenerator.addReservedWords(
  "player,on,players,winners,here,say,set_block,give_item,teleport,add_points,after," +
    "on_touch,on_power,on_round_start,on_round_end,on_talk,answer,npc,talk,ask,remember,recall," +
    "quest,finish_quest,take_item",
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
const logic = (on: boolean) => ({
  block: { type: "logic_boolean", fields: { BOOL: on ? "TRUE" : "FALSE" } },
});
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
      kinds("sc_on_touch", "sc_on_talk", "sc_on_power", "sc_on_round_start", "sc_on_round_end"),
    ),
    category("Story", STORY, [
      ...kinds("sc_npc", "sc_on_talk"),
      { kind: "block", type: "sc_talk", inputs: { PLAYER: who, TEXT: text("Hello there!") } },
      { kind: "block", type: "sc_ask", inputs: { PLAYER: who, TEXT: text("Will you help me?") } },
      ...kinds("sc_answer"),
      { kind: "block", type: "sc_quest", inputs: { PLAYER: who, STEP: text("Find 3 white wool") } },
      { kind: "block", type: "sc_finish_quest", inputs: { PLAYER: who } },
      { kind: "block", type: "sc_remember", inputs: { PLAYER: who, VALUE: logic(true) } },
      { kind: "block", type: "sc_recall", inputs: { PLAYER: who } },
      { kind: "block", type: "sc_take_item", inputs: { PLAYER: who, COUNT: num(3) } },
    ]),
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
