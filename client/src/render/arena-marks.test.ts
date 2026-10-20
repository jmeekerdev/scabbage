import { Color, Mesh, MeshBasicMaterial, Scene } from "three";
import { describe, expect, it } from "vitest";
import { ArenaMarks, TEAM_COLORS } from "./arena-marks";

const color = (m: Mesh) => (m.material as MeshBasicMaterial).color;

describe("arena marks", () => {
  it("puts a team-colored pad at each spawn and a flag at each goal", () => {
    const scene = new Scene();
    const marks = new ArenaMarks(scene);
    marks.set([
      { marks: [{ x: 2, y: 5, z: -3, team: "red", goal: false }] },
      { marks: [{ x: 0, y: 1, z: 0, team: "blue", goal: true }] },
    ]);
    const [pad, flag] = marks.group.children;
    expect(pad.position.toArray()).toEqual([2.5, expect.closeTo(5, 1), -2.5]);
    expect(color(pad.children[0] as Mesh)).toEqual(new Color(TEAM_COLORS.red));
    expect(flag.position.toArray()).toEqual([0.5, 1, 0.5]);
    expect(flag.children.some((c) => color(c as Mesh).equals(new Color(TEAM_COLORS.blue)))).toBe(
      true,
    );
  });

  it("shows marks for everyone in white and replaces old marks", () => {
    const marks = new ArenaMarks(new Scene());
    marks.set([{ marks: [{ x: 0, y: 0, z: 0, team: "", goal: false }] }]);
    expect(color(marks.group.children[0].children[0] as Mesh)).toEqual(new Color(TEAM_COLORS[""]));
    marks.set([]);
    expect(marks.group.children).toHaveLength(0);
  });
});
