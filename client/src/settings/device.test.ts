import { describe, expect, it } from "vitest";
import { deviceTier } from "./device";

describe("deviceTier", () => {
  it("gives strong desktops the high preset", () => {
    expect(deviceTier({ touch: false, cores: 8, memoryGb: 16 })).toBe("high");
    expect(deviceTier({ touch: false, cores: 8 })).toBe("high");
  });

  it("gives modest desktops the medium preset", () => {
    expect(deviceTier({ touch: false, cores: 4, memoryGb: 8 })).toBe("medium");
    expect(deviceTier({ touch: false, cores: 8, memoryGb: 4 })).toBe("medium");
  });

  it("never starts phones and tablets above medium", () => {
    expect(deviceTier({ touch: true, cores: 8, memoryGb: 8 })).toBe("medium");
    expect(deviceTier({ touch: true, cores: 6 })).toBe("medium");
  });

  it("gives weak devices the low preset", () => {
    expect(deviceTier({ touch: true, cores: 2 })).toBe("low");
    expect(deviceTier({ touch: true, cores: 8, memoryGb: 3 })).toBe("low");
    expect(deviceTier({ touch: false, cores: 2, memoryGb: 8 })).toBe("low");
    expect(deviceTier({ touch: false, cores: 4, memoryGb: 2 })).toBe("low");
  });
});
