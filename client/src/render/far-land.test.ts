import { lookById } from "../looks/registry";
import { describe, expect, it } from "vitest";
import { FAR_CELLS } from "../world/far-mesh";
import { FarLand, farDistance } from "./far-land";

const bounds = { radius: -1, height: 384 };
const SAMPLES = FAR_CELLS * FAR_CELLS;

function setup() {
  const sent: { x: number; z: number }[][] = [];
  const far = new FarLand((columns) => sent.push(columns), lookById("today"));
  far.start(bounds);
  far.setColors(Uint8Array.of(0, 0, 0, 90, 160, 60));
  return { sent, far };
}

/** Answers every request with flat grass, then lets the meshes catch up. */
function answerAll(far: FarLand, sent: { x: number; z: number }[][], x = 0) {
  for (let i = 0; i < 100 && sent.length; i++) {
    for (const batch of sent.splice(0)) {
      far.receive(
        batch.map((pos) => ({
          pos,
          tops: Array(SAMPLES).fill(100),
          blocks: Array(SAMPLES).fill(1),
        })),
      );
    }
    far.tick(x, 0, 2);
  }
  for (let i = 0; i < 100; i++) far.tick(x, 0, 2);
}

describe("FarLand", () => {
  it("asks for columns out to three times the view distance, nearest first, 64 at a time", () => {
    const { sent, far } = setup();
    far.tick(0, 0, 2);
    expect(farDistance(2)).toBe(6);
    expect(sent[0][0]).toEqual({ x: 0, z: 0 });
    expect(sent.every((b) => b.length <= 64)).toBe(true);
    answerAll(far, sent);
    const asked = (far as unknown as { tiles: Map<string, unknown> }).tiles;
    expect(asked.has("6,0")).toBe(true);
    expect(asked.has("7,0")).toBe(false);
  });

  it("draws arrived columns as region meshes", () => {
    const { sent, far } = setup();
    far.tick(0, 0, 2);
    answerAll(far, sent);
    expect(far.group.children.length).toBeGreaterThan(0);
  });

  it("hides the columns the chunks draw, wrapping negative ones into its mask", () => {
    const { far } = setup();
    far.tick(0, 0, 2);
    far.cover(-1, 2, true);
    expect(far.tick(0, 0, 2)).toBe(true);
    expect(far.tick(0, 0, 2)).toBe(false);
    const mask = (far as unknown as { mask: { image: { data: Uint8Array } } }).mask.image.data;
    expect(mask[2 * 64 + 63]).toBe(255);
    far.cover(-1, 2, false);
    expect(mask[2 * 64 + 63]).toBe(0);
  });

  it("forgets columns and regions left far behind", () => {
    const { sent, far } = setup();
    far.tick(0, 0, 2);
    answerAll(far, sent);
    far.tick(32 * 100, 0, 2);
    for (let i = 0; i < 100; i++) far.tick(32 * 100, 0, 2);
    expect(far.group.children.length).toBe(0);
  });
});
