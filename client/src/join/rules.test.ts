import { describe, expect, it } from "vitest";
import { AVATAR_COLORS, cleanName, colorCss } from "./rules";

describe("display names", () => {
  it("accepts friendly names and trims them", () => {
    expect(cleanName("  Ann ")).toBe("Ann");
    expect(cleanName("Zoë_the-2nd")).toBe("Zoë_the-2nd");
  });

  it("rejects empty, long or odd names like the server does", () => {
    for (const bad of ["", "   ", "this name is far too long", "<script>", "a\u0000b"]) {
      expect(cleanName(bad)).toBeUndefined();
    }
  });
});

describe("avatar colors", () => {
  it("are distinct 24-bit colors", () => {
    expect(new Set(AVATAR_COLORS).size).toBe(AVATAR_COLORS.length);
    expect(AVATAR_COLORS.every((c) => c >= 0 && c <= 0xffffff)).toBe(true);
  });

  it("convert to CSS hex", () => {
    expect(colorCss(0x00ff08)).toBe("#00ff08");
  });
});
