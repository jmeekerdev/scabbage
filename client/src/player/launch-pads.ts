import { METER } from "../world/constants";

export type Velocity = { x: number; y: number; z: number };

const ALONG = 9 * METER; // with UP_ALONG, about six meters forward
const UP_ALONG = 9 * METER;
const UP = 12.5 * METER; // about 2.8 meters high, just under the height that hurts on landing

// Each facing's name suffix and the way it throws; mirrors world.LaunchPad.
const THROWS: Record<string, Velocity> = {
  "": { x: 0, y: UP_ALONG, z: -ALONG },
  _east: { x: ALONG, y: UP_ALONG, z: 0 },
  _south: { x: 0, y: UP_ALONG, z: ALONG },
  _west: { x: -ALONG, y: UP_ALONG, z: 0 },
  _up: { x: 0, y: UP, z: 0 },
};

/** How each powered launch pad throws players who stand on it, by block ID. */
export function launchPads(blocks: readonly { id: number; name: string }[]): Map<number, Velocity> {
  const out = new Map<number, Velocity>();
  for (const b of blocks) {
    const powered = b.name.match(/^launch_pad(_east|_south|_west|_up)?_on$/);
    if (powered) out.set(b.id, THROWS[powered[1] ?? ""]!);
  }
  return out;
}
