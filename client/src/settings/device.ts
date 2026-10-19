import type { Preset } from "./settings";

/** Hardware hints browsers expose; memoryGb is missing outside Chromium. */
export interface Device {
  touch: boolean;
  cores: number;
  memoryGb?: number;
}

export function readDevice(): Device {
  const nav = navigator as Navigator & { deviceMemory?: number };
  return {
    touch: matchMedia("(pointer: coarse)").matches,
    cores: nav.hardwareConcurrency || 2,
    memoryGb: nav.deviceMemory,
  };
}

/** The starting preset for a device; phones run hot, so they never start on high. */
export function deviceTier({ touch, cores, memoryGb = Infinity }: Device): Preset {
  if (cores <= 2 || memoryGb <= (touch ? 3 : 2)) return "low";
  if (touch || cores <= 4 || memoryGb <= 4) return "medium";
  return "high";
}
