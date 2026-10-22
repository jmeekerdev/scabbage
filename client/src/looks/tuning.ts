import { at, LIGHT_COLOURS, LOOK_RANGES } from "./check";
import type { Look, Rgb } from "./look";

export interface NumberControl {
  path: string;
  min: number;
  max: number;
  step: number;
  value: number;
}

export interface ColourControl {
  path: string;
  value: Rgb;
}

// Changing these rebuilds every loaded chunk, so they are left to the look's file.
const REMESHING = new Set(["shape.softCorners"]);
// Optional parts a control may create; others, like the sun or outline, must already be there.
const CREATABLE = new Set(["after"]);
const DEFAULTS: Record<string, number> = { "after.saturation": 1 };

const EXTRA: [path: string, min: number, max: number, step?: number][] = [
  ["light.sun.dir.x", -1, 1],
  ["light.sun.dir.y", -1, 1],
  ["light.sun.dir.z", -1, 1],
  ["light.sun.bands", 0, 8, 1],
  ...LIGHT_COLOURS.flatMap((p) =>
    ["r", "g", "b"].map((k): [string, number, number] => [`${p}.${k}`, 0, 2]),
  ),
];

const parentOf = (path: string) => path.slice(0, path.lastIndexOf("."));

function offered(look: Look, path: string): boolean {
  const parent = parentOf(path);
  return !REMESHING.has(path) && (at(look, parent) !== undefined || CREATABLE.has(parent));
}

/** The sliders that apply to a look, with their current values. */
export function numberControls(look: Look): NumberControl[] {
  const all = [...LOOK_RANGES, ...EXTRA];
  return all
    .filter(([path]) => offered(look, path))
    .map(([path, min, max, step]) => ({
      path,
      min,
      max,
      step: step ?? (max - min) / 200,
      value: (at(look, path) as number | undefined) ?? DEFAULTS[path] ?? 0,
    }));
}

/** The look's sRGB colours that can be picked: sky stops, and stitches and lines when it has them. */
export function colourControls(look: Look): ColourControl[] {
  const paths = [
    ...Object.keys(look.sky).map((k) => `sky.${k}`),
    "shape.stitch",
    "after.outline.color",
  ];
  return paths.flatMap((path) => {
    const value = at(look, path) as Rgb | undefined;
    return value ? [{ path, value }] : [];
  });
}

/** A copy of the look with one value changed, creating the parts on the way that are missing. */
export function setAt(look: Look, path: string, value: unknown): Look {
  const set = (o: Record<string, unknown> | undefined, keys: string[]): Record<string, unknown> => {
    const [k, ...rest] = keys;
    const next = { ...o };
    next[k] = rest.length ? set(o?.[k] as Record<string, unknown> | undefined, rest) : value;
    return next;
  };
  return set(look as unknown as Record<string, unknown>, path.split(".")) as unknown as Look;
}

const byte = (v: number) => Math.round(Math.min(1, Math.max(0, v)) * 255);

/** An sRGB colour as a colour input's "#rrggbb". */
export function toHex(c: Rgb): string {
  return `#${[c.r, c.g, c.b].map((v) => byte(v).toString(16).padStart(2, "0")).join("")}`;
}

export function fromHex(h: string): Rgb {
  const n = parseInt(h.slice(1), 16);
  return { r: (n >> 16) / 255, g: ((n >> 8) & 255) / 255, b: (n & 255) / 255 };
}

/** A look as a file for the looks folder, named from the look's name. */
export function lookFile(look: Look): { name: string; text: string } {
  const slug = look.name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
  const round = (_: string, v: unknown) => (typeof v === "number" ? Math.round(v * 1e4) / 1e4 : v);
  const body = JSON.stringify(look, round, 2);
  const text = `import type { Look } from "./look";\n\n// Made with the look tuner.\nexport default ${body} satisfies Look;\n`;
  return { name: `${slug || "new-look"}.look.ts`, text };
}
