import { CLEAR, type PixelImage, type RGBA } from "../editor/pixels";
import type { Rect } from "../render/skin";

export type Tool = "pen" | "fill" | "eraser" | "picker";

const MAX_SIDE = 288; // CSS pixels
const MAX_ZOOM = 40;

export interface PaintHandlers {
  tool(): Tool;
  color(): RGBA;
  beforeStroke(): void;
  picked(c: RGBA): void;
  changed(): void;
}

/** A zoomed view of one rectangle of a picture that paints where it's touched. */
export class PixelCanvas {
  readonly node = document.createElement("canvas");
  private rect: Rect = { x: 0, y: 0, w: 1, h: 1 };
  private last?: [number, number];

  constructor(
    private readonly image: PixelImage,
    private readonly on: PaintHandlers,
  ) {
    this.node.className = "pixels";
    this.node.addEventListener("pointerdown", (e) => this.down(e));
    this.node.addEventListener("pointermove", (e) => this.move(e));
    for (const type of ["pointerup", "pointercancel"])
      this.node.addEventListener(type, () => (this.last = undefined));
  }

  show(rect: Rect): void {
    this.rect = rect;
    const zoom = Math.min(MAX_ZOOM, Math.floor(MAX_SIDE / Math.max(rect.w, rect.h)));
    this.node.width = rect.w * zoom;
    this.node.height = rect.h * zoom;
    this.draw();
  }

  draw(): void {
    const ctx = this.node.getContext("2d");
    if (!ctx) return;
    const { x, y, w, h } = this.rect;
    const zoom = this.node.width / w;
    for (let py = 0; py < h; py++)
      for (let px = 0; px < w; px++) {
        const [r, g, b, a] = this.image.get(x + px, y + py);
        const checker = (px + py) % 2 ? "#9aa3b5" : "#c7cdd8"; // shows see-through pixels
        ctx.fillStyle = a ? `rgb(${r} ${g} ${b})` : checker;
        ctx.fillRect(px * zoom, py * zoom, zoom, zoom);
      }
    ctx.strokeStyle = "rgb(0 0 0 / 15%)";
    ctx.beginPath();
    for (let i = 1; i < w; i++) ctx.rect(i * zoom, 0, 0, h * zoom);
    for (let i = 1; i < h; i++) ctx.rect(0, i * zoom, w * zoom, 0);
    ctx.stroke();
  }

  private pixelAt(e: PointerEvent): [number, number] {
    const box = this.node.getBoundingClientRect();
    const { x, y, w, h } = this.rect;
    const px = Math.floor(((e.clientX - box.left) / box.width) * w);
    const py = Math.floor(((e.clientY - box.top) / box.height) * h);
    return [x + Math.max(0, Math.min(w - 1, px)), y + Math.max(0, Math.min(h - 1, py))];
  }

  private down(e: PointerEvent): void {
    e.preventDefault();
    this.node.setPointerCapture(e.pointerId);
    const [x, y] = this.pixelAt(e);
    const tool = this.on.tool();
    if (tool === "picker") return this.on.picked(this.image.get(x, y));
    this.on.beforeStroke();
    if (tool === "fill") {
      if (this.image.fill(x, y, this.on.color(), this.rect)) this.on.changed();
      return;
    }
    this.last = [x, y];
    this.paint(x, y);
  }

  private move(e: PointerEvent): void {
    if (!this.last) return;
    const [x, y] = this.pixelAt(e);
    const [lx, ly] = this.last;
    const steps = Math.max(Math.abs(x - lx), Math.abs(y - ly));
    // Fill in the pixels between pointer events so quick strokes stay unbroken.
    for (let i = 1; i <= steps; i++)
      this.paint(Math.round(lx + ((x - lx) * i) / steps), Math.round(ly + ((y - ly) * i) / steps));
    this.last = [x, y];
  }

  private paint(x: number, y: number): void {
    const c = this.on.tool() === "eraser" ? CLEAR : this.on.color();
    if (this.image.set(x, y, c)) this.on.changed();
  }
}
