// Must match the server's rule (server/internal/access/names.go).
const NAME_PATTERN = /^[\p{L}\p{Nd} _-]{1,16}$/u;

/** Returns the trimmed name if the server will accept it. */
export function cleanName(input: string): string | undefined {
  const name = input.trim();
  return NAME_PATTERN.test(name) ? name : undefined;
}

export const AVATAR_COLORS: readonly number[] = [
  0xe53935, 0xfb8c00, 0xfdd835, 0x43a047, 0x00acc1, 0x1e88e5, 0x8e24aa, 0xd81b60,
];

export function colorCss(color: number): string {
  return `#${color.toString(16).padStart(6, "0")}`;
}
