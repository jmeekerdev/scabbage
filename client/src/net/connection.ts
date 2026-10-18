import type { Welcome } from "../pb/scabbage/v1/net_pb";
import { decodeServer, encodeHello, encodePing } from "./codec";
import { LatencyTracker } from "./latency";

export type Status = "connecting" | "connected" | "disconnected";

export interface ConnectionEvents {
  onStatus(status: Status, detail?: string): void;
  onWelcome(welcome: Welcome): void;
  onRtt(rttMs: number): void;
}

const PING_INTERVAL_MS = 2000;
const RECONNECT_MS = 3000;

/** Maintains the game socket: handshake, periodic pings and auto-reconnect. */
export class Connection {
  private socket?: WebSocket;
  private pingTimer?: number;
  private rejected = false;
  private readonly latency = new LatencyTracker();

  constructor(
    private readonly url: string,
    private readonly clientName: string,
    private readonly events: ConnectionEvents,
  ) {}

  start(): void {
    this.events.onStatus("connecting");
    const ws = new WebSocket(this.url);
    ws.binaryType = "arraybuffer";
    ws.onopen = () => ws.send(encodeHello(this.clientName));
    ws.onmessage = (ev) => this.receive(new Uint8Array(ev.data as ArrayBuffer));
    ws.onclose = () => this.closed();
    this.socket = ws;
  }

  private receive(bytes: Uint8Array): void {
    const msg = decodeServer(bytes);
    switch (msg.body.case) {
      case "welcome":
        this.events.onStatus("connected");
        this.events.onWelcome(msg.body.value);
        this.startPinging();
        break;
      case "pong":
        this.latency.record(msg.body.value.clientTimeMs, performance.now());
        if (this.latency.rttMs !== undefined) this.events.onRtt(this.latency.rttMs);
        break;
      case "disconnect":
        this.rejected = true;
        this.events.onStatus("disconnected", msg.body.value.reason);
        break;
    }
  }

  private startPinging(): void {
    const ping = () => {
      // Skip pings while the tab is hidden so idle devices stay quiet.
      if (document.visibilityState === "visible") this.socket?.send(encodePing(performance.now()));
    };
    ping();
    this.pingTimer = window.setInterval(ping, PING_INTERVAL_MS);
  }

  private closed(): void {
    window.clearInterval(this.pingTimer);
    if (this.rejected) return;
    this.events.onStatus("disconnected", "reconnecting…");
    window.setTimeout(() => this.start(), RECONNECT_MS);
  }
}
