interface NamedBlock {
  name: string;
  item?: number; // set when players hold another block for this one
  pixels?: Uint8Array; // custom blocks only, which scripts can't place
}

const OFF_LIMITS = new Set(["air", "bedrock", "script_block"]);

/** The blocks a script may place or give, as the server names them; "air" clears a spot. */
export function scriptBlockNames(blocks: readonly NamedBlock[]): string[] {
  const usable = blocks.filter((b) => !OFF_LIMITS.has(b.name) && !b.item && !b.pixels?.length);
  return [...usable.map((b) => b.name), "air"];
}
