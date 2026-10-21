import type { Look } from "./look";

// Each *.look.ts file here is a look, named by its file; adding a look is adding a file.
const files = import.meta.glob<Look>("./*.look.ts", { eager: true, import: "default" });

export const LOOKS: ReadonlyMap<string, Look> = new Map(
  Object.entries(files).map(([path, look]) => [path.slice(2, -".look.ts".length), look]),
);

export const DEFAULT_LOOK = "today";

/** The named look, or the default one if it no longer exists. */
export function lookById(id: string): Look {
  return LOOKS.get(id) ?? (LOOKS.get(DEFAULT_LOOK) as Look);
}
