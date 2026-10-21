import type { ScriptShown } from "../pb/scabbage/v1/net_pb";
import { el } from "./dom";
import { scriptStatus } from "./script-status";

type Pos = { x: number; y: number; z: number };

const EXAMPLE = `-- Runs when someone uses this block.
function on_touch(player)
  say("Hello, " .. player .. "!")
end

-- Runs when power reaches this block, or stops.
function on_power(on)
  if on then say("Power on!") end
end
`;

const HELP = `on_touch(player)   someone used this block
on_power(on)       power arrived (true) or stopped (false)
on_round_start(players)   a minigame round started in this arena
on_round_end(winners)     the round ended
say(text)          everyone nearby sees text
set_block(x, y, z, "stone")   places a block nearby ("air" clears it)
give_item(player, "torch", 3) hands someone nearby some blocks
teleport(player, x, y, z)     moves someone nearby
add_points(player, 2)         scores points in this arena's round
after(seconds, function() ... end)   runs something later`;

/** A dialog for reading and writing the Lua in one script block. */
export class ScriptEditor {
  private readonly text = el("textarea", {
    spellcheck: false,
    autocapitalize: "off",
    autocomplete: "off",
    "aria-label": "Script",
  });
  private readonly status = el("p", { class: "status", "aria-live": "polite" });
  private readonly save = el("button", { type: "button" }, "Save");
  private readonly dialog = el(
    "dialog",
    { class: "card script-editor" },
    el("h1", {}, "📜 Script block"),
    this.text,
    this.status,
    el("details", {}, el("summary", {}, "What can scripts do?"), el("pre", {}, HELP)),
    el(
      "div",
      { class: "actions" },
      this.save,
      el("button", { type: "button", onclick: () => this.dialog.close() }, "Close"),
    ),
  );
  private at?: Pos;

  constructor(onSave: (at: Pos, source: string) => void, onClose: () => void) {
    document.body.append(this.dialog);
    this.save.addEventListener("click", () => this.at && onSave(this.at, this.text.value));
    this.text.addEventListener("keydown", (e) => this.key(e));
    this.dialog.addEventListener("close", onClose);
  }

  /** Opens the script the server sent, or just updates the status after a save. */
  show(s: ScriptShown): void {
    const at = { x: s.pos?.x ?? 0, y: s.pos?.y ?? 0, z: s.pos?.z ?? 0 };
    if (!s.saved || !this.dialog.open) {
      this.at = at;
      this.text.value = s.source || (s.editable ? EXAMPLE : "");
      this.text.readOnly = !s.editable;
      this.save.hidden = !s.editable;
    }
    const { text, ok } = scriptStatus(s);
    this.status.textContent = text;
    this.status.classList.toggle("problem", !ok);
    if (!this.dialog.open) {
      if (document.pointerLockElement) document.exitPointerLock();
      this.dialog.showModal();
    }
  }

  /** Tab indents instead of leaving the box; keys never reach the game. */
  private key(e: KeyboardEvent): void {
    e.stopPropagation();
    if (e.key !== "Tab" || this.text.readOnly) return;
    e.preventDefault();
    this.text.setRangeText("  ", this.text.selectionStart, this.text.selectionEnd, "end");
  }
}
