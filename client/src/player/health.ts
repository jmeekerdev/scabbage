import { Harm } from "../pb/scabbage/v1/net_pb";

/** Full health, matching server/internal/vitals; two points per heart. */
export const MAX_HEALTH = 20;

export type Heart = "full" | "half" | "empty";

export function hearts(health: number): Heart[] {
  return Array.from({ length: MAX_HEALTH / 2 }, (_, i) => {
    const left = health - i * 2;
    return left >= 2 ? "full" : left === 1 ? "half" : "empty";
  });
}

const FAINT_TEXT: Partial<Record<Harm, string>> = {
  [Harm.FALL]: "You fell from too high up. Back to the start!",
  [Harm.PLAYER]: "You got tagged out. Back to the start!",
};

export function faintText(cause: Harm): string {
  return FAINT_TEXT[cause] ?? "Back to the start!";
}
