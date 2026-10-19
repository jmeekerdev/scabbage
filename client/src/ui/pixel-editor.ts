import { parseHex, PixelImage, SWATCHES, Undo, hexColor, type RGBA } from "../editor/pixels";
import { ArtKind } from "../pb/scabbage/v1/net_pb";
import { SkinPreview } from "../render/skin-preview";
import { faceRect, FACES, SKIN_H, SKIN_W, type Face, type Part } from "../render/skin";
import { TEXTURE_SIZE } from "../render/textures";
import { el } from "./dom";
import { PixelCanvas, type Tool } from "./pixel-canvas";

export type EditorMode =
  | { kind: "skin"; pixels: Uint8Array; color: number }
  | { kind: "block" }
  | { kind: "texture"; names: readonly string[]; read(name: string): Uint8Array | undefined };

export interface Submission {
  kind: ArtKind;
  name: string;
  pixels: Uint8Array; // empty resets a repainted texture
}

const PARTS: [Part, string][] = [
  ["head", "Head"],
  ["body", "Body"],
  ["arm", "Arms"],
  ["leg", "Legs"],
];
const TOOLS: [Tool, string, string][] = [
  ["pen", "✏️", "Draw"],
  ["fill", "🪣", "Fill"],
  ["eraser", "🧽", "Erase"],
  ["picker", "💧", "Pick a color from the picture"],
];
const TITLES = { skin: "Make your look", block: "Make a new block", texture: "Repaint a block" };
const BLANK_BLOCK = parseHex("#ffffff");

/** Opens a modal pixel editor; `submit` gets the finished art. */
export function openPixelEditor(
  mode: EditorMode,
  admin: boolean,
  submit: (s: Submission) => void,
): void {
  const editor = new PixelEditor(mode, admin, submit);
  document.body.append(editor.dialog);
  editor.dialog.showModal();
}

class PixelEditor {
  readonly dialog = el("dialog", { class: "card editor" });
  private readonly image: PixelImage;
  private readonly undo: Undo;
  private readonly canvas: PixelCanvas;
  private readonly preview?: SkinPreview;
  private readonly tiles = el("canvas", { class: "tiles", width: 48, height: 48 });
  private readonly colorInput = el("input", { type: "color", "aria-label": "Any color" });
  private readonly toolButtons = new Map<Tool, HTMLButtonElement>();
  private tool: Tool = "pen";
  private color: RGBA = parseHex(SWATCHES[0] ?? "#000000");
  private part: Part = "head";
  private face: Face = "front";
  private textureName = "";

  constructor(
    private readonly mode: EditorMode,
    admin: boolean,
    private readonly submit: (s: Submission) => void,
  ) {
    this.image = this.startingImage();
    this.undo = new Undo(this.image);
    this.canvas = new PixelCanvas(this.image, {
      tool: () => this.tool,
      color: () => this.color,
      beforeStroke: () => this.undo.save(),
      picked: (c) => this.picked(c),
      changed: () => this.changed(),
    });
    if (mode.kind === "skin") this.preview = new SkinPreview(this.image.data, mode.color);
    this.colorInput.value = hexColor(this.color);
    this.colorInput.addEventListener("input", () => this.pick(parseHex(this.colorInput.value)));
    this.dialog.addEventListener("close", () => this.dispose());
    this.dialog.append(
      el("h1", {}, TITLES[mode.kind]),
      ...this.chooser(),
      el("div", { class: "easel" }, this.canvas.node, this.preview?.canvas ?? this.tiles),
      this.toolbar(),
      this.palette(),
      this.actions(admin),
    );
    this.showView();
  }

  private startingImage(): PixelImage {
    const m = this.mode;
    if (m.kind === "skin") return new PixelImage(SKIN_W, SKIN_H, m.pixels);
    const img = new PixelImage(TEXTURE_SIZE, TEXTURE_SIZE);
    if (m.kind === "block")
      img.fill(0, 0, BLANK_BLOCK, { x: 0, y: 0, w: TEXTURE_SIZE, h: TEXTURE_SIZE });
    return img;
  }

  /** Part and face tabs for skins, a name box for blocks, a texture list for repaints. */
  private chooser(): HTMLElement[] {
    const m = this.mode;
    if (m.kind === "skin")
      return [
        tabs(
          PARTS,
          () => this.part,
          (p) => ((this.part = p), this.showView()),
        ),
        tabs(
          FACES.map((f) => [f, f[0]?.toUpperCase() + f.slice(1)] as [Face, string]),
          () => this.face,
          (f) => ((this.face = f), this.showView()),
        ),
      ];
    if (m.kind === "block") return [];
    const names = m.names.filter((n) => n !== "missing");
    const select = el(
      "select",
      {},
      ...names.map((n) => el("option", { value: n }, n.replace(/_/g, " "))),
    );
    select.addEventListener("change", () => this.loadTexture(select.value));
    this.loadTexture(names[0] ?? "");
    return [el("label", {}, "Block texture", select)];
  }

  private loadTexture(name: string): void {
    if (this.mode.kind !== "texture") return;
    this.textureName = name;
    const px = this.mode.read(name);
    if (px) this.image.data.set(px);
    this.canvas?.draw();
    this.drawTiles();
  }

  private toolbar(): HTMLElement {
    const tools = TOOLS.filter(([t]) => t !== "eraser" || this.mode.kind !== "skin");
    const buttons = tools.map(([tool, icon, label]) => {
      const b = el(
        "button",
        { class: "tool", title: label, "aria-label": label, onclick: () => this.useTool(tool) },
        icon,
      );
      this.toolButtons.set(tool, b);
      return b;
    });
    const undo = el(
      "button",
      { class: "tool", title: "Undo", "aria-label": "Undo", onclick: () => this.back() },
      "↩️",
    );
    this.useTool("pen");
    return el("div", { class: "tools" }, ...buttons, undo);
  }

  private palette(): HTMLElement {
    const swatches = SWATCHES.map((hex) =>
      el("button", {
        class: "swatch",
        style: `--swatch: ${hex}`,
        "aria-label": hex,
        onclick: () => this.pick(parseHex(hex)),
      }),
    );
    return el("div", { class: "palette" }, ...swatches, this.colorInput);
  }

  private actions(admin: boolean): HTMLElement {
    const send = el(
      "button",
      { class: "primary", onclick: () => this.send() },
      admin ? "Save" : "Send to an admin",
    );
    const cancel = el("button", { onclick: () => this.dialog.close() }, "Cancel");
    if (this.mode.kind === "block") {
      const name = el("input", {
        maxlength: 20,
        placeholder: "Block name",
        "aria-label": "Block name",
      });
      const sync = () => (send.disabled = name.value.trim() === "");
      name.addEventListener("input", sync);
      sync();
      send.onclick = () => this.send(name.value.trim());
      return el("div", { class: "actions" }, name, cancel, send);
    }
    if (this.mode.kind === "texture") {
      const reset = el(
        "button",
        { onclick: () => this.send(this.textureName, true) },
        "Put back original",
      );
      return el("div", { class: "actions" }, reset, cancel, send);
    }
    const original = el(
      "button",
      { onclick: () => this.send("", true) },
      "Back to my original look",
    );
    return el("div", { class: "actions" }, original, cancel, send);
  }

  private showView(): void {
    const full = { x: 0, y: 0, w: this.image.width, h: this.image.height };
    this.canvas.show(this.mode.kind === "skin" ? faceRect(this.part, this.face) : full);
    this.drawTiles();
  }

  private useTool(tool: Tool): void {
    this.tool = tool;
    for (const [t, b] of this.toolButtons) b.classList.toggle("selected", t === tool);
  }

  private pick(c: RGBA): void {
    this.color = c;
    this.colorInput.value = hexColor(c);
    if (this.tool === "picker" || this.tool === "eraser") this.useTool("pen");
  }

  /** A see-through pixel picks the eraser, where there is one. */
  private picked(c: RGBA): void {
    if (c[3]) this.pick(c);
    else if (this.toolButtons.has("eraser")) this.useTool("eraser");
  }

  private back(): void {
    if (this.undo.back()) this.changed();
  }

  private changed(): void {
    this.canvas.draw();
    this.preview?.refresh();
    this.drawTiles();
  }

  /** Blocks are previewed as a 3×3 patch, so seams between neighbors show. */
  private drawTiles(): void {
    if (this.mode.kind === "skin") return;
    const ctx = this.tiles.getContext("2d");
    if (!ctx) return;
    const img = new ImageData(new Uint8ClampedArray(this.image.data), TEXTURE_SIZE, TEXTURE_SIZE);
    ctx.clearRect(0, 0, 48, 48);
    for (let i = 0; i < 9; i++) ctx.putImageData(img, (i % 3) * 16, Math.floor(i / 3) * 16);
  }

  private send(name = "", reset = false): void {
    const kinds = { skin: ArtKind.SKIN, block: ArtKind.BLOCK, texture: ArtKind.TEXTURE };
    const kind = kinds[this.mode.kind];
    this.submit({
      kind,
      name: this.mode.kind === "texture" ? this.textureName : name,
      pixels: reset ? new Uint8Array() : this.image.data.slice(),
    });
    this.dialog.close();
  }

  private dispose(): void {
    this.preview?.dispose();
    this.dialog.remove();
  }
}

/** A row of toggle buttons, one of them selected. */
function tabs<T extends string>(
  items: [T, string][],
  current: () => T,
  choose: (v: T) => void,
): HTMLElement {
  const row = el("div", { class: "tabs" });
  const draw = () =>
    row.replaceChildren(
      ...items.map(([value, label]) =>
        el(
          "button",
          { class: value === current() ? "selected" : "", onclick: () => (choose(value), draw()) },
          label,
        ),
      ),
    );
  draw();
  return row;
}
