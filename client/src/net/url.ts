/** Builds the game WebSocket URL from the page location, matching its security. */
export function socketUrl(loc: Pick<Location, "protocol" | "host">): string {
  const scheme = loc.protocol === "https:" ? "wss:" : "ws:";
  return `${scheme}//${loc.host}/ws`;
}
