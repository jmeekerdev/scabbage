import { DisconnectCode } from "../pb/scabbage/v1/net_pb";

export type Action = "retry" | "reload" | "restart";

export type Screen =
  | { kind: "join"; error?: string }
  | { kind: "message"; title: string; text: string; action?: Action };

/** Decides what to show after the server refuses this device. */
export function screenForRejection(d: { code: DisconnectCode; reason: string }): Screen {
  switch (d.code) {
    case DisconnectCode.NEED_JOIN_CODE:
      return { kind: "join" };
    case DisconnectCode.BAD_JOIN_CODE:
      return { kind: "join", error: "That code didn't work. It may be used up or expired." };
    case DisconnectCode.BAD_NAME:
      return { kind: "join", error: "Use up to 16 letters, numbers, spaces, _ or -." };
    case DisconnectCode.DENIED:
      return {
        kind: "message",
        title: "Not this time",
        text: "An admin didn't let this device in.",
        action: "restart",
      };
    case DisconnectCode.RATE_LIMITED:
      return {
        kind: "message",
        title: "Too many tries",
        text: "Wait a few minutes, then try again.",
        action: "retry",
      };
    case DisconnectCode.NOT_ADMIN:
      return { kind: "message", title: "Admins only", text: d.reason };
    case DisconnectCode.PROTOCOL:
      return { kind: "message", title: "Update needed", text: d.reason, action: "reload" };
    default:
      return { kind: "message", title: "Disconnected", text: d.reason, action: "retry" };
  }
}
