import type { ScriptShown } from "../pb/scabbage/v1/net_pb";

/** The line under the script editor: whether a save worked, or what went wrong last. */
export function scriptStatus(s: Pick<ScriptShown, "editable" | "error" | "saved">): {
  text: string;
  ok: boolean;
} {
  if (s.saved)
    return s.error
      ? { text: `Saved, but it has a problem: ${s.error}`, ok: false }
      : { text: "Saved!", ok: true };
  if (s.error) return { text: `Last time it ran: ${s.error}`, ok: false };
  if (!s.editable)
    return { text: "You can read this script, but only builders here can change it.", ok: true };
  return { text: "", ok: true };
}
