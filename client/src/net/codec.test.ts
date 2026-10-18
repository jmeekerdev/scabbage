import { create, toBinary } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ServerMessageSchema } from "../pb/scabbage/v1/net_pb";
import { PROTOCOL_VERSION, decodeServer, encodeHello, encodePing } from "./codec";

describe("codec", () => {
  it("encodes a hello with the current protocol version", () => {
    expect(encodeHello("phone").length).toBeGreaterThan(0);
    expect(PROTOCOL_VERSION).toBe(1);
  });

  it("encodes a ping", () => {
    expect(encodePing(12.5).length).toBeGreaterThan(0);
  });

  it("decodes a server pong", () => {
    const bytes = toBinary(
      ServerMessageSchema,
      create(ServerMessageSchema, {
        body: { case: "pong", value: { clientTimeMs: 7, serverTimeMs: 9 } },
      }),
    );
    const msg = decodeServer(bytes);
    expect(msg.body.case).toBe("pong");
    if (msg.body.case === "pong") expect(msg.body.value.clientTimeMs).toBe(7);
  });
});
