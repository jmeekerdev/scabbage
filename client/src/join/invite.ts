// Must match the server's alphabet (server/internal/access/codes.go).
const CODE_ALPHABET = "23456789ABCDEFGHJKMNPQRSTUVWXYZ";

/** Uppercases and keeps only code characters, so "abcd-efgh" becomes "ABCDEFGH". */
export function normalizeCode(input: string): string {
  return [...input.toUpperCase()].filter((c) => CODE_ALPHABET.includes(c)).join("");
}

export function formatCode(code: string): string {
  return code.length === 8 ? `${code.slice(0, 4)}-${code.slice(4)}` : code;
}

/** Extracts the code from a `#join=CODE` URL hash. */
export function parseInvite(hash: string): string | undefined {
  const code = normalizeCode(new URLSearchParams(hash.replace(/^#/, "")).get("join") ?? "");
  return code || undefined;
}

export function inviteLink(origin: string, code: string): string {
  return `${origin}/#join=${formatCode(code)}`;
}
