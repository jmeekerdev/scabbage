import { describe, expect, it } from "vitest";
import { buildWorldFor } from "../player/building";
import { EYE_HEIGHT, newBody, stepBody } from "../player/physics";
import { FULL_BLOCK } from "../world/shapes";
import { Autopilot, type Plan } from "./autopilot";
import { tile } from "./job";

const [AIR, STONE, PLANKS] = [0, 1, 5];

/** Stone below y=10 and whatever was set; air elsewhere. */
function world() {
  const set = new Map<string, number>();
  const blockAt = (x: number, y: number, z: number) =>
    set.get(`${x},${y},${z}`) ?? (y < 10 ? STONE : AIR);
  const blocks = [
    { id: AIR, name: "air", solid: false, shape: 0, mineMs: 0 },
    { id: STONE, name: "stone", solid: true, shape: 0, mineMs: 0 },
    { id: PLANKS, name: "planks", solid: true, shape: 0, mineMs: 0 },
  ];
  const w = buildWorldFor({ radius: -1, height: 64, blocks }, blockAt);
  return { w, put: (x: number, y: number, z: number, b: number) => set.set(`${x},${y},${z}`, b) };
}

function setup() {
  const events: string[] = [];
  const pilot = new Autopilot({
    progress: (p) => events.push(`${p.kind} ${Math.round(p.fraction * 100)}%`),
    done: (kind, r) =>
      events.push(`${kind} done ${r.blocks}${r.missed ? ` missed ${r.missed}` : ""}`),
  });
  return { pilot, events, ...world() };
}

const eyeOf = (b: { x: number; y: number; z: number }) => ({ ...b, y: b.y + EYE_HEIGHT });
const fill = (max: number, block = PLANKS): Plan => ({
  kind: "fill",
  steps: tile({ min: { x: 0, y: 10, z: 0 }, max: { x: max, y: 10, z: 0 } }, false).map((cube) => ({
    cube,
    block,
  })),
});

describe("Autopilot", () => {
  it("does nothing until given a job", () => {
    const { pilot, w } = setup();
    const body = newBody(0, 10, 0);
    expect(pilot.busy).toBe(false);
    expect(pilot.tick(0.1, body, eyeOf(body), w)).toBeUndefined();
  });

  it("flies to a goto target and stops there", () => {
    const { pilot, events, w } = setup();
    const body = newBody(0, 20, 0);
    pilot.start({ kind: "goto", steps: [], target: { x: 30, y: 20, z: 0 } });
    const t = pilot.tick(0.1, body, eyeOf(body), w)!;
    expect(t.move.forward).toBe(1);
    expect(t.edits).toEqual([]);
    Object.assign(body, { x: 30 });
    pilot.tick(0.1, body, eyeOf(body), w);
    expect(pilot.busy).toBe(false);
    expect(events).toEqual(["goto done 0"]);
  });

  it("builds only what is in reach, at a watchable pace", () => {
    const { pilot, w } = setup();
    const body = newBody(1, 12, 0);
    pilot.start(fill(2)); // three 1-block cubes
    const edits = [0.05, 0.05, 0.1, 0.125].flatMap(
      (dt) => pilot.tick(dt, body, eyeOf(body), w)!.edits,
    );
    expect(edits.map((e) => e.x)).toEqual([0, 1, 2]);
    expect(edits[0]).toMatchObject({ y: 10, block: PLANKS, cells: [{ x: 0, y: 10, z: 0 }] });
  });

  it("flies closer when the next cube is out of reach", () => {
    const { pilot, w } = setup();
    const body = newBody(-40, 12, 0);
    pilot.start(fill(2));
    const t = pilot.tick(1, body, eyeOf(body), w)!;
    expect(t.edits).toEqual([]);
    expect(t.move.forward).toBeGreaterThan(0);
    expect(t.fly).toBe(true);
  });

  it("skips cubes with nothing to change and reports progress and the total", () => {
    const { pilot, events, w, put } = setup();
    put(0, 10, 0, STONE);
    const body = newBody(1, 12, 0);
    pilot.start(fill(2));
    for (let i = 0; i < 4; i++) pilot.tick(0.125, body, eyeOf(body), w);
    expect(pilot.busy).toBe(false);
    expect(events).toEqual(["fill 50%", "fill done 2"]);
  });

  it("stops on cancel", () => {
    const { pilot, w } = setup();
    const body = newBody(1, 12, 0);
    pilot.start(fill(2));
    expect(pilot.cancel()).toBe(true);
    expect(pilot.tick(1, body, eyeOf(body), w)).toBeUndefined();
    expect(pilot.cancel()).toBe(false);
  });

  it("builds from where it got stuck if the cube is in reach", () => {
    const { pilot, w } = setup();
    const body = newBody(1, 18, 0); // above the stand point: too far to start, close enough to reach
    pilot.start(fill(0));
    let edits = 0;
    for (let i = 0; i < 8; i++) edits += pilot.tick(0.125, body, eyeOf(body), w)?.edits.length ?? 0;
    expect(edits).toBe(1);
  });

  it("gives up on cubes and places it can't get near, and says so", () => {
    const { pilot, events, w } = setup();
    const body = newBody(-40, 12, 0); // never moves, as if walled in
    pilot.start(fill(0));
    for (let i = 0; i < 200 && pilot.busy; i++) pilot.tick(0.125, body, eyeOf(body), w);
    pilot.start({ kind: "goto", steps: [], target: { x: 30, y: 20, z: 0 } });
    for (let i = 0; i < 200 && pilot.busy; i++) pilot.tick(0.125, body, eyeOf(body), w);
    expect(events).toEqual(["fill done 0 missed 1", "goto done 0 missed 1"]);
  });

  it("flies around a wall, with real physics, to fill a cube behind it", () => {
    const { pilot, events, w, put } = setup();
    for (let y = 10; y < 64; y++) for (let z = -12; z <= 12; z++) put(20, y, z, STONE);
    const body = { ...newBody(0.5, 12, 0.5), flying: true };
    const boxesAt = (x: number, y: number, z: number) =>
      w.solid[w.blockAt(x, y, z) ?? STONE] ? FULL_BLOCK : [];
    pilot.start({
      kind: "fill",
      steps: [{ cube: { x: 40, y: 10, z: 0, size: 1 }, block: PLANKS }],
    });
    for (let i = 0; i < 60 * 20 && pilot.busy; i++) {
      const t = pilot.tick(1 / 60, body, eyeOf(body), w)!;
      for (const e of t.edits) {
        expect(Math.hypot(body.x - 40.5, body.y + EYE_HEIGHT - 10.5, body.z - 0.5)).toBeLessThan(
          15,
        );
        for (const c of e.cells) put(c.x, c.y, c.z, e.block);
      }
      body.flying = true;
      stepBody(body, t.move, t.yaw ?? 0, 1 / 60, boxesAt);
    }
    expect(events).toEqual(["fill done 1"]);
    expect(w.blockAt(40, 10, 0)).toBe(PLANKS);
  });

  it("moves out of the way of a cell its own body is in, instead of skipping the cell", () => {
    const { pilot, events, w, put } = setup();
    const body = { ...newBody(0.5, 10, 0.5), flying: true }; // standing in the cell to fill
    const boxesAt = (x: number, y: number, z: number) =>
      w.solid[w.blockAt(x, y, z) ?? STONE] ? FULL_BLOCK : [];
    pilot.start({ kind: "fill", steps: [{ cube: { x: 0, y: 10, z: 0, size: 1 }, block: PLANKS }] });
    for (let i = 0; i < 60 * 10 && pilot.busy; i++) {
      const t = pilot.tick(1 / 60, body, eyeOf(body), w)!;
      for (const e of t.edits) for (const c of e.cells) put(c.x, c.y, c.z, e.block);
      body.flying = true;
      stepBody(body, t.move, t.yaw ?? 0, 1 / 60, boxesAt);
    }
    expect(events).toEqual(["fill done 1"]);
    expect(w.blockAt(0, 10, 0)).toBe(PLANKS);
  });
});
