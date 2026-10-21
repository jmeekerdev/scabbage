const TYPING = "input, select, textarea, [contenteditable]";

/** Whether a key press is a plain, fresh press of `code` that isn't typing into a box. */
export function isMenuKey(e: KeyboardEvent, code: string): boolean {
  if (e.code !== code || e.repeat || e.ctrlKey || e.metaKey || e.altKey) return false;
  const target = e.target as Partial<Element> | null;
  return !target?.closest?.(TYPING);
}
