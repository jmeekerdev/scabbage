import { Scene } from "three";
import { describe, expect, it } from "vitest";
import { Ghost, ghostGeometry } from "./ghost";

const faces = (g: ReturnType<typeof ghostGeometry>) => (g.index?.count ?? 0) / 6;

describe("ghostGeometry", () => {
  it("draws only the outside faces of each set of cells", () => {
    expect(faces(ghostGeometry({ add: [{ x: 0, y: 0, z: 0 }], remove: [] }))).toBe(6);
    const pair = [
      { x: 0, y: 0, z: 0 },
      { x: 1, y: 0, z: 0 },
    ];
    expect(faces(ghostGeometry({ add: pair, remove: [] }))).toBe(10);
  });

  it("colors cells to build and cells to break apart, even side by side", () => {
    const g = ghostGeometry({ add: [{ x: 0, y: 0, z: 0 }], remove: [{ x: 1, y: 0, z: 0 }] });
    expect(faces(g)).toBe(12);
    const colors = g.getAttribute("color");
    expect(colors.getX(0)).not.toBe(colors.getX(colors.count - 1));
  });
});

describe("Ghost", () => {
  it("adds one mesh to the scene while shown, and removes it after", () => {
    const scene = new Scene();
    const ghost = new Ghost(scene);
    ghost.show({ add: [{ x: 0, y: 0, z: 0 }], remove: [] });
    expect(scene.children).toHaveLength(1);
    ghost.show({ add: [{ x: 5, y: 0, z: 0 }], remove: [] });
    expect(scene.children).toHaveLength(1);
    ghost.show(undefined);
    expect(scene.children).toHaveLength(0);
  });
});
