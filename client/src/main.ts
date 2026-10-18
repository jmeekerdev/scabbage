import "./style.css";
import { Connection } from "./net/connection";
import { socketUrl } from "./net/url";
import { startScene } from "./render/scene";

const canvas = document.querySelector<HTMLCanvasElement>("#game")!;
const statusEl = document.querySelector<HTMLElement>("#status")!;
const pingEl = document.querySelector<HTMLElement>("#ping")!;

startScene(canvas);

const deviceName = navigator.userAgent.match(/\(([^;)]+)/)?.[1] ?? "unknown";
new Connection(socketUrl(location), deviceName, {
  onStatus(status, detail) {
    statusEl.textContent = detail ? `${status}: ${detail}` : status;
    statusEl.dataset.status = status;
    if (status !== "connected") pingEl.textContent = "";
  },
  onWelcome(welcome) {
    statusEl.textContent = `connected to ${welcome.serverName}`;
  },
  onRtt(rttMs) {
    pingEl.textContent = `ping ${Math.round(rttMs)} ms`;
  },
}).start();
