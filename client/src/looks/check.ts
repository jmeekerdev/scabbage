import { TEXTURE_NAMES } from "../render/textures";
import type { Look, Rgb } from "./look";

// Numbers the shader and passes expect within these bounds; missing optional ones are skipped.
export const LOOK_RANGES: [path: string, min: number, max: number][] = [
  ["surface.tint", 0, 1],
  ["surface.wash", 0, 1],
  ["surface.fibre", 0, 1],
  ["light.faces.top", 0, 4],
  ["light.faces.bottom", 0, 4],
  ["light.faces.x", 0, 4],
  ["light.faces.z", 0, 4],
  ["light.lampFade", 0, 1],
  ["light.sun.wrap", 0, 1],
  ["light.sun.shine", 0, 4],
  ["shape.softCorners", 0, 0.5],
  ["shape.pool", 0, 1],
  ["fog.start", 0, 1],
  ["water.opacity", 0, 1],
  ["after.outline.width", 0.1, 8],
  ["after.outline.strength", 0, 1],
  ["after.paper", 0, 1],
  ["after.tiltShift", 0, 1],
  ["after.vignette", 0, 1],
  ["after.saturation", 0, 2],
];

// Light colours multiply the surface, so they may go past white; the rest are sRGB.
export const LIGHT_COLOURS = ["light.ao", "light.lamp", "light.sun.lit", "light.sun.shade"];

/** The value at a dotted path in a look, or undefined where any part is missing. */
export function at(look: Look, path: string): unknown {
  return path.split(".").reduce<unknown>((o, k) => (o as Record<string, unknown>)?.[k], look);
}

function range(path: string, v: unknown, min: number, max: number): string[] {
  if (v === undefined) return [];
  return typeof v === "number" && v >= min && v <= max
    ? []
    : [`${path} must be ${min}–${max}, is ${String(v)}`];
}

function colour(path: string, c: Rgb | undefined, max: number): string[] {
  if (!c) return [];
  return (["r", "g", "b"] as const).flatMap((k) => range(`${path}.${k}`, c[k], 0, max));
}

function srgbColours(look: Look): [string, Rgb | undefined][] {
  return [
    ...Object.entries(look.sky).map(([k, c]): [string, Rgb] => [`sky.${k}`, c]),
    ...Object.entries(look.surface.palette ?? {}).map(([k, c]): [string, Rgb] => [
      `surface.palette.${k}`,
      c,
    ]),
    ["shape.stitch", look.shape.stitch],
    ["after.outline.color", look.after?.outline?.color],
  ];
}

function sunProblems(look: Look): string[] {
  const sun = look.light.sun;
  if (!sun) return [];
  const { x, y, z } = sun.dir;
  return [
    ...(sun.bands === 0 || (Number.isInteger(sun.bands) && sun.bands >= 2)
      ? []
      : ["light.sun.bands must be 0 or a whole 2+"]),
    ...(x * x + y * y + z * z > 0 ? [] : ["light.sun.dir points nowhere"]),
  ];
}

function textureProblems(look: Look): string[] {
  const known = new Set(TEXTURE_NAMES);
  const missing = (field: string, names: string[]) =>
    names.filter((n) => !known.has(n)).map((n) => `surface.${field} has no texture ${n}`);
  return [
    ...missing("palette", Object.keys(look.surface.palette ?? {})),
    ...missing("solid", look.surface.solid ?? []),
  ];
}

function edgeProblems(look: Look): string[] {
  const { edges, pool, stitch } = look.shape;
  if (edges) return [];
  return [
    ...(pool ? ["shape.pool needs shape.edges"] : []),
    ...(stitch ? ["shape.stitch needs shape.edges"] : []),
  ];
}

/** Everything wrong with a look's config, as readable lines; empty when it is fine. */
export function checkLook(look: Look): string[] {
  return [
    ...(look.name.trim() ? [] : ["name is empty"]),
    ...(look.about.trim() ? [] : ["about is empty"]),
    ...LOOK_RANGES.flatMap(([path, min, max]) => range(path, at(look, path), min, max)),
    ...LIGHT_COLOURS.flatMap((path) => colour(path, at(look, path) as Rgb | undefined, 4)),
    ...srgbColours(look).flatMap(([path, c]) => colour(path, c, 1)),
    ...sunProblems(look),
    ...textureProblems(look),
    ...edgeProblems(look),
  ];
}
