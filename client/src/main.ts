import "./style.css";
import { browserStore, loadSecret, resetSecret } from "./join/identity";
import { parseInvite } from "./join/invite";
import { screenForRejection, type Action, type Screen } from "./join/outcome";
import { encodeRequestChunks, type JoinInit } from "./net/codec";
import { Connection, type ConnectionEvents } from "./net/connection";
import { socketUrl } from "./net/url";
import type { Vec3, Welcome } from "./pb/scabbage/v1/net_pb";
import { startScene } from "./render/scene";
import { createWorldRenderer } from "./render/world-renderer";
import { defaultSettings, loadSettings, saveSettings } from "./settings/settings";
import { adminRouter, InvitePanel, PendingList } from "./ui/admin";
import { $, el } from "./ui/dom";
import { settingsPanel } from "./ui/settings-panel";
import { hideOverlay, showJoinForm, showMessage } from "./ui/screens";

const overlay = $("#overlay");
const statusEl = $("#status");
const pingEl = $("#ping");
const store = browserStore();
const clientName = navigator.userAgent.match(/\(([^;)]+)/)?.[1] ?? "unknown";
const inviteCode = takeInviteFromUrl();
let conn: Connection | undefined;
let lastJoin: JoinInit | undefined;
let onAdminMessage: ConnectionEvents["onMessage"] | undefined;

let settings = loadSettings(store, defaultSettings(matchMedia("(pointer: coarse)").matches));
const gameScene = startScene($("#game"), settings);
const world = createWorldRenderer(gameScene, (chunks) => conn?.send(encodeRequestChunks(chunks)));
gameScene.onFrame((timeMs) => {
  lookAround(timeMs);
  const { x, z } = gameScene.camera.position;
  world.tick(x, z, settings.viewDistance);
});
addHudDialog(
  "Settings",
  settingsPanel(settings, (s) => {
    settings = s;
    saveSettings(store, s);
    gameScene.apply(s);
  }),
);
connect();

/** Reads `#join=CODE` and removes it from the address bar so it isn't re-shared by accident. */
function takeInviteFromUrl(): string | undefined {
  const code = parseInvite(location.hash);
  if (code) history.replaceState(null, "", location.pathname + location.search);
  return code;
}

function connect(join?: JoinInit): void {
  conn?.stop();
  lastJoin = join ?? lastJoin;
  conn = new Connection(
    socketUrl(location),
    { clientName, secret: loadSecret(store), join: lastJoin },
    {
      onStatus(status, detail) {
        statusEl.textContent = detail ? `${status}: ${detail}` : status;
        statusEl.dataset.status = status;
        if (status !== "connected") pingEl.textContent = "";
      },
      onRtt: (ms) => (pingEl.textContent = `ping ${Math.round(ms)} ms`),
      onPending: () =>
        showMessage(overlay, {
          title: "Almost in!",
          text: "An admin needs to let you in. Keep this page open.",
        }),
      onWelcome: enterGame,
      onRejected: (d) => show(screenForRejection(d)),
      onMessage: (msg) => {
        if (msg.body.case === "chunkBatch") world.receive(msg.body.value);
        else onAdminMessage?.(msg);
      },
    },
  );
  conn.start();
}

function show(screen: Screen): void {
  if (screen.kind === "join") {
    showJoinForm(overlay, {
      code: lastJoin?.code ?? inviteCode,
      name: lastJoin?.name,
      color: lastJoin?.color,
      error: screen.error,
      onSubmit: connect,
    });
    return;
  }
  const labels: Record<Action, string> = {
    retry: "Try again",
    reload: "Reload",
    restart: "Start over",
  };
  showMessage(overlay, {
    ...screen,
    actionLabel: screen.action && labels[screen.action],
    onAction: () => act(screen.action),
  });
}

function act(action?: Action): void {
  if (action === "reload") location.reload();
  if (action === "retry") connect();
  if (action === "restart") {
    resetSecret(store);
    show({ kind: "join" });
  }
}

function enterGame(w: Welcome): void {
  hideOverlay(overlay);
  statusEl.textContent = `${w.you?.name ?? "?"} on ${w.serverName}`;
  if (w.you?.admin && !onAdminMessage) mountAdminTools();
  if (w.world) {
    world.start(w.world);
    standAt(w.world.spawn);
  }
}

const EYE_HEIGHT = 1.62;

function standAt(spawn?: Vec3): void {
  gameScene.camera.position.set(spawn?.x ?? 0, (spawn?.y ?? 64) + EYE_HEIGHT, spawn?.z ?? 0);
}

/** Slowly turns the view until player movement exists. */
function lookAround(timeMs: number): void {
  gameScene.camera.rotation.set(-0.15, timeMs / 30000, 0, "YXZ");
}

/** In-game admin tools: join-request toasts and an invite dialog. */
function mountAdminTools(): void {
  const send = (b: Parameters<Connection["send"]>[0]) => conn?.send(b);
  const pending = new PendingList($("#toasts"), send);
  const invite = new InvitePanel(send, location.origin);
  addHudDialog("Invite", invite.node);
  onAdminMessage = adminRouter(pending, invite);
}

/** Adds a HUD button that opens `content` in a modal dialog. */
function addHudDialog(label: string, content: Node): void {
  const dialog = el(
    "dialog",
    { class: "card" },
    content,
    el("button", { onclick: () => dialog.close() }, "Close"),
  );
  document.body.append(dialog);
  $("#hud-buttons").append(el("button", { onclick: () => dialog.showModal() }, label));
}
