import { describe, expect, it } from "vitest";
import { mapFileName, mapImportName, MAX_MAP_BYTES } from "./map-files";

describe("map files", () => {
  it("spots /map import NAME, which picks a file here instead of going to the server", () => {
    expect(mapImportName("/map import Castle")).toBe("Castle");
    expect(mapImportName("  /MAP   Import   Pip  ")).toBe("Pip");
    expect(mapImportName("/map import")).toBeUndefined();
    expect(mapImportName("/map import two words")).toBeUndefined();
    expect(mapImportName("/map export Castle")).toBeUndefined();
    expect(mapImportName("map import Castle")).toBeUndefined();
  });

  it("names downloads after the map, with nothing a file system dislikes", () => {
    expect(mapFileName("Castle")).toBe("Castle.scabmap");
    expect(mapFileName('a/b\\c:d*?"<>|')).toBe("a_b_c_d______.scabmap");
    expect(mapFileName("")).toBe("map.scabmap");
  });

  it("matches the server's file limit", () => {
    expect(MAX_MAP_BYTES).toBe(4 << 20);
  });
});
