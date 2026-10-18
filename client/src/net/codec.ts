import { create, fromBinary, toBinary, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ClientMessageSchema,
  ProtocolVersion,
  ServerMessageSchema,
  type JoinRequestSchema,
  type ServerMessage,
} from "../pb/scabbage/v1/net_pb";

export const PROTOCOL_VERSION: number = ProtocolVersion.CURRENT;

type ClientBody = MessageInitShape<typeof ClientMessageSchema>["body"];
export type JoinInit = MessageInitShape<typeof JoinRequestSchema>;
export type Bytes = Uint8Array<ArrayBuffer>;

export interface HelloOptions {
  clientName: string;
  secret: Uint8Array;
  join?: JoinInit;
  console?: boolean;
}

// toBinary always allocates a plain ArrayBuffer, which WebSocket.send requires.
function encode(body: ClientBody): Bytes {
  return toBinary(ClientMessageSchema, create(ClientMessageSchema, { body })) as Bytes;
}

export function encodeHello(o: HelloOptions): Bytes {
  return encode({
    case: "hello",
    value: {
      protocolVersion: PROTOCOL_VERSION,
      clientName: o.clientName,
      deviceSecret: o.secret,
      join: o.join,
      console: o.console ?? false,
    },
  });
}

export function encodePing(clientTimeMs: number): Bytes {
  return encode({ case: "ping", value: { clientTimeMs } });
}

export function encodeDecideJoin(requestId: bigint, approve: boolean): Bytes {
  return encode({ case: "decideJoin", value: { requestId, approve } });
}

export function encodeCreateJoinCode(uses: number, ttlHours: number): Bytes {
  return encode({ case: "createJoinCode", value: { uses, ttlHours } });
}

export function encodeRequestChunks(chunks: readonly { x: number; y: number; z: number }[]): Bytes {
  return encode({
    case: "requestChunks",
    value: { chunks: chunks.map(({ x, y, z }) => ({ x, y, z })) },
  });
}

export function decodeServer(bytes: Uint8Array): ServerMessage {
  return fromBinary(ServerMessageSchema, bytes);
}
