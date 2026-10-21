import { describe, expect, it } from "vitest";
import { lookById } from "../looks/registry";
import { lookDefines } from "./chunk-material";

describe("lookDefines", () => {
  it("draws Today exactly as the game did before looks", () => {
    expect(lookDefines(lookById("today"))).toEqual({
      TINT: "0.06",
      SHADE_TOP: "1.00",
      SHADE_BOTTOM: "0.50",
      SHADE_X: "0.80",
      SHADE_Z: "0.65",
      AO: "0.50",
      LAMP: "vec3(1.00, 0.86, 0.66)",
      LAMP_FADE: "0.82",
    });
  });
});
