import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ArtKind, ClientMessageSchema, Emote, ServerMessageSchema } from "../pb/scabbage/v1/net_pb";
import {
  PROTOCOL_VERSION,
  decodeServer,
  encodeCreateJoinCode,
  encodeDecideArt,
  encodeDecideJoin,
  encodeHello,
  encodeEditBlock,
  encodeEmote,
  encodeMove,
  encodePing,
  encodeRequestChunks,
  encodeSendChat,
  encodeSubmitArt,
} from "./codec";

const decodeClient = (b: Uint8Array) => fromBinary(ClientMessageSchema, b).body;

describe("codec", () => {
  it("encodes a chat line and an emote", () => {
    expect(decodeClient(encodeSendChat("hi"))).toMatchObject({
      case: "sendChat",
      value: { text: "hi" },
    });
    expect(decodeClient(encodeEmote(Emote.WAVE))).toMatchObject({
      case: "sendEmote",
      value: { emote: Emote.WAVE },
    });
  });

  it("encodes a block edit", () => {
    const body = decodeClient(encodeEditBlock({ x: -4, y: 20, z: 9, block: 8 }));
    expect(body.case).toBe("editBlock");
    if (body.case !== "editBlock") return;
    const { pos, block } = body.value;
    expect([pos?.x, pos?.y, pos?.z, block]).toEqual([-4, 20, 9, 8]);
  });

  it("encodes a move with pose and teleport id", () => {
    const body = decodeClient(encodeMove({ x: 1.5, y: 64, z: -2.25, yaw: 0.5, pitch: -0.25 }, 3));
    expect(body.case).toBe("move");
    if (body.case !== "move") return;
    const { pos, yaw, pitch, teleport } = body.value;
    expect([pos?.x, pos?.y, pos?.z, yaw, pitch, teleport]).toEqual([1.5, 64, -2.25, 0.5, -0.25, 3]);
  });

  it("encodes a chunk request, including negative coordinates", () => {
    const body = decodeClient(encodeRequestChunks([{ x: -3, y: 2, z: 7 }]));
    expect(body.case).toBe("requestChunks");
    if (body.case !== "requestChunks") return;
    expect(body.value.chunks.map(({ x, y, z }) => [x, y, z])).toEqual([[-3, 2, 7]]);
  });

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

  it("encodes art and its approval", () => {
    const art = decodeClient(encodeSubmitArt(ArtKind.BLOCK, "Flower", new Uint8Array([1, 2])));
    expect(art).toMatchObject({
      case: "submitArt",
      value: { kind: ArtKind.BLOCK, name: "Flower" },
    });
    expect(art.case === "submitArt" && [...art.value.pixels]).toEqual([1, 2]);
    const decide = decodeClient(encodeDecideArt(5n, false));
    expect(decide).toMatchObject({ case: "decideArt", value: { id: 5n, approve: false } });
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
