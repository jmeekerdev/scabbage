import { DoubleTap, moveInputFromKeys } from "./input";
import type { MoveInput } from "./physics";
import type { Player } from "./player";

const FLY_DOUBLE_TAP_MS = 300;
const NO_INPUT: MoveInput = { forward: 0, right: 0, jump: false, descend: false };
// Keys the page would otherwise use to scroll.
const CAPTURED = new Set(["Space", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"]);

/** Pointer-lock mouse look and keyboard movement; inactive until the canvas is clicked. */
export class KeyboardMouse {
  private readonly keys = new Set<string>();
  private readonly flyTap = new DoubleTap(FLY_DOUBLE_TAP_MS);
  private locked = false;

  constructor(
    canvas: HTMLCanvasElement,
    private readonly player: Player,
    onLockChange: (locked: boolean) => void,
  ) {
    canvas.addEventListener("click", () => void canvas.requestPointerLock());
    document.addEventListener("pointerlockchange", () => {
      this.locked = document.pointerLockElement === canvas;
      if (!this.locked) this.keys.clear();
      onLockChange(this.locked);
    });
    document.addEventListener("mousemove", (e) => {
      if (this.locked) player.look.turn(e.movementX, e.movementY);
    });
    document.addEventListener("keydown", (e) => this.keyDown(e));
    document.addEventListener("keyup", (e) => this.keys.delete(e.code));
    window.addEventListener("blur", () => this.keys.clear());
  }

  input(): MoveInput {
    return this.locked ? moveInputFromKeys(this.keys) : NO_INPUT;
  }

  private keyDown(e: KeyboardEvent): void {
    if (!this.locked) return;
    if (CAPTURED.has(e.code)) e.preventDefault();
    this.keys.add(e.code);
    if (e.repeat) return;
    const flyTap = e.code === "Space" && this.flyTap.press(e.timeStamp);
    if (flyTap || e.code === "KeyF") this.player.toggleFly();
  }
}
