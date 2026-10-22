import type { Box } from "./shapes";

/** How a block meets drawn wire, as bit flags in RenderTable.wiring. */
export const Wiring = {
  WIRE: 1, // wire itself, drawn joined to what's around it
  JOINS: 2, // carries or uses power, so wire reaches out to it
  HOLDS: 4, // solid, so wire lies on it or runs up it
} as const;

/** The wiring of the block at an offset from the wire. */
export type Around = (dx: number, dy: number, dz: number) => number;

type Dir = readonly [number, number, number];

const FLATWAYS: readonly Dir[] = [
  [1, 0, 0],
  [-1, 0, 0],
  [0, 0, 1],
  [0, 0, -1],
];
const UP: Dir = [0, 1, 0];
const DOWN: Dir = [0, -1, 0];
const LO = 3; // wire is a quarter of a block wide, from 3 to 5 eighths
const HI = 5;
const FULL = 8;

const conducts = (flags: number) => (flags & (Wiring.WIRE | Wiring.JOINS)) !== 0;
const opposite = (d: Dir): Dir => [-d[0], -d[1], -d[2]];

/** The boxes, in eighths of a block, that draw a wire joined to the blocks around it. */
export function wireBoxes(around: Around): Box[] {
  const at = (d: Dir, dy = 0) => around(d[0], d[1] + dy, d[2]);
  if (at(DOWN) & Wiring.HOLDS) return onFloor(around);
  const wall = FLATWAYS.find((d) => at(d) & Wiring.HOLDS);
  const ways = (wall ? [...alongWall(wall), UP, DOWN] : [...FLATWAYS, UP, DOWN]).filter((d) =>
    conducts(at(d)),
  );
  if (!wall && ways.length === 0) return onFloor(around);
  const center: Box = wall ? onSide(wall, LO, HI, LO, HI) : [LO, LO, LO, HI, HI, HI];
  return withArms(center, ways, [UP]);
}

/** Wire on the floor: arms to what it joins, up steps and walls toward wire overhead. */
function onFloor(around: Around): Box[] {
  const at = (d: Dir, dy = 0) => around(d[0], d[1] + dy, d[2]);
  const roomOver = !(at(UP) & Wiring.HOLDS);
  const stepsUp = (d: Dir) => roomOver && (at(d, 1) & Wiring.WIRE) !== 0;
  const stepsDown = (d: Dir) => !(at(d) & Wiring.HOLDS) && (at(d, -1) & Wiring.WIRE) !== 0;
  const ways = FLATWAYS.filter((d) => conducts(at(d)) || stepsUp(d) || stepsDown(d));
  const boxes = withArms([LO, 0, LO, HI, 1, HI], ways, FLATWAYS);
  for (const d of FLATWAYS.filter(stepsUp)) boxes.push(onSide(d, 1, FULL, LO, HI));
  if (conducts(at(UP))) {
    const wall = FLATWAYS.find((d) => at(d) & Wiring.HOLDS);
    boxes.push(wall ? onSide(wall, 1, FULL, LO, HI) : [LO, 1, LO, HI, FULL, HI]);
  }
  return boxes;
}

/**
 * A center box with an arm out each of `ways`; a lone arm runs straight through,
 * and with none, arms go out every one of `idle`.
 */
function withArms(center: Box, ways: readonly Dir[], idle: readonly Dir[]): Box[] {
  let arms = ways;
  if (ways.length === 0) arms = [...idle, ...idle.map(opposite)];
  else if (ways.length === 1) arms = [ways[0], opposite(ways[0])];
  return [center, ...unique(arms).map((d) => reach(center, d))];
}

function unique(dirs: readonly Dir[]): Dir[] {
  return dirs.filter((d, i) => dirs.findIndex((e) => e.join() === d.join()) === i);
}

/** Stretches the center box from its side toward d out to the edge of the block. */
function reach(center: Box, d: Dir): Box {
  const box = [...center] as [number, number, number, number, number, number];
  const axis = d.findIndex((v) => v !== 0);
  if (d[axis] > 0) [box[axis], box[axis + 3]] = [center[axis + 3], FULL];
  else [box[axis], box[axis + 3]] = [0, center[axis]];
  return box;
}

/** The two flat ways along a wall that faces d. */
function alongWall(d: Dir): Dir[] {
  return d[0] !== 0
    ? [
        [0, 0, 1],
        [0, 0, -1],
      ]
    : [
        [1, 0, 0],
        [-1, 0, 0],
      ];
}

/** A thin box against the wall toward d: y from y0 to y1, and a0 to a1 along the wall. */
function onSide(d: Dir, y0: number, y1: number, a0: number, a1: number): Box {
  const [lo, hi] = d[0] + d[2] > 0 ? [FULL - 1, FULL] : [0, 1];
  return d[0] !== 0 ? [lo, y0, a0, hi, y1, a1] : [a0, y0, lo, a1, y1, hi];
}
