import { describe, expect, it } from "vitest";
import { lookCost, type Look } from "./look";
import { DEFAULT_LOOK, LOOKS, lookById } from "./registry";

describe("look registry", () => {
  it("finds every look file, named by its file", () => {
    expect(LOOKS.has("today")).toBe(true);
    expect(LOOKS.get("today")?.name).toBe("Today");
    expect(DEFAULT_LOOK).toBe("today");
  });

  it("falls back to the default look for unknown names", () => {
    expect(lookById("no-such-look")).toBe(LOOKS.get(DEFAULT_LOOK));
  });

  it("gives every look a name and a description", () => {
    for (const look of LOOKS.values()) {
      expect(look.name.length).toBeGreaterThan(0);
      expect(look.about.length).toBeGreaterThan(0);
    }
  });
});

describe("lookCost", () => {
  const today = lookById("today");
  const shaped = (shape: Partial<Look["shape"]>): Look => ({
    ...today,
    shape: { ...today.shape, ...shape },
  });

  it("calls a plain look the fastest", () => {
    expect(lookCost(today)).toBe("fastest");
  });

  it("charges for mesher work", () => {
    expect(lookCost(shaped({ edges: true }))).toBe("light");
    expect(lookCost(shaped({ edges: true, softCorners: 0.5 }))).toBe("heavier");
  });
});
