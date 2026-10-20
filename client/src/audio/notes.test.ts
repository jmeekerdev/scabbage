import { describe, expect, it } from "vitest";
import { noteTable, noteToPlay } from "./notes";

const t = noteTable([
  { id: 1, name: "stone" },
  { id: 10, name: "note_block" },
  { id: 11, name: "note_block_on" },
  { id: 12, name: "note_block_1" },
  { id: 13, name: "note_block_1_on" },
  { id: 24, name: "note_block_7" },
]);

describe("noteToPlay", () => {
  it("sounds a note block when power reaches it, rising up the C major scale", () => {
    expect(noteToPlay(t, 10, 11)).toBe(72);
    expect(noteToPlay(t, 12, 13)).toBe(74);
  });

  it("sounds the new pitch when retuned, even unpowered", () => {
    expect(noteToPlay(t, 12, 24)).toBe(84);
  });

  it("stays quiet when power leaves, when placed and for other blocks", () => {
    expect(noteToPlay(t, 11, 10)).toBeUndefined();
    expect(noteToPlay(t, 0, 10)).toBeUndefined();
    expect(noteToPlay(t, 0, 1)).toBeUndefined();
  });
});
