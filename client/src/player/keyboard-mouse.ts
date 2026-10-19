import type { Hotbar } from "./hotbar";
import { DoubleTap, HoldRepeat, moveInputFromKeys, NO_INPUT, type Intent } from "./input";
import type { Player } from "./player";

const FLY_DOUBLE_TAP_MS = 300;
const BUILD_REPEAT_MS = 250;

// Keys the page would otherwise use to scroll.
const CAPTURED = new Set(["Space", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"]);

/** A menu aimed by mouse movement while a key is held, like the emote wheel. */
export interface AimedMenu {
  readonly isOpen: boolean;
  open(): void;
  move(dx: number, dy: number): void;
  release(): void;
}

/** Pointer-lock mouse look, building and keyboard movement; inactive until the canvas is clicked. */
export class KeyboardMouse {
  private readonly keys = new Set<string>();
  private readonly flyTap = new DoubleTap(FLY_DOUBLE_TAP_MS);
  private readonly breaking = new HoldRepeat(BUILD_REPEAT_MS);
  private readonly placing = new HoldRepeat(BUILD_REPEAT_MS);
  private picking = false;
  private locked = false;

  constructor(
    canvas: HTMLCanvasElement,
    private readonly player: Player,
    private readonly hotbar: Hotbar,
    private readonly wheel: AimedMenu,
    onLockChange: (locked: boolean) => void,
  ) {
    canvas.addEventListener("click", (e) => {
      if ((e as PointerEvent).pointerType !== "touch") void canvas.requestPointerLock();
    });
    canvas.addEventListener("contextmenu", (e) => e.preventDefault());
    document.addEventListener("pointerlockchange", () => {
      this.locked = document.pointerLockElement === canvas;
      if (!this.locked) this.releaseAll();
      onLockChange(this.locked);
    });
    document.addEventListener("mousedown", (e) => this.mouseDown(e));
    document.addEventListener("mouseup", (e) => this.buttonFor(e)?.release());
    document.addEventListener("wheel", (e) => this.locked && hotbar.scroll(Math.sign(e.deltaY)));
    document.addEventListener("mousemove", (e) => {
      if (!this.locked) return;
      if (wheel.isOpen) wheel.move(e.movementX, e.movementY);
      else player.look.turn(e.movementX, e.movementY);
    });
    document.addEventListener("keydown", (e) => this.keyDown(e));
    document.addEventListener("keyup", (e) => this.keyUp(e));
    window.addEventListener("blur", () => this.releaseAll());
  }

  /** Movement and what to build now; held buttons repeat. */
  poll(nowMs: number): Intent {
    const picking = this.picking;
    this.picking = false;
    return {
      move: this.locked ? moveInputFromKeys(this.keys) : NO_INPUT,
      build: {
        breaking: this.breaking.due(nowMs),
        digging: this.breaking.held,
        placing: this.placing.due(nowMs),
        picking,
      },
    };
  }

  private mouseDown(e: MouseEvent): void {
    if (!this.locked) return;
    if (this.wheel.isOpen) return this.wheel.release(); // a click picks the emote, never a block
    if (e.button === 1) this.picking = true;
    this.buttonFor(e)?.press(e.timeStamp);
  }

  private buttonFor(e: MouseEvent): HoldRepeat | undefined {
    return e.button === 0 ? this.breaking : e.button === 2 ? this.placing : undefined;
  }

  private releaseAll(): void {
    this.keys.clear();
    this.breaking.release();
    this.placing.release();
  }

  private keyDown(e: KeyboardEvent): void {
    if (!this.locked) return;
    if (CAPTURED.has(e.code)) e.preventDefault();
    this.keys.add(e.code);
    if (e.repeat) return;
    if (/^Digit[1-9]$/.test(e.code)) this.hotbar.select(Number(e.code.slice(5)) - 1);
    const flyTap = e.code === "Space" && this.flyTap.press(e.timeStamp);
    if (flyTap || e.code === "KeyF") this.player.toggleFly();
    if (e.code === "KeyE") this.wheel.open();
  }

  private keyUp(e: KeyboardEvent): void {
    this.keys.delete(e.code);
    if (e.code === "KeyE" && this.locked && this.wheel.isOpen) this.wheel.release();
  }
}
