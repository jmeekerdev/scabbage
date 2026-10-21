import { MobKind } from "../pb/scabbage/v1/net_pb";
import type { MobState } from "../player/mobs";
import { el } from "./dom";
import { METRE } from "../world/constants";

const BOSS_NAMES: Partial<Record<MobKind, string>> = {
  [MobKind.KING_SLIME]: "King Slime",
  [MobKind.GOLEM]: "Stone Golem",
};
const SHOW_WITHIN = 24 * METRE; // blocks

export interface BossShown {
  name: string;
  health: number;
  maxHealth: number;
}

/** The nearest boss close enough to fight, if any. */
export function bossNear(
  mobs: readonly MobState[],
  me: { x: number; y: number; z: number },
): BossShown | undefined {
  let best: BossShown | undefined;
  let bestDistance = SHOW_WITHIN;
  for (const m of mobs) {
    const name = BOSS_NAMES[m.kind];
    const d = Math.hypot(m.x - me.x, m.y - me.y, m.z - me.z);
    if (name && m.maxHealth > 0 && d < bestDistance) {
      best = { name, health: m.health, maxHealth: m.maxHealth };
      bestDistance = d;
    }
  }
  return best;
}

/** A health bar along the top of the screen while a boss is near. */
export class BossBar {
  readonly node = el("div", { class: "boss-bar", hidden: true, "aria-live": "polite" });
  private readonly label = el("span");
  private readonly fill = el("div", { class: "fill" });
  private shown = "";

  constructor() {
    this.node.append(this.label, el("div", { class: "track" }, this.fill));
  }

  /** Shows b, or hides the bar; cheap to call every frame. */
  update(b: BossShown | undefined): void {
    const key = b ? `${b.name}:${b.health}/${b.maxHealth}` : "";
    if (key === this.shown) return;
    this.shown = key;
    this.node.hidden = !b;
    if (!b) return;
    this.label.textContent = `${b.name} ${b.health}/${b.maxHealth}`;
    this.fill.style.width = `${(100 * b.health) / b.maxHealth}%`;
  }
}
