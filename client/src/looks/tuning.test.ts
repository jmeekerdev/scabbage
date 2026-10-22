import { describe, expect, it } from "vitest";
import { checkLook } from "./check";
import { lookById } from "./registry";
import { colourControls, lookFile, numberControls, setAt, toHex, fromHex } from "./tuning";

const today = lookById("today");
const ink = lookById("ink-and-cel");

describe("numberControls", () => {
  it("offers only numbers whose part of the look exists", () => {
    const paths = numberControls(today).map((c) => c.path);
    expect(paths).toContain("light.lampFade");
    expect(paths).toContain("surface.wash"); // optional, but its part is there
    expect(paths).not.toContain("light.sun.wrap");
    expect(paths).not.toContain("after.outline.width");
  });

  it("includes the sun and outline when the look has them", () => {
    const paths = numberControls(ink).map((c) => c.path);
    expect(paths).toEqual(expect.arrayContaining(["light.sun.bands", "light.sun.dir.y"]));
    expect(paths).toContain("after.outline.width");
  });

  it("leaves out shape numbers that would rebuild every chunk", () => {
    expect(numberControls(ink).map((c) => c.path)).not.toContain("shape.softCorners");
  });

  it("reads each number's current value, or its default", () => {
    const at = (path: string) => numberControls(today).find((c) => c.path === path)?.value;
    expect(at("fog.start")).toBe(today.fog.start);
    expect(at("surface.wash")).toBe(0);
    expect(at("after.saturation")).toBe(1);
  });
});

describe("colourControls", () => {
  it("offers the sky and present outline colours as sRGB", () => {
    const paths = colourControls(ink).map((c) => c.path);
    expect(paths).toEqual(expect.arrayContaining(["sky.day", "after.outline.color"]));
    expect(colourControls(today).map((c) => c.path)).not.toContain("after.outline.color");
  });
});

describe("setAt", () => {
  it("changes one number without touching the original", () => {
    const next = setAt(today, "light.faces.top", 0.5);
    expect(next.light.faces.top).toBe(0.5);
    expect(today.light.faces.top).not.toBe(0.5);
    expect(next.light.lamp).toBe(today.light.lamp);
  });

  it("makes the optional parts it needs", () => {
    expect(setAt(today, "after.vignette", 0.3).after?.vignette).toBe(0.3);
  });
});

describe("hex colours", () => {
  it("round-trips an sRGB colour through a colour input's value", () => {
    expect(toHex({ r: 1, g: 0.5, b: 0 })).toBe("#ff8000");
    expect(fromHex("#ff8000")).toEqual({ r: 1, g: 128 / 255, b: 0 });
  });
});

describe("lookFile", () => {
  it("writes a look file named for the look, that reads back as the same look", () => {
    const tuned = setAt(ink, "after.vignette", 0.123456);
    const { name, text } = lookFile({ ...tuned, name: "My Ink Look" });
    expect(name).toBe("my-ink-look.look.ts");
    expect(text).toMatch(/^import type \{ Look \} from "\.\/look";/);
    expect(text).toMatch(/satisfies Look;\n$/);
    const body = text.slice(text.indexOf("{", text.indexOf("export")), text.lastIndexOf("}") + 1);
    const back = JSON.parse(body);
    expect(back.after.vignette).toBe(0.1235);
    expect(checkLook(back)).toEqual([]);
  });
});
