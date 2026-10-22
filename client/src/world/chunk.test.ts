import { describe, expect, it } from "vitest";
import golden from "../../../server/internal/world/testdata/golden-chunk.hex?raw";
import { blockAt, chunkIndex, decodeChunk } from "./chunk";
import { CHUNK_SIZE as S } from "./constants";

const hexBytes = (hex: string) => Uint8Array.from(hex.trim().match(/../g)!, (b) => parseInt(b, 16));

// Block IDs from server/internal/world/block.go.
const STONE = 1;
const GLASS = 10;

describe("decodeChunk", () => {
  it("decodes the server's golden chunk", () => {
    const c = decodeChunk(hexBytes(golden));
    for (let i = 0; i < 100; i++)
      expect(blockAt(c, i % S, 0, Math.floor(i / S))).toBe((i * 7) % 13);
    expect(blockAt(c, S - 1, S - 1, S - 1)).toBe(GLASS);
    expect(blockAt(c, 5, 8, 5)).toBe(STONE);
  });

  it("keeps uniform chunks compact", () => {
    const c = decodeChunk(Uint8Array.of(2, 1, STONE));
    expect(c.blocks).toBeUndefined();
    expect(blockAt(c, 3, 4, 5)).toBe(STONE);
  });

  it("decodes block IDs past the second built-in range's start", () => {
    // 4100 then 32767 stones: palette [stone, 4100], runs (1, 1) and (32767, 0).
    const c = decodeChunk(Uint8Array.of(2, 2, STONE, 0x84, 0x20, 1, 1, 0xff, 0xff, 0x01, 0));
    expect(blockAt(c, 0, 0, 0)).toBe(4100);
    expect(blockAt(c, 1, 0, 0)).toBe(STONE);
  });

  it("rejects malformed data", () => {
    expect(() => decodeChunk(Uint8Array.of(1, 1, 0))).toThrow(); // an older encoding
    expect(() => decodeChunk(Uint8Array.of(2, 2, 0, 1, 5, 0))).toThrow(); // runs stop short
    expect(() => decodeChunk(Uint8Array.of(2, 1, 0, 9))).toThrow(); // trailing bytes
  });

  it("indexes x fastest, then z, then y like the server", () => {
    expect(chunkIndex(1, 0, 0)).toBe(1);
    expect(chunkIndex(0, 0, 1)).toBe(S);
    expect(chunkIndex(0, 1, 0)).toBe(S * S);
  });
});
