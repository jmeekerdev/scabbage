import { BoxGeometry } from "three";
import { describe, expect, it } from "vitest";
import { boxUVs, defaultSkin, faceRect, FACES, frontView, PARTS, SKIN_H, SKIN_W } from "./skin";

const px = (skin: Uint8Array, x: number, y: number) =>
  Array.from(skin.subarray((y * SKIN_W + x) * 4, (y * SKIN_W + x) * 4 + 4));

describe("skin layout", () => {
  it("lays each part's faces out side by side without overlapping", () => {
    const used = new Set<number>();
    for (const part of Object.keys(PARTS) as (keyof typeof PARTS)[])
      for (const face of FACES) {
        const r = faceRect(part, face);
        for (let y = r.y; y < r.y + r.h; y++)
          for (let x = r.x; x < r.x + r.w; x++) {
            expect(x).toBeLessThan(SKIN_W);
            expect(y).toBeLessThan(SKIN_H);
            expect(used.has(y * SKIN_W + x)).toBe(false);
            used.add(y * SKIN_W + x);
          }
      }
  });

  it("puts the head's front where classic skins do", () => {
    expect(faceRect("head", "front")).toEqual({ x: 8, y: 8, w: 8, h: 8 });
    expect(faceRect("body", "back")).toEqual({ x: 32, y: 20, w: 8, h: 12 });
  });
});

describe("boxUVs", () => {
  const box = new BoxGeometry(0.5, 0.5, 0.5);
  const uvs = boxUVs("head", box.attributes.position.array, box.attributes.normal.array);
  const pos = box.attributes.position.array;
  const nor = box.attributes.normal.array;
  const vertices = Array.from({ length: pos.length / 3 }, (_, i) => ({
    p: [pos[i * 3], pos[i * 3 + 1], pos[i * 3 + 2]] as number[],
    n: [nor[i * 3], nor[i * 3 + 1], nor[i * 3 + 2]] as number[],
    uv: [(uvs[i * 2] as number) * SKIN_W, (uvs[i * 2 + 1] as number) * SKIN_H],
  }));
  const near = (a: number, b: number) => Math.abs(a - b) < 0.05;

  it("maps the front face's top corners to the top of the front rectangle", () => {
    // The figure faces -z, so its right (+x) is on the viewer's left.
    const front = vertices.filter((v) => v.n[2] === -1);
    const topLeft = front.find((v) => (v.p[0] as number) > 0 && (v.p[1] as number) > 0);
    expect(near(topLeft?.uv[0] ?? -1, 8) && near(topLeft?.uv[1] ?? -1, 8)).toBe(true);
    const bottomRight = front.find((v) => (v.p[0] as number) < 0 && (v.p[1] as number) < 0);
    expect(near(bottomRight?.uv[0] ?? -1, 16) && near(bottomRight?.uv[1] ?? -1, 16)).toBe(true);
  });

  it("joins the top face to the front along their shared edge", () => {
    const top = vertices.filter((v) => v.n[1] === 1 && (v.p[2] as number) < 0);
    for (const v of top) expect(near(v.uv[1] as number, 8)).toBe(true);
  });

  it("keeps every face inside its rectangle", () => {
    for (const v of vertices) {
      const face = FACES.find((f) => normalOf(f).every((c, i) => c === v.n[i]));
      const r = faceRect("head", face ?? "front");
      expect(v.uv[0]).toBeGreaterThanOrEqual(r.x);
      expect(v.uv[0]).toBeLessThanOrEqual(r.x + r.w);
      expect(v.uv[1]).toBeGreaterThanOrEqual(r.y);
      expect(v.uv[1]).toBeLessThanOrEqual(r.y + r.h);
    }
  });
});

function normalOf(face: (typeof FACES)[number]): number[] {
  const n = { right: [1, 0, 0], left: [-1, 0, 0], top: [0, 1, 0], bottom: [0, -1, 0] };
  return n[face as keyof typeof n] ?? (face === "back" ? [0, 0, 1] : [0, 0, -1]);
}

describe("defaultSkin", () => {
  const skin = defaultSkin(0x3366cc);

  it("is opaque everywhere a part is drawn", () => {
    for (const face of FACES) {
      const r = faceRect("body", face);
      expect(px(skin, r.x, r.y)[3]).toBe(255);
    }
  });

  it("wears a shirt of the player's color", () => {
    const r = faceRect("body", "front");
    expect(px(skin, r.x + 3, r.y + 5)).toEqual([0x33, 0x66, 0xcc, 255]);
  });
});

describe("frontView", () => {
  it("shows the head front at the top center", () => {
    const skin = defaultSkin(0x3366cc);
    const view = frontView(skin);
    const r = faceRect("head", "front");
    const at = (x: number, y: number) =>
      Array.from(view.subarray((y * 16 + x) * 4, (y * 16 + x) * 4 + 4));
    expect(at(4, 0)).toEqual(px(skin, r.x, r.y));
    expect(at(11, 7)).toEqual(px(skin, r.x + 7, r.y + 7));
  });
});
