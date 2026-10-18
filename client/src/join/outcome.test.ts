import { describe, expect, it } from "vitest";
import { DisconnectCode } from "../pb/scabbage/v1/net_pb";
import { screenForRejection } from "./outcome";

const rejected = (code: DisconnectCode, reason = "why") => screenForRejection({ code, reason });

describe("screenForRejection", () => {
  it("asks for a code when the device is unknown", () => {
    expect(rejected(DisconnectCode.NEED_JOIN_CODE)).toEqual({ kind: "join" });
  });

  it("shows the join form with an error for bad codes and names", () => {
    expect(rejected(DisconnectCode.BAD_JOIN_CODE)).toMatchObject({ kind: "join", error: /code/ });
    expect(rejected(DisconnectCode.BAD_NAME)).toMatchObject({ kind: "join", error: /16/ });
  });

  it("offers a fresh start after a denial", () => {
    expect(rejected(DisconnectCode.DENIED)).toMatchObject({ kind: "message", action: "restart" });
  });

  it("suggests a reload on protocol mismatch, showing the server's reason", () => {
    expect(rejected(DisconnectCode.PROTOCOL, "version 1 unsupported")).toMatchObject({
      kind: "message",
      text: "version 1 unsupported",
      action: "reload",
    });
  });

  it("lets the player retry after rate limiting", () => {
    expect(rejected(DisconnectCode.RATE_LIMITED)).toMatchObject({ action: "retry" });
  });
});
