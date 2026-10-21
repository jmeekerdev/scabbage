import { describe, expect, it } from "vitest";
import type { Look } from "../looks/look";
import { lookById } from "../looks/registry";
import { afterFeatures } from "./after-pass";

const today = lookById("today");
const after = (a: Look["after"]): Look => ({ ...today, after: a });

describe("afterFeatures", () => {
  it("runs no pass for a look without one", () => {
    expect(afterFeatures(today, false)).toBeUndefined();
    expect(afterFeatures(after({}), false)).toBeUndefined();
  });

  it("switches on only what the look uses", () => {
    expect(afterFeatures(after({ vignette: 0.2, saturation: 1 }), false)).toEqual({
      VIGNETTE: "",
    });
    expect(
      afterFeatures(after({ outline: { width: 1, strength: 0.5, creases: true } }), false),
    ).toEqual({ OUTLINE: "", CREASES: "", DARKEN: "" });
    expect(
      afterFeatures(
        after({ outline: { width: 1, strength: 1, color: { r: 0, g: 0, b: 0 } } }),
        false,
      ),
    ).toEqual({ OUTLINE: "" });
    expect(afterFeatures(after({ paper: 1, saturation: 1.1 }), false)).toEqual({
      PAPER: "",
      SATURATE: "",
    });
  });

  it("tilt-shifts only the overhead map view", () => {
    expect(afterFeatures(after({ tiltShift: 1 }), false)).toBeUndefined();
    expect(afterFeatures(after({ tiltShift: 1 }), true)).toEqual({ TILT: "" });
  });
});
