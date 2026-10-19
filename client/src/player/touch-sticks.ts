import { stickAxes } from "./input";

const DEAD_ZONE = 0.15;

interface Stick {
  id: number;
  x: number;
  y: number;
  dx: number;
  dy: number;
}

/** Turns screen touches into walking (a floating stick on the left half) and look drags (the rest). */
export class TouchSticks {
  private stick?: Stick;
  private readonly looks = new Map<number, { x: number; y: number }>();
  private look: [number, number] = [0, 0];

  constructor(
    public width: number,
    private readonly radius: number,
  ) {}

  down(id: number, x: number, y: number): void {
    if (!this.stick && x < this.width / 2) this.stick = { id, x, y, dx: 0, dy: 0 };
    else this.looks.set(id, { x, y });
  }

  move(id: number, x: number, y: number): void {
    const s = this.stick;
    if (s?.id === id) {
      const len = Math.hypot(x - s.x, y - s.y);
      const scale = len > this.radius ? this.radius / len : 1;
      [s.dx, s.dy] = [(x - s.x) * scale, (y - s.y) * scale];
      return;
    }
    const at = this.looks.get(id);
    if (!at) return;
    this.look = [this.look[0] + x - at.x, this.look[1] + y - at.y];
    [at.x, at.y] = [x, y];
  }

  up(id: number): void {
    if (this.stick?.id === id) this.stick = undefined;
    this.looks.delete(id);
  }

  walk(): { forward: number; right: number } {
    if (!this.stick) return { forward: 0, right: 0 };
    const [x, y] = stickAxes(this.stick.dx / this.radius, this.stick.dy / this.radius, DEAD_ZONE);
    return { forward: -y, right: x };
  }

  /** Where to draw the stick: its centre and the knob's offset. */
  joystick(): { x: number; y: number; dx: number; dy: number } | undefined {
    if (!this.stick) return undefined;
    const { x, y, dx, dy } = this.stick;
    return { x, y, dx, dy };
  }

  /** Pixels dragged since the last call. */
  takeLook(): [number, number] {
    const look = this.look;
    this.look = [0, 0];
    return look;
  }
}
