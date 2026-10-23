import type { Cell } from "../../player/brush";
import { coordinate, type Job } from "../job";
import { FACINGS, placeTemplate, TEMPLATES, type Facing, type Size } from "./templates";

const AHEAD = 3; // blocks in front of the feet a /build's front goes

const USAGE = `Use /build ${[...TEMPLATES.keys()].join("|")} [w d h] [block] [north|east|south|west] [at x y z].`;

// The way each quarter turn of yaw looks: yaw 0 looks north, turning left raises it.
const BY_YAW: Facing[] = ["north", "west", "south", "east"];
const STEP: Record<Facing, [number, number]> = {
  north: [0, -1],
  east: [1, 0],
  south: [0, 1],
  west: [-1, 0],
};
const BEHIND: Record<Facing, Facing> = {
  north: "south",
  east: "west",
  south: "north",
  west: "east",
};

/** The way a yaw looks, to the nearest quarter turn. */
export function lookFacing(yaw: number): Facing {
  const quarter = Math.round(yaw / (Math.PI / 2));
  return BY_YAW[((quarter % 4) + 4) % 4]!;
}

/**
 * Reads /build's arguments into a job: by default just ahead of the feet with its front toward
 * the player. Or says what is wrong.
 */
export function parseBuild(
  arg: string,
  feet: Cell,
  yaw: number,
  names: ReadonlyMap<string, number>,
): Extract<Job, { kind: "build" }> | string {
  const [template, ...words] = arg.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (!template) return USAGE;
  const sizeWords = words.slice(0, 3).filter((w) => /^\d+$/.test(w));
  if (sizeWords.length && sizeWords.length < 3) return `Give all three of w d h, or none. ${USAGE}`;
  const [w, d, h] = sizeWords.map(Number);
  const size: Partial<Size> = sizeWords.length ? { w, d, h } : {};
  const looks = lookFacing(yaw);
  let facing = BEHIND[looks];
  let material: string | undefined;
  let at: Cell | undefined;
  for (let i = sizeWords.length; i < words.length; i++) {
    const word = words[i]!;
    if (FACINGS.includes(word as Facing)) facing = word as Facing;
    else if (word === "at") {
      at = spot(words.slice(i + 1, i + 4), feet);
      if (!at) return `Use at x y z, where ~ means where you are. ${USAGE}`;
      i += 3;
    } else material = word;
  }
  const [sx, sz] = STEP[looks];
  const from = at ?? { x: feet.x + sx * AHEAD, y: feet.y, z: feet.z + sz * AHEAD };
  const cells = placeTemplate(template, size, from, facing, material, names);
  return typeof cells === "string" ? cells : { kind: "build", template, cells };
}

/** x y z, each a whole number or ~ / ~n for the feet's own plus n. */
function spot(words: string[], feet: Cell): Cell | undefined {
  const n = words.map((w, i) => coordinate(w, feet, i));
  if (n.length !== 3 || n.some((v) => v === undefined)) return undefined;
  return { x: n[0]!, y: n[1]!, z: n[2]! };
}
