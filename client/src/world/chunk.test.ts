import { describe, expect, it } from "vitest";
import golden from "../../../server/internal/world/testdata/golden-chunk.hex?raw";
import { blockAt, chunkIndex, decodeChunk } from "./chunk";

const hexBytes = (hex: string) => Uint8Array.from(hex.trim().match(/../g)!, (b) => parseInt(b, 16));

// Block IDs from server/internal/world/block.go.
const STONE = 1;
const GLASS = 10;

describe("decodeChunk", () => {
  it("decodes the server's golden chunk", () => {
    const c = decodeChunk(hexBytes(golden));
    for (let i = 0; i < 100; i++)
      expect(blockAt(c, i % 16, 0, Math.floor(i / 16))).toBe((i * 7) % 13);
    expect(blockAt(c, 15, 15, 15)).toBe(GLASS);
    expect(blockAt(c, 5, 8, 5)).toBe(STONE);
  });

  it("keeps uniform chunks compact", () => {
    const c = decodeChunk(Uint8Array.of(1, 1, STONE));
    expect(c.blocks).toBeUndefined();
    expect(blockAt(c, 3, 4, 5)).toBe(STONE);
  });

  it("rejects malformed data", () => {
    expect(() => decodeChunk(Uint8Array.of(2, 1, 0))).toThrow();
    expect(() => decodeChunk(Uint8Array.of(1, 2, 0, 1, 5, 0))).toThrow(); // runs stop short
    expect(() => decodeChunk(Uint8Array.of(1, 1, 0, 9))).toThrow(); // trailing bytes
  });

  it("indexes x fastest, then z, then y like the server", () => {
    expect(chunkIndex(1, 0, 0)).toBe(1);
    expect(chunkIndex(0, 0, 1)).toBe(16);
    expect(chunkIndex(0, 1, 0)).toBe(256);
  });
});
