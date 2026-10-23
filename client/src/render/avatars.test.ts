import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Sprite } from "three";
import { Avatar } from "./avatars";

// Name tags draw on a canvas; tests have no DOM, so hand out a blank one.
beforeEach(() => {
  vi.stubGlobal("document", {
    createElement: () => ({ width: 0, height: 0, getContext: () => null }),
  });
});
afterEach(() => vi.unstubAllGlobals());

function tagOf(a: Avatar): Sprite | undefined {
  return a.root.children.find((c): c is Sprite => (c as Sprite).isSprite);
}

describe("Avatar name tag", () => {
  it("shows a tag for a named player", () => {
    expect(tagOf(new Avatar("Ann", 0xff0000, new Uint8Array()))).toBeDefined();
  });

  it("frees the old tag when going away replaces it", () => {
    const a = new Avatar("Ann", 0xff0000, new Uint8Array());
    const old = tagOf(a)!;
    const freed = vi.spyOn(old.material, "dispose");
    a.setAway(true);
    expect(freed).toHaveBeenCalled();
    expect(a.root.children.filter((c) => (c as Sprite).isSprite)).toEqual([tagOf(a)]);
  });
});
