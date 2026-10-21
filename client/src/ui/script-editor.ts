import type { ScriptShown } from "../pb/scabbage/v1/net_pb";
import { el } from "./dom";
import type { BlockArea } from "./script-blocks";
import { joinBlocks, splitBlocks } from "./script-source";
import { scriptStatus } from "./script-status";

type Pos = { x: number; y: number; z: number };
type Mode = "blocks" | "lua";

const HELP = `on_touch(player)   someone used this block
on_power(on)       power arrived (true) or stopped (false)
on_round_start(players)   a minigame round started in this arena
on_round_end(winners)     the round ended
here.x, here.y, here.z    where this block is
say(text)          everyone nearby sees text
set_block(x, y, z, "stone")   places a block nearby ("air" clears it)
give_item(player, "torch", 3) hands someone nearby some blocks
teleport(player, x, y, z)     moves someone nearby
add_points(player, 2)         scores points in this arena's round
after(seconds, function() ... end)   runs something later`;

const LOSE_EDITS = "Go back to the blocks? Changes you made to the Lua will be lost.";
const START_OVER =
  "Blocks can't be made from Lua, so this starts over with blocks. Your Lua is only replaced if you save. Start over?";

// Blockly is big, so it loads the first time someone opens the block editor.
let blockly: Promise<typeof import("./script-blocks")> | undefined;
const loadBlockly = () => (blockly ??= import("./script-blocks"));

/** A dialog for building a script block's script from blocks, or reading and writing its Lua. */
export class ScriptEditor {
  private readonly text = el("textarea", {
    spellcheck: false,
    autocapitalize: "off",
    autocomplete: "off",
    "aria-label": "Script",
  });
  private readonly blockHost = el("div", { class: "blocks" });
  private readonly blocksTab = el("button", { type: "button" }, "🧩 Blocks");
  private readonly luaTab = el("button", { type: "button" }, "⌨️ Lua");
  private readonly help = el(
    "details",
    {},
    el("summary", {}, "What can scripts do?"),
    el("pre", {}, HELP),
  );
  private readonly status = el("p", { class: "status", "aria-live": "polite" });
  private readonly save = el("button", { type: "button" }, "Save");
  private readonly dialog = el(
    "dialog",
    { class: "card script-editor" },
    el("h1", {}, "📜 Script block"),
    el("div", { class: "modes", role: "group" }, this.blocksTab, this.luaTab),
    this.blockHost,
    this.text,
    this.status,
    this.help,
    el(
      "div",
      { class: "actions" },
      this.save,
      el("button", { type: "button", onclick: () => this.dialog.close() }, "Close"),
    ),
  );
  private at?: Pos;
  private editable = false;
  private mode: Mode = "lua";
  private area?: BlockArea;
  private fromBlocks?: string; // the Lua last made from the blocks, to spot edits to it
  private opening = 0; // bumped on each open, so a slow load never lands in a later script
  private blockNames: string[] = [];

  constructor(onSave: (at: Pos, source: string) => void, onClose: () => void) {
    document.body.append(this.dialog);
    this.save.addEventListener("click", () => {
      const source = this.source();
      if (this.at && source !== undefined) onSave(this.at, source);
    });
    this.blocksTab.addEventListener("click", () => this.toBlocks());
    this.luaTab.addEventListener("click", () => this.toLua());
    this.text.addEventListener("keydown", (e) => this.key(e));
    // Escape closes Blockly's menus; it must not also close the editor and lose the blocks.
    this.dialog.addEventListener("cancel", (e) => this.mode === "blocks" && e.preventDefault());
    this.dialog.addEventListener("close", () => {
      if (this.dialog.open) return; // reopened before this queued event arrived
      this.closeBlocks();
      onClose();
    });
  }

  /** Sets which blocks the place and give blocks offer. */
  setBlockNames(names: string[]): void {
    this.blockNames = names;
  }

  /** Opens the script the server sent, or just updates the status after a save. */
  show(s: ScriptShown): void {
    if (!s.saved || !this.dialog.open) {
      this.open({ x: s.pos?.x ?? 0, y: s.pos?.y ?? 0, z: s.pos?.z ?? 0 }, s.source, s.editable);
    }
    this.say(scriptStatus(s));
    if (!this.dialog.open) {
      if (document.pointerLockElement) document.exitPointerLock();
      this.dialog.showModal();
    }
  }

  /** Shows blocks if the script was made with them or is new, and Lua otherwise. */
  private open(at: Pos, source: string, editable: boolean): void {
    this.closeBlocks();
    this.at = at;
    this.editable = editable;
    const { lua, blocks } = splitBlocks(source);
    this.text.value = lua;
    this.text.readOnly = !editable;
    this.save.hidden = !editable;
    this.blocksTab.hidden = !editable && !blocks;
    if (blocks || (editable && source === "")) void this.showBlocks(blocks);
    else this.setMode("lua");
  }

  private async showBlocks(state?: object): Promise<void> {
    const ticket = ++this.opening;
    this.setMode("blocks");
    try {
      const { BlockArea, setBlockNames } = await loadBlockly();
      if (ticket !== this.opening || !this.dialog.open) return;
      setBlockNames(this.blockNames);
      this.area = new BlockArea(this.blockHost, state, !this.editable);
    } catch {
      if (ticket !== this.opening) return;
      this.closeBlocks();
      this.setMode("lua");
      this.say({ text: "The blocks couldn't be shown, so here is the Lua.", ok: false });
    }
  }

  private toLua(): void {
    if (this.mode === "lua") return;
    if (this.area) this.text.value = this.fromBlocks = this.area.lua();
    this.setMode("lua");
  }

  private toBlocks(): void {
    if (this.mode === "blocks") return;
    if (this.area) {
      if (this.editable && this.text.value !== this.fromBlocks && !confirm(LOSE_EDITS)) return;
      this.setMode("blocks");
      this.area.resize();
    } else if (this.editable && (this.text.value.trim() === "" || confirm(START_OVER))) {
      void this.showBlocks();
    }
  }

  /** What Save sends: the blocks' Lua with the blocks kept, or the typed Lua alone. */
  private source(): string | undefined {
    if (this.mode === "blocks") return this.area && joinBlocks(this.area.lua(), this.area.state());
    if (this.area && this.text.value === this.fromBlocks) {
      return joinBlocks(this.text.value, this.area.state());
    }
    return this.text.value;
  }

  private setMode(mode: Mode): void {
    this.mode = mode;
    this.blockHost.hidden = mode !== "blocks";
    this.text.hidden = this.help.hidden = mode !== "lua";
    this.blocksTab.setAttribute("aria-pressed", String(mode === "blocks"));
    this.luaTab.setAttribute("aria-pressed", String(mode === "lua"));
  }

  private closeBlocks(): void {
    this.opening++;
    this.area?.dispose();
    this.area = this.fromBlocks = undefined;
    this.blockHost.replaceChildren();
  }

  private say({ text, ok }: { text: string; ok: boolean }): void {
    this.status.textContent = text;
    this.status.classList.toggle("problem", !ok);
  }

  /** Tab indents instead of leaving the box; keys never reach the game. */
  private key(e: KeyboardEvent): void {
    e.stopPropagation();
    if (e.key !== "Tab" || this.text.readOnly) return;
    e.preventDefault();
    this.text.setRangeText("  ", this.text.selectionStart, this.text.selectionEnd, "end");
  }
}
