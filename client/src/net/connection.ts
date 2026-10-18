import type { Disconnect, ServerMessage, Welcome } from "../pb/scabbage/v1/net_pb";
import { decodeServer, encodeHello, encodePing, type Bytes, type HelloOptions } from "./codec";
import { LatencyTracker } from "./latency";

export type Status = "connecting" | "waiting" | "connected" | "disconnected";

export interface ConnectionEvents {
  onStatus(status: Status, detail?: string): void;
  onWelcome(welcome: Welcome): void;
  onPending(): void;
  /** The server refused us; the connection will not retry. */
  onRejected(reason: Disconnect): void;
  onRtt(rttMs: number): void;
  /** Any other server message, e.g. admin updates. */
  onMessage(msg: ServerMessage): void;
}

const PING_INTERVAL_MS = 2000;
const RECONNECT_MS = 3000;

/** Maintains the game socket: handshake, periodic pings and auto-reconnect. */
export class Connection {
  private socket?: WebSocket;
  private pingTimer?: number;
  private stopped = false;
  private readonly latency = new LatencyTracker();

  constructor(
    private readonly url: string,
    private readonly hello: HelloOptions,
    private readonly events: Partial<ConnectionEvents>,
  ) {}

  start(): void {
    this.events.onStatus?.("connecting");
    const ws = new WebSocket(this.url);
    ws.binaryType = "arraybuffer";
    ws.onopen = () => ws.send(encodeHello(this.hello));
    ws.onmessage = (ev) => this.receive(new Uint8Array(ev.data as ArrayBuffer));
    ws.onclose = () => this.closed();
    this.socket = ws;
  }

  /** Closes the socket for good. */
  stop(): void {
    this.stopped = true;
    this.socket?.close();
  }

  send(bytes: Bytes): void {
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(bytes);
  }

  private receive(bytes: Uint8Array): void {
    const msg = decodeServer(bytes);
    switch (msg.body.case) {
      case "welcome":
        this.events.onStatus?.("connected");
        this.events.onWelcome?.(msg.body.value);
        this.startPinging();
        break;
      case "joinPending":
        this.events.onStatus?.("waiting");
        this.events.onPending?.();
        this.startPinging();
        break;
      case "pong":
        this.latency.record(msg.body.value.clientTimeMs, performance.now());
        if (this.latency.rttMs !== undefined) this.events.onRtt?.(this.latency.rttMs);
        break;
      case "disconnect":
        this.stopped = true;
        this.events.onStatus?.("disconnected", msg.body.value.reason);
        this.events.onRejected?.(msg.body.value);
        break;
      default:
        this.events.onMessage?.(msg);
    }
  }

  private startPinging(): void {
    window.clearInterval(this.pingTimer);
    const ping = () => {
      // Skip pings while the tab is hidden so idle devices stay quiet.
      if (document.visibilityState === "visible") this.send(encodePing(performance.now()));
    };
    ping();
    this.pingTimer = window.setInterval(ping, PING_INTERVAL_MS);
  }

  private closed(): void {
    window.clearInterval(this.pingTimer);
    if (this.stopped) return;
    this.events.onStatus?.("disconnected", "reconnecting…");
    window.setTimeout(() => this.start(), RECONNECT_MS);
  }
}
