import { DoubleSide, FrontSide, Texture } from "three";
import { describe, expect, it } from "vitest";
import { lookById } from "../looks/registry";
import { opaqueMaterial, translucentMaterial } from "./chunk-material";

describe("chunk materials", () => {
  it("draw water's surface from below too, so it shows from underwater", () => {
    expect(translucentMaterial(new Texture(), lookById("today")).side).toBe(DoubleSide);
  });

  it("cull the backs of solid blocks", () => {
    expect(opaqueMaterial(new Texture(), lookById("today")).side).toBe(FrontSide);
  });
});
