import { describe, expect, it } from "vitest";
import { isMenuKey } from "./hotkey";

const press = (code: string, more: Partial<KeyboardEvent> = {}) =>
  ({
    code,
    repeat: false,
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    target: null,
    ...more,
  }) as KeyboardEvent;
const box = (tag: string) => ({ closest: (s: string) => (s.includes(tag) ? {} : null) });

describe("menu hotkeys", () => {
  it("answer a plain press of their key", () => {
    expect(isMenuKey(press("KeyB"), "KeyB")).toBe(true);
    expect(isMenuKey(press("KeyV"), "KeyB")).toBe(false);
  });

  it("ignore held keys and shortcuts like Ctrl+B", () => {
    expect(isMenuKey(press("KeyB", { repeat: true }), "KeyB")).toBe(false);
    expect(isMenuKey(press("KeyB", { ctrlKey: true }), "KeyB")).toBe(false);
    expect(isMenuKey(press("KeyB", { metaKey: true }), "KeyB")).toBe(false);
  });

  it("leave typing alone", () => {
    const target = box("input") as unknown as EventTarget;
    expect(isMenuKey(press("KeyB", { target }), "KeyB")).toBe(false);
  });
});
