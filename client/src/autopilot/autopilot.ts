import { cubeEdit, REACH, type BuildWorld, type Edit } from "../player/building";
import type { Cube } from "../player/brush";
import { NO_INPUT } from "../player/input";
import { EYE_HEIGHT, HALF_WIDTH, HEIGHT, type Body, type MoveInput } from "../player/physics";
import { flyPath, type Passable } from "./path";
import { Steering, type SteerStep, type Vec } from "./route";

const EDIT_S = 1 / 8; // a watchable pace, well under the server's edit limit
const NEAR = REACH - 4; // flying stops this close, so a little drift keeps the next cubes in reach
const SKIPS_PER_TICK = 256; // empty cubes passed over in one frame
const MAX_PITCH = Math.PI / 2 - 0.01;
const MAX_STUCK = 8; // times stuck on one cube or goto before giving up on it
const WAYPOINT_PASSED = 1.5; // blocks from a waypoint that count as through it
const GOTO_NEAR = 3; // blocks from a goto target the route search aims for, before the last stretch

/** One brush to send: AIR clears the cube, any other block fills its empty spots. */
export interface Step {
  cube: Cube;
  block: number;
}

/** A job worked out into steps, and for goto a point to end at. */
export interface Plan {
  kind: string;
  steps: Step[];
  target?: Vec;
}

/** What the autopilot presses this frame, where it looks, and the edits to send. */
export interface Tick {
  move: MoveInput;
  yaw?: number;
  pitch?: number;
  fly: boolean;
  edits: Edit[];
}

/** How a plan went: blocks changed, and cubes or a goto target it couldn't get near. */
export interface Outcome {
  blocks: number;
  missed: number;
}

export interface AutopilotEvents {
  progress(p: { kind: string; fraction: number }): void;
  done(kind: string, outcome: Outcome): void;
}

/** Flies the player's own avatar through a plan, editing every cube once it is in reach. */
export class Autopilot {
  private plan?: Plan;
  private next = 0;
  private quarter = 0;
  private blocks = 0;
  private missed = 0;
  private stuckTimes = 0;
  private sinceEditS = 0;
  private editing = false;
  private route?: Vec[]; // waypoints to the current cube or target, worked out when needed
  private routed = false; // whether the route was searched out, so is clear to fly straight along
  private readonly steering = new Steering();

  constructor(private readonly events: AutopilotEvents) {}

  get busy(): boolean {
    return this.plan !== undefined;
  }

  start(plan: Plan): void {
    Object.assign(this, { plan, next: 0, quarter: 0, blocks: 0, missed: 0, stuckTimes: 0 });
    this.sinceEditS = EDIT_S;
    this.editing = false;
    this.route = undefined;
    this.steering.reset();
  }

  /** Drops the job, saying whether there was one. */
  cancel(): boolean {
    const was = this.busy;
    this.plan = undefined;
    return was;
  }

  tick(dt: number, body: Body, eye: Vec, w: BuildWorld): Tick | undefined {
    const plan = this.plan;
    if (!plan) return undefined;
    this.sinceEditS += dt;
    this.skipEmpty(plan, body, w);
    const step = plan.steps[this.next];
    if (step) {
      const tick = this.work(step, dt, body, eye, w);
      this.report(plan);
      return tick;
    }
    const target = plan.target;
    if (target && !this.steering.arrived(body, target)) {
      const near = (feet: Vec) => distance(feet, target) <= GOTO_NEAR;
      const steer = this.fly(dt, body, target, near, w, target);
      if (!(steer.stuck && this.givesUp())) return { ...steer, fly: true, edits: [] };
    }
    this.finish(plan);
    return { move: NO_INPUT, fly: true, edits: [] };
  }

  /** Edits the step once it is in reach and the pace allows, or flies closer. */
  private work(step: Step, dt: number, body: Body, eye: Vec, w: BuildWorld): Tick {
    const center = cubeCenter(step.cube);
    const far = distance(eye, center);
    this.editing = far <= (this.editing ? REACH : NEAR);
    if (!this.editing) {
      const near = (feet: Vec) => canWorkFrom(feet, step.cube);
      const steer = this.fly(dt, body, standPoint(step.cube), near, w);
      // Blocked short of the stand point, e.g. by trees: work from here if the cube is in reach.
      this.editing = steer.stuck && far <= REACH;
      if (!this.editing) {
        if (steer.stuck && this.givesUp()) this.advance();
        return { ...steer, fly: true, edits: [] };
      }
    }
    this.steering.reset();
    const edits: Edit[] = [];
    if (this.sinceEditS >= EDIT_S) {
      this.sinceEditS = 0;
      const edit = cubeEdit(step.cube, step.block, w, body);
      if (edit) edits.push(edit);
      this.blocks += edit?.cells.length ?? 0;
      this.advance();
    }
    return { move: NO_INPUT, ...facing(eye, center), fly: true, edits };
  }

  /** Steers along a route found through open air, or straight at `goal` if there's none. */
  private fly(
    dt: number,
    body: Body,
    goal: Vec,
    near: (feet: Vec) => boolean,
    w: BuildWorld,
    last?: Vec,
  ): SteerStep {
    if (!this.route) {
      const found = flyPath(body, goal, near, passable(w));
      this.routed = found !== undefined;
      this.route = [...(found ?? (last ? [] : [goal])), ...(last ? [last] : [])];
    }
    const route = this.route;
    while (route.length > 1 && distance(body, route[0]!) <= WAYPOINT_PASSED) route.shift();
    const steer = this.steering.steer(dt, body, route[0]!, !this.routed);
    if (steer.stuck) this.route = undefined; // look for another way from here
    return steer;
  }

  /** Passes over loaded cubes that are already as the plan wants them. */
  private skipEmpty(plan: Plan, body: Body, w: BuildWorld): void {
    for (let n = 0; n < SKIPS_PER_TICK; n++) {
      const step = plan.steps[this.next];
      if (!step || !loaded(step.cube, w) || cubeEdit(step.cube, step.block, w, body)) return;
      this.advance();
    }
  }

  private advance(): void {
    this.next++;
    this.stuckTimes = 0;
    this.route = undefined;
  }

  /** Counts a time stuck, saying whether that's too many and the cube or target is missed. */
  private givesUp(): boolean {
    if (++this.stuckTimes < MAX_STUCK) return false;
    this.missed++;
    return true;
  }

  /** Says each time another quarter of the plan is done, short of all of it. */
  private report(plan: Plan): void {
    const quarter = Math.floor((this.next / plan.steps.length) * 4);
    if (quarter <= this.quarter || quarter >= 4) return;
    this.quarter = quarter;
    this.events.progress({ kind: plan.kind, fraction: quarter / 4 });
  }

  private finish(plan: Plan): void {
    this.plan = undefined;
    this.events.done(plan.kind, { blocks: this.blocks, missed: this.missed });
  }
}

const cubeCenter = (c: Cube): Vec => ({
  x: c.x + c.size / 2,
  y: c.y + c.size / 2,
  z: c.z + c.size / 2,
});

/** Where the feet go to work a cube: over its middle, a block above its top. */
const standPoint = (c: Cube): Vec => ({
  x: c.x + c.size / 2,
  y: c.y + c.size + 1,
  z: c.z + c.size / 2,
});

/** Whether feet here put the eye close enough to work a cube, with the body clear of it. */
function canWorkFrom(feet: Vec, c: Cube): boolean {
  const eye = { ...feet, y: feet.y + EYE_HEIGHT };
  const clear =
    feet.x + HALF_WIDTH <= c.x ||
    feet.x - HALF_WIDTH >= c.x + c.size ||
    feet.z + HALF_WIDTH <= c.z ||
    feet.z - HALF_WIDTH >= c.z + c.size ||
    feet.y >= c.y + c.size ||
    feet.y + HEIGHT <= c.y;
  return clear && distance(eye, cubeCenter(c)) <= NEAR - 1;
}

const passable =
  (w: BuildWorld): Passable =>
  (x, y, z) => {
    const b = w.blockAt(x, y, z);
    return b !== undefined && !w.solid[b];
  };

const distance = (a: Vec, b: Vec) => Math.hypot(a.x - b.x, a.y - b.y, a.z - b.z);

/** Turns the view toward a point, as a player watching their own hands would. */
function facing(eye: Vec, p: Vec): { yaw: number; pitch: number } {
  const [dx, dy, dz] = [p.x - eye.x, p.y - eye.y, p.z - eye.z];
  const pitch = Math.atan2(dy, Math.hypot(dx, dz));
  return { yaw: Math.atan2(-dx, -dz), pitch: Math.max(-MAX_PITCH, Math.min(MAX_PITCH, pitch)) };
}

function loaded(c: Cube, w: BuildWorld): boolean {
  const corners = [c.x, c.x + c.size - 1].flatMap((x) =>
    [c.z, c.z + c.size - 1].map((z) => w.blockAt(x, c.y, z)),
  );
  return corners.every((b) => b !== undefined);
}
