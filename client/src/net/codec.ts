import { create, fromBinary, toBinary, type MessageInitShape } from "@bufbuild/protobuf";
import {
  type ArtKind,
  ClientMessageSchema,
  type Emote,
  type GameMode,
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

export function encodeMove(
  p: { x: number; y: number; z: number; yaw: number; pitch: number; flying?: boolean },
  teleport: number,
): Bytes {
  const { yaw, pitch, flying = false } = p;
  return encode({
    case: "move",
    value: { pos: { x: p.x, y: p.y, z: p.z }, yaw, pitch, teleport, flying },
  });
}

/** `slot` is the inventory slot a survival placement comes from. */
export function encodeEditBlock(
  e: { x: number; y: number; z: number; block: number },
  slot = 0,
): Bytes {
  return encode({
    case: "editBlock",
    value: { pos: { x: e.x, y: e.y, z: e.z }, block: e.block, slot },
  });
}

export function encodeUseBlock(p: { x: number; y: number; z: number }): Bytes {
  return encode({ case: "useBlock", value: { pos: { x: p.x, y: p.y, z: p.z } } });
}

export function encodeOpenScript(p: { x: number; y: number; z: number }): Bytes {
  return encode({ case: "openScript", value: { pos: { x: p.x, y: p.y, z: p.z } } });
}

export function encodeSaveScript(p: { x: number; y: number; z: number }, source: string): Bytes {
  return encode({ case: "saveScript", value: { pos: { x: p.x, y: p.y, z: p.z }, source } });
}

export function encodeMoveItem(from: number, to: number): Bytes {
  return encode({ case: "moveItem", value: { from, to } });
}

export function encodeCraft(recipe: number): Bytes {
  return encode({ case: "craft", value: { recipe } });
}

export function encodeHitMob(mob: number): Bytes {
  return encode({ case: "hitMob", value: { mob } });
}

export function encodeHitPlayer(entity: number): Bytes {
  return encode({ case: "hitPlayer", value: { entity } });
}

export function encodeSendChat(text: string): Bytes {
  return encode({ case: "sendChat", value: { text } });
}

export function encodeEmote(emote: Emote): Bytes {
  return encode({ case: "sendEmote", value: { emote } });
}

export function encodeSubmitArt(kind: ArtKind, name: string, pixels: Uint8Array): Bytes {
  return encode({ case: "submitArt", value: { kind, name, pixels } });
}

export function encodeDecideArt(id: bigint, approve: boolean): Bytes {
  return encode({ case: "decideArt", value: { id, approve } });
}

export function encodeSetAdmin(playerId: bigint, admin: boolean): Bytes {
  return encode({ case: "setAdmin", value: { playerId, admin } });
}

export function encodeSetGameMode(playerId: bigint, mode: GameMode): Bytes {
  return encode({ case: "setGameMode", value: { playerId, mode } });
}

export function decodeServer(bytes: Uint8Array): ServerMessage {
  return fromBinary(ServerMessageSchema, bytes);
}
