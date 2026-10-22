/** Chunk edge length in blocks; must match the server's world.Size. */
export const CHUNK_SIZE = 32;

/**
 * Blocks across a meter, the size blocks used to be; must match the server's world.Meter.
 * Bodies, speeds and distances are written as meters times METER so they keep their real size.
 */
export const METER = 3;
