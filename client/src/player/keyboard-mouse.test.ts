import { describe, expect, it } from "vitest";
import { relocksOn } from "./keyboard-mouse";

describe("relocksOn", () => {
  it("takes the mouse back on a game key", () => {
    expect(relocksOn({ key: "w", repeat: false }, false)).toBe(true);
  });

  it("leaves it free on Esc, held keys and while typing", () => {
    expect(relocksOn({ key: "Escape", repeat: false }, false)).toBe(false);
    expect(relocksOn({ key: "w", repeat: true }, false)).toBe(false);
    expect(relocksOn({ key: "w", repeat: false }, true)).toBe(false);
  });
});
