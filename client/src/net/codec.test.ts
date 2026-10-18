import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ClientMessageSchema, ServerMessageSchema } from "../pb/scabbage/v1/net_pb";
import {
  PROTOCOL_VERSION,
  decodeServer,
  encodeCreateJoinCode,
  encodeDecideJoin,
  encodeHello,
  encodePing,
} from "./codec";

const decodeClient = (b: Uint8Array) => fromBinary(ClientMessageSchema, b).body;

describe("codec", () => {
  it("encodes a hello with version, secret and optional join request", () => {
    const secret = new Uint8Array(32).fill(3);
    const body = decodeClient(
      encodeHello({ clientName: "phone", secret, join: { code: "ABCD", name: "Ann", color: 5 } }),
    );
    expect(body.case).toBe("hello");
    if (body.case !== "hello") return;
    expect(body.value.protocolVersion).toBe(PROTOCOL_VERSION);
    expect(body.value.deviceSecret).toEqual(secret);
    expect(body.value.join?.name).toBe("Ann");
    expect(body.value.console).toBe(false);
  });

  it("encodes a console hello", () => {
    const body = decodeClient(
      encodeHello({ clientName: "pc", secret: new Uint8Array(32), console: true }),
    );
    expect(body.case === "hello" && body.value.console && !body.value.join).toBe(true);
  });

  it("encodes a ping", () => {
    const body = decodeClient(encodePing(12.5));
    expect(body.case === "ping" && body.value.clientTimeMs).toBe(12.5);
  });

  it("encodes admin messages", () => {
    const decide = decodeClient(encodeDecideJoin(4n, true));
    expect(decide.case === "decideJoin" && decide.value.requestId).toBe(4n);
    const code = decodeClient(encodeCreateJoinCode(3, 24));
    expect(code.case === "createJoinCode" && code.value.uses).toBe(3);
  });

  it("decodes a server pong", () => {
    const bytes = toBinary(
      ServerMessageSchema,
      create(ServerMessageSchema, {
        body: { case: "pong", value: { clientTimeMs: 7, serverTimeMs: 9 } },
      }),
    );
    const msg = decodeServer(bytes);
    expect(msg.body.case === "pong" && msg.body.value.clientTimeMs).toBe(7);
  });
});
