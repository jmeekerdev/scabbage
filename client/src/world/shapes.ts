/** Block geometry; numbers match pb.Shape and server/internal/world/shape.go. */
export const Shape = {
  CUBE: 0,
  SLAB: 1,
  STAIRS_NORTH: 2, // tall half toward -z; east, south and west follow
  STAIRS_EAST: 3,
  STAIRS_SOUTH: 4,
  STAIRS_WEST: 5,
  TORCH: 6,
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
];

const inBlocks = SHAPE_BOXES.map((boxes) =>
  boxes.map((b) => b.map((v) => v / 8) as unknown as Box),
);

export const FULL_BLOCK: readonly Box[] = inBlocks[Shape.CUBE];

/** What players collide with in a block of this shape; non-solid blocks have nothing. */
export function collisionBoxes(shape: number, solid: boolean): readonly Box[] {
  return solid ? (inBlocks[shape] ?? FULL_BLOCK) : [];
}

export const isStairs = (shape: number) =>
  shape >= Shape.STAIRS_NORTH && shape <= Shape.STAIRS_WEST;

/** The block the player holds for a placed block: north-facing stairs stand for all four. */
export function heldBlock(id: number, shape: number): number {
  return isStairs(shape) ? id - (shape - Shape.STAIRS_NORTH) : id;
}

/**
 * Stairs come as four consecutive block IDs, north first. Picks the one whose tall half
 * is away from a player facing yaw (0 looks toward -z), so they walk up it.
 */
export function stairsFacing(northId: number, yaw: number): number {
  const quarter = Math.round(yaw / (Math.PI / 2));
  const facing = [0, 3, 2, 1][((quarter % 4) + 4) % 4]; // yaw turns counter-clockwise from north
  return northId + facing;
}
