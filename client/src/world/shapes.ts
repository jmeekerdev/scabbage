/** Block geometry; numbers match pb.Shape and server/internal/world/shape.go. */
export const Shape = {
  CUBE: 0,
  SLAB: 1,
  STAIRS_NORTH: 2, // tall half toward -z; east, south and west follow
  STAIRS_EAST: 3,
  STAIRS_SOUTH: 4,
  STAIRS_WEST: 5,
  TORCH: 6,
  FLAT: 7, // a thin layer on the floor, like wire
  TUFT: 8, // a few blades of grass
  FLOWER: 9, // a stem with a head on top
} as const;

/** An axis-aligned box inside a block: x0, y0, z0, x1, y1, z1. */
export type Box = readonly [number, number, number, number, number, number];

const LOWER: Box = [0, 0, 0, 8, 4, 8];

/** Each shape's boxes in eighths of a block, as drawn. */
export const SHAPE_BOXES: readonly (readonly Box[])[] = [
  [[0, 0, 0, 8, 8, 8]],
  [LOWER],
  [LOWER, [0, 4, 0, 8, 8, 4]],
  [LOWER, [4, 4, 0, 8, 8, 8]],
  [LOWER, [0, 4, 4, 8, 8, 8]],
  [LOWER, [0, 4, 0, 4, 8, 8]],
  [[3, 0, 3, 5, 5, 5]],
  [[0, 0, 0, 8, 1, 8]],
  [
    [1, 0, 1, 2, 5, 2],
    [5, 0, 2, 6, 7, 3],
    [3, 0, 5, 4, 6, 6],
    [6, 0, 6, 7, 4, 7],
  ],
  [
    [4, 0, 4, 5, 5, 5],
    [2, 5, 2, 7, 8, 7],
  ],
];

const inBlocks = SHAPE_BOXES.map((boxes) =>
  boxes.map((b) => b.map((v) => v / 8) as unknown as Box),
);

export const FULL_BLOCK: readonly Box[] = inBlocks[Shape.CUBE];

/** What players collide with in a block of this shape; non-solid blocks have nothing. */
export function collisionBoxes(shape: number, solid: boolean): readonly Box[] {
  return solid ? (inBlocks[shape] ?? FULL_BLOCK) : [];
}

const PIXELS_PER_EIGHTH = 2;

/**
 * Which pixels of a 16×16 icon (rows top-down) the shape covers, seen from the side along x.
 * Flat blocks show whole, since a sliver would be unreadable.
 */
export function sideView(shape: number): (x: number, y: number) => boolean {
  if (shape === Shape.FLAT) return () => true;
  const boxes = SHAPE_BOXES[shape] ?? SHAPE_BOXES[Shape.CUBE];
  return (x, y) => {
    const z = (15 - x) / PIXELS_PER_EIGHTH;
    const up = (15 - y) / PIXELS_PER_EIGHTH;
    return boxes.some((b) => z >= b[2] && z < b[5] && up >= b[1] && up < b[4]);
  };
}

export const isStairs = (shape: number) =>
  shape >= Shape.STAIRS_NORTH && shape <= Shape.STAIRS_WEST;

/**
 * Stairs come as four consecutive block IDs, north first. Picks the one whose tall half
 * is away from a player facing yaw (0 looks toward -z), so they walk up it.
 */
export function stairsFacing(northId: number, yaw: number): number {
  return northId + compassFacing(yaw);
}

/** The way a player facing yaw looks: 0 north, 1 east, 2 south, 3 west. */
function compassFacing(yaw: number): number {
  const quarter = Math.round(yaw / (Math.PI / 2));
  return [0, 3, 2, 1][((quarter % 4) + 4) % 4]; // yaw turns counter-clockwise from north
}

/**
 * Pistons come as north, east, south, west, up and down, each followed by its powered state.
 * Picks the one that pushes where the player looks.
 */
export function pistonFacing(northId: number, yaw: number, pitch: number): number {
  const steep = Math.PI / 4;
  const facing = pitch > steep ? 4 : pitch < -steep ? 5 : compassFacing(yaw);
  return northId + 2 * facing;
}
