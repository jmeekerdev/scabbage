import { create, fromBinary, toBinary, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ClientMessageSchema,
  ProtocolVersion,
  ServerMessageSchema,
  type ServerMessage,
} from "../pb/scabbage/v1/net_pb";

export const PROTOCOL_VERSION: number = ProtocolVersion.CURRENT;

type ClientBody = MessageInitShape<typeof ClientMessageSchema>["body"];

// toBinary always allocates a plain ArrayBuffer, which WebSocket.send requires.
function encode(body: ClientBody): Uint8Array<ArrayBuffer> {
  return toBinary(
    ClientMessageSchema,
    create(ClientMessageSchema, { body }),
  ) as Uint8Array<ArrayBuffer>;
}

export function encodeHello(clientName: string): Uint8Array<ArrayBuffer> {
  return encode({ case: "hello", value: { protocolVersion: PROTOCOL_VERSION, clientName } });
}

export function encodePing(clientTimeMs: number): Uint8Array<ArrayBuffer> {
  return encode({ case: "ping", value: { clientTimeMs } });
}

export function decodeServer(bytes: Uint8Array): ServerMessage {
  return fromBinary(ServerMessageSchema, bytes);
}
