/** Player skins: one 64×32 RGBA picture (rows top-down) in the classic voxel-skin layout. */
export const SKIN_W = 64;
export const SKIN_H = 32;

export type Part = "head" | "body" | "arm" | "leg";
export const FACES = ["front", "back", "left", "right", "top", "bottom"] as const;
export type Face = (typeof FACES)[number];
export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

/** Where each box's net starts and its size in pixels: w across, h tall, d deep. */
export const PARTS: Record<Part, { u: number; v: number; w: number; h: number; d: number }> = {
  head: { u: 0, v: 0, w: 8, h: 8, d: 8 },
  body: { u: 16, v: 16, w: 8, h: 12, d: 4 },
  arm: { u: 40, v: 16, w: 4, h: 12, d: 4 },
  leg: { u: 0, v: 16, w: 4, h: 12, d: 4 },
};

/** The rectangle holding one face of a part, seen from outside with its top up. */
export function faceRect(part: Part, face: Face): Rect {
  const { u, v, w, h, d } = PARTS[part];
  const rects: Record<Face, Rect> = {
    top: { x: u + d, y: v, w, h: d },
    bottom: { x: u + d + w, y: v, w, h: d },
    right: { x: u, y: v + d, w: d, h },
    front: { x: u + d, y: v + d, w, h },
    left: { x: u + d + w, y: v + d, w: d, h },
    back: { x: u + 2 * d + w, y: v + d, w, h },
  };
  return rects[face];
}

type Vec = [number, number, number];
// Screen right and up for each face as drawn in its rectangle; the figure faces -z.
const AXES: Record<Face, { normal: Vec; right: Vec; up: Vec }> = {
  front: { normal: [0, 0, -1], right: [-1, 0, 0], up: [0, 1, 0] },
  back: { normal: [0, 0, 1], right: [1, 0, 0], up: [0, 1, 0] },
  right: { normal: [1, 0, 0], right: [0, 0, -1], up: [0, 1, 0] },
  left: { normal: [-1, 0, 0], right: [0, 0, 1], up: [0, 1, 0] },
  top: { normal: [0, 1, 0], right: [-1, 0, 0], up: [0, 0, 1] },
  bottom: { normal: [0, -1, 0], right: [-1, 0, 0], up: [0, 0, -1] },
};
const INSET = 0.02; // pixels, so edges never sample the neighboring face

/** UVs for a centered box's vertices so each face shows its part of the skin. */
export function boxUVs(
  part: Part,
  positions: ArrayLike<number>,
  normals: ArrayLike<number>,
): Float32Array {
  const n = positions.length / 3;
  const half = [0, 1, 2].map((axis) => {
    let m = 0;
    for (let i = 0; i < n; i++) m = Math.max(m, Math.abs(positions[i * 3 + axis] as number));
    return m;
  });
  const out = new Float32Array(n * 2);
  for (let i = 0; i < n; i++) {
    const p = [0, 1, 2].map((a) => positions[i * 3 + a] as number);
    const face = FACES.find((f) =>
      AXES[f].normal.every((c, a) => c === Math.round(normals[i * 3 + a] as number)),
    );
    const { right, up } = AXES[face ?? "front"];
    const along = (axis: Vec) =>
      (axis.reduce((sum, c, a) => sum + (c * (p[a] as number)) / (half[a] || 1), 0) + 1) / 2;
    const r = faceRect(part, face ?? "front");
    out[i * 2] = (r.x + INSET + along(right) * (r.w - 2 * INSET)) / SKIN_W;
    out[i * 2 + 1] = (r.y + INSET + (1 - along(up)) * (r.h - 2 * INSET)) / SKIN_H;
  }
  return out;
}

type RGBA = [number, number, number, number];
const SKIN_TONE: RGBA = [240, 200, 160, 255];
const HAIR: RGBA = [90, 58, 34, 255];
const SHOES: RGBA = [50, 40, 35, 255];
const EYE_WHITE: RGBA = [250, 250, 250, 255];
const EYE: RGBA = [32, 32, 32, 255];

const rgb = (color: number, k = 1): RGBA => [
  Math.round(((color >> 16) & 255) * k),
  Math.round(((color >> 8) & 255) * k),
  Math.round((color & 255) * k),
  255,
];

/** Paints rows `from` (inclusive) to `to` (exclusive) of a face one color. */
export function fillFace(
  skin: Uint8Array,
  part: Part,
  face: Face,
  c: RGBA,
  from = 0,
  to = Infinity,
): void {
  const r = faceRect(part, face);
  for (let y = r.y + from; y < r.y + Math.min(r.h, to); y++)
    for (let x = r.x; x < r.x + r.w; x++) skin.set(c, (y * SKIN_W + x) * 4);
}

const sides: Face[] = ["front", "back", "left", "right"];

/** The look of a player who hasn't made a skin: a shirt and trousers of their color. */
export function defaultSkin(color: number): Uint8Array {
  const skin = new Uint8Array(SKIN_W * SKIN_H * 4);
  for (const face of FACES) {
    fillFace(skin, "head", face, face === "top" || face === "back" ? HAIR : SKIN_TONE);
    fillFace(skin, "body", face, rgb(color));
    fillFace(skin, "arm", face, face === "bottom" ? SKIN_TONE : rgb(color));
    fillFace(skin, "leg", face, face === "bottom" ? SHOES : rgb(color, 0.55));
  }
  for (const face of sides) {
    fillFace(skin, "head", face, HAIR, 0, face === "front" ? 2 : 3);
    fillFace(skin, "arm", face, SKIN_TONE, 9);
    fillFace(skin, "leg", face, SHOES, 10);
  }
  const eyes = faceRect("head", "front");
  const dot = (x: number, c: RGBA) => skin.set(c, ((eyes.y + 4) * SKIN_W + eyes.x + x) * 4);
  dot(1, EYE_WHITE);
  dot(2, EYE);
  dot(5, EYE);
  dot(6, EYE_WHITE);
  return skin;
}

export const FRONT_W = 16;
export const FRONT_H = 32;

/** A flat 16×32 picture of the figure from the front, e.g. for an admin to approve. */
export function frontView(skin: Uint8Array): Uint8Array {
  const out = new Uint8Array(FRONT_W * FRONT_H * 4);
  const place = (part: Part, x: number, y: number) => {
    const r = faceRect(part, "front");
    for (let row = 0; row < r.h; row++) {
      const from = ((r.y + row) * SKIN_W + r.x) * 4;
      out.set(skin.subarray(from, from + r.w * 4), ((y + row) * FRONT_W + x) * 4);
    }
  };
  place("head", 4, 0);
  place("body", 4, 8);
  place("arm", 0, 8);
  place("arm", 12, 8);
  place("leg", 4, 20);
  place("leg", 8, 20);
  return out;
}
