import { DoubleTap, HoldRepeat, type Intent } from "../player/input";
import type { Player } from "../player/player";
import { TouchSticks } from "../player/touch-sticks";
import { el } from "./dom";

const STICK_RADIUS = 56; // px
const LOOK_SCALE = 2; // touch drags turn faster than the mouse
const FLY_DOUBLE_TAP_MS = 300;
const BUILD_REPEAT_MS = 250;

/** On-screen controls: drag left to walk, drag right to look, buttons to jump, fly and build. */
export class TouchControls {
  private readonly sticks = new TouchSticks(innerWidth, STICK_RADIUS);
  private readonly flyTap = new DoubleTap(FLY_DOUBLE_TAP_MS);
  private readonly breaking = new HoldRepeat(BUILD_REPEAT_MS);
  private readonly placing = new HoldRepeat(BUILD_REPEAT_MS);
  private readonly knob = el("div", { class: "knob" });
  private readonly stick = el("div", { class: "stick", hidden: true }, this.knob);
  private readonly down: HTMLButtonElement;
  private jumping = false;
  private descending = false;

  constructor(
    canvas: HTMLCanvasElement,
    root: HTMLElement,
    private readonly player: Player,
  ) {
    this.down = this.button("down", "⬇", (on) => (this.descending = on));
    root.append(
      this.stick,
      el(
        "div",
        { class: "touch-buttons" },
        this.button("break", "⛏", (on) => this.hold(this.breaking, on)),
        this.button("place", "🧱", (on) => this.hold(this.placing, on)),
        this.down,
        this.button("jump", "⬆", (on) => this.jump(on)),
      ),
    );
    this.listen(canvas);
    new ResizeObserver(() => (this.sticks.width = canvas.clientWidth)).observe(canvas);
  }

  poll(nowMs: number): Intent {
    const [dx, dy] = this.sticks.takeLook();
    this.player.look.turn(dx * LOOK_SCALE, dy * LOOK_SCALE);
    this.drawStick();
    this.down.hidden = !this.player.body.flying;
    return {
      move: { ...this.sticks.walk(), jump: this.jumping, descend: this.descending },
      build: {
        breaking: this.breaking.due(nowMs),
        placing: this.placing.due(nowMs),
        picking: false,
      },
    };
  }

  private listen(canvas: HTMLCanvasElement): void {
    const touch = (f: (e: PointerEvent) => void) => (e: PointerEvent) => {
      if (e.pointerType === "touch") f(e);
    };
    canvas.addEventListener(
      "pointerdown",
      touch((e) => this.sticks.down(e.pointerId, e.clientX, e.clientY)),
    );
    canvas.addEventListener(
      "pointermove",
      touch((e) => this.sticks.move(e.pointerId, e.clientX, e.clientY)),
    );
    for (const end of ["pointerup", "pointercancel"] as const)
      canvas.addEventListener(
        end,
        touch((e) => this.sticks.up(e.pointerId)),
      );
  }

  /** A button that reports presses and releases; a finger sliding off still counts as held. */
  private button(name: string, label: string, held: (on: boolean) => void): HTMLButtonElement {
    const b = el("button", { class: name, "aria-label": name }, label);
    b.addEventListener("pointerdown", (e) => {
      e.preventDefault();
      held(true);
    });
    for (const end of ["pointerup", "pointercancel"] as const)
      b.addEventListener(end, () => held(false));
    return b;
  }

  private jump(on: boolean): void {
    this.jumping = on;
    if (on && this.flyTap.press(performance.now())) this.player.toggleFly();
  }

  private hold(r: HoldRepeat, on: boolean): void {
    if (on) r.press(performance.now());
    else r.release();
  }

  private drawStick(): void {
    const s = this.sticks.joystick();
    this.stick.hidden = !s;
    if (!s) return;
    this.stick.style.transform = `translate(${s.x}px, ${s.y}px)`;
    this.knob.style.transform = `translate(${s.dx}px, ${s.dy}px)`;
  }
}
