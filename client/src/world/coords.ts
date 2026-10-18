export interface ChunkPos {
  x: number;
  y: number;
  z: number;
}

export type ChunkKey = string;

export function chunkKey(x: number, y: number, z: number): ChunkKey {
  return `${x},${y},${z}`;
}

export function parseKey(key: ChunkKey): ChunkPos {
  const [x, y, z] = key.split(",").map(Number);
  return { x, y, z };
}
