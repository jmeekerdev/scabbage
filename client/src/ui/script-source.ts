// Scripts made with blocks keep the blocks in a Lua comment after the code, so they reopen as blocks.
const MARK = "scabbage-blocks";
const TRAILER = /\n?--\[(=*)\[scabbage-blocks\n([\s\S]*)\n\]\1\]\n?$/;

/** The Lua with the blocks it came from tucked into a closing long comment. */
export function joinBlocks(lua: string, blocks: object): string {
  const json = JSON.stringify(blocks);
  let level = "";
  while (json.includes(`]${level}]`)) level += "=";
  const body = lua.endsWith("\n") || lua === "" ? lua : lua + "\n";
  return `${body}--[${level}[${MARK}\n${json}\n]${level}]\n`;
}

/** Splits a saved script into its Lua and, if it was made with blocks, the blocks. */
export function splitBlocks(source: string): { lua: string; blocks?: object } {
  const m = TRAILER.exec(source);
  if (!m) return { lua: source };
  try {
    return { lua: source.slice(0, m.index) + (m.index > 0 ? "\n" : ""), blocks: JSON.parse(m[2]) };
  } catch {
    return { lua: source };
  }
}
