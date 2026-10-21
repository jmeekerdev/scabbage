/** Chunk edge length in blocks; must match the server's world.Size. */
export const CHUNK_SIZE = 32;

/**
 * Blocks across a metre, the size blocks used to be; must match the server's world.Metre.
 * Bodies, speeds and distances are written as metres times METRE so they keep their real size.
 */
export const METRE = 3;
