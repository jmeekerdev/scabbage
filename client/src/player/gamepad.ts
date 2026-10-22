import type { Emote } from "../pb/scabbage/v1/net_pb";
import { EMOTES } from "../social/emotes";
import type { Hotbar } from "./hotbar";
import { DoubleTap, HoldRepeat, NO_BUILD, NO_INPUT, stickAxes, type Intent } from "./input";
import type { Player } from "./player";

/** The parts of a browser Gamepad this reads. */
export interface PadState {
  readonly axes: readonly number[];
  readonly buttons: readonly { readonly pressed: boolean }[];
}

// Standard-mapping button indices.
const BTN = { a: 0, b: 1, x: 2, y: 3, lb: 4, rb: 5, lt: 6, rt: 7, back: 8, rs: 11 } as const;
// D-pad up, right, down, left send the emotes in those directions on the wheel.
const DPAD_EMOTES: [number, Emote][] = [12, 15, 13, 14].map((b, i) => [b, EMOTES[i * 2].emote]);
const DEAD_ZONE = 0.2;
const LOOK_RADIANS_PER_SECOND = 3;
const FLY_DOUBLE_TAP_MS = 300;
const BUILD_REPEAT_MS = 160;

/** A standard gamepad: left stick walks, right stick looks, triggers build, bumpers pick a slot, D-pad emotes, clicking the right stick resizes the brush. */
export class GamepadControls {
  /** True once the pad has been used, so phantom pads don't change the HUD. */
  active = false;
  private held: readonly boolean[] = [];
  private lastMs?: number;
  private readonly flyTap = new DoubleTap(FLY_DOUBLE_TAP_MS);
  private readonly breaking = new HoldRepeat(BUILD_REPEAT_MS);
  private readonly placing = new HoldRepeat(BUILD_REPEAT_MS);

  constructor(
    private readonly player: Player,
    private readonly hotbar: Hotbar,
    private readonly pads: () => readonly (PadState | null)[],
    private readonly onEmote: (e: Emote) => void,
  ) {}

  poll(nowMs: number): Intent {
    const dt = this.lastMs === undefined ? 0 : (nowMs - this.lastMs) / 1000;
    this.lastMs = nowMs;
    const pad = this.pads().find((p) => p !== null);
    if (!pad) return this.lost();
    const [prev, held] = [this.held, pad.buttons.map((b) => b.pressed)];
    const pressed = (i: number) => (held[i] ?? false) && !(prev[i] ?? false);
    this.held = held;
    const [mx, my] = stickAxes(pad.axes[0] ?? 0, pad.axes[1] ?? 0, DEAD_ZONE);
    this.look(pad.axes[2] ?? 0, pad.axes[3] ?? 0, dt);
    this.buttons(pressed, nowMs);
    this.active ||= mx !== 0 || my !== 0 || held.some(Boolean);
    return {
      move: { forward: -my, right: mx, jump: held[BTN.a] ?? false, descend: held[BTN.b] ?? false },
      build: {
        breaking: this.breaking.due(nowMs),
        digging: this.breaking.held,
        placing: this.placing.due(nowMs),
        picking: pressed(BTN.x),
        scripting: pressed(BTN.back),
        resizing: pressed(BTN.rs),
      },
    };
  }

  /** Turns at a rate; squaring the tilt gives fine aim near the center. */
  private look(x: number, y: number, dt: number): void {
    const [lx, ly] = stickAxes(x, y, DEAD_ZONE);
    const rate = Math.hypot(lx, ly) * LOOK_RADIANS_PER_SECOND * dt;
    this.player.look.turnBy(lx * rate, ly * rate);
    this.active ||= lx !== 0 || ly !== 0;
  }

  private buttons(pressed: (i: number) => boolean, nowMs: number): void {
    if ((pressed(BTN.a) && this.flyTap.press(nowMs)) || pressed(BTN.y)) this.player.toggleFly();
    if (pressed(BTN.lb)) this.hotbar.scroll(-1);
    if (pressed(BTN.rb)) this.hotbar.scroll(1);
    for (const [b, e] of DPAD_EMOTES) if (pressed(b)) this.onEmote(e);
    this.hold(this.breaking, BTN.rt, pressed, nowMs);
    this.hold(this.placing, BTN.lt, pressed, nowMs);
  }

  private hold(r: HoldRepeat, button: number, pressed: (i: number) => boolean, nowMs: number) {
    if (pressed(button)) r.press(nowMs);
    else if (!(this.held[button] ?? false)) r.release();
  }

  private lost(): Intent {
    this.active = false;
    this.held = [];
    this.breaking.release();
    this.placing.release();
    return { move: NO_INPUT, build: NO_BUILD };
  }
}
