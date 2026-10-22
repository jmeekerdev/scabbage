import { Scene, type LineBasicMaterial, type LineSegments, type Mesh } from "three";
import { describe, expect, it } from "vitest";
import { lookById } from "../looks/registry";
import { Selection } from "./selection";

function parts(scene: Scene) {
  const box = scene.getObjectByName("selection") as LineSegments;
  return { box, shade: box.children[0] as Mesh };
}

/** Every vertex of a geometry, as [x, y, z] in the selection's own space. */
function points(o: LineSegments | Mesh): number[][] {
  const a = o.geometry.attributes.position;
  return Array.from({ length: a.count }, (_, i) => [a.getX(i), a.getY(i), a.getZ(i)]);
}

const near = (p: number[], q: number[]) => p.every((v, i) => Math.abs(v - q[i]) < 0.01);

describe("Selection", () => {
  it("draws a plain cube when no corner moves", () => {
    const scene = new Scene();
    const s = new Selection(scene, () => [0, 0, 0]);
    s.show({ x: 2, y: 3, z: 4 });
    const { box } = parts(scene);
    expect(box.visible).toBe(true);
    expect(box.position.toArray()).toEqual([2.5, 3.5, 4.5]);
    expect(points(box).some((p) => near(p, [0.5, 0.5, 0.5]))).toBe(true);
  });

  it("bends its corners to where soft corners moved the block's", () => {
    const scene = new Scene();
    const bend = (x: number, y: number, z: number) =>
      x === 3 && y === 4 && z === 5 ? [-0.25, -0.25, -0.25] : [0, 0, 0];
    new Selection(scene, bend).show({ x: 2, y: 3, z: 4 }, 0.5);
    const { box, shade } = parts(scene);
    for (const o of [box, shade]) {
      expect(points(o).some((p) => near(p, [0.25, 0.25, 0.25]))).toBe(true);
      expect(points(o).some((p) => near(p, [0.5, 0.5, 0.5]))).toBe(false);
    }
  });

  it("scales the bend with the brush size", () => {
    const scene = new Scene();
    const bend = (x: number, y: number, z: number) => (x + y + z === 6 ? [-0.5, 0, 0] : [0, 0, 0]);
    new Selection(scene, bend).show({ x: 0, y: 0, z: 0, size: 2 });
    expect(points(parts(scene).box).some((p) => near(p, [0.25, 0.5, 0.5]))).toBe(true);
  });

  it("draws in the look's outline color, or black without one", () => {
    const scene = new Scene();
    const s = new Selection(scene);
    const line = () => (parts(scene).box.material as LineBasicMaterial).color;
    s.setLook(lookById("ink-and-cel"));
    expect(line().getHex()).not.toBe(0);
    s.setLook(lookById("today"));
    expect(line().getHex()).toBe(0);
  });
});
