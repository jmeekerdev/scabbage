import { Emote } from "../pb/scabbage/v1/net_pb";

/** The emote wheel, clockwise from the top. */
export const EMOTES: readonly { emote: Emote; emoji: string; label: string }[] = [
  { emote: Emote.WAVE, emoji: "👋", label: "Wave" },
  { emote: Emote.HAPPY, emoji: "😀", label: "Happy" },
  { emote: Emote.LAUGH, emoji: "😂", label: "Laugh" },
  { emote: Emote.LOVE, emoji: "❤️", label: "Love" },
  { emote: Emote.THUMBS_UP, emoji: "👍", label: "Thumbs up" },
  { emote: Emote.WOW, emoji: "😮", label: "Wow" },
  { emote: Emote.SAD, emoji: "😢", label: "Sad" },
  { emote: Emote.PARTY, emoji: "🎉", label: "Party" },
];

export function emojiFor(e: Emote): string | undefined {
  return EMOTES.find((x) => x.emote === e)?.emoji;
}

/** Picks a wheel slot from relative pointer movement, e.g. a locked mouse. */
export class WheelAim {
  private x = 0;
  private y = 0;

  constructor(
    private readonly count: number,
    private readonly minDistance: number,
  ) {}

  /** Adds movement, keeping the aim within twice minDistance so changing slots stays quick. */
  move(dx: number, dy: number): void {
    const [x, y] = [this.x + dx, this.y + dy];
    const scale = Math.min(1, (2 * this.minDistance) / (Math.hypot(x, y) || 1));
    [this.x, this.y] = [x * scale, y * scale];
  }

  slot(): number | undefined {
    if (Math.hypot(this.x, this.y) < this.minDistance) return undefined;
    const turn = Math.atan2(this.x, -this.y) / (2 * Math.PI); // 0 at the top, clockwise
    return (Math.round(turn * this.count) + this.count) % this.count;
  }

  reset(): void {
    this.x = this.y = 0;
  }
}
