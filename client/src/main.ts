import "./style.css";
import { browserStore, loadSecret, resetSecret } from "./join/identity";
import { parseInvite } from "./join/invite";
import { screenForRejection, type Action, type Screen } from "./join/outcome";
import {
  encodeEditBlock,
  encodeEmote,
  encodeMove,
  encodeRequestChunks,
  encodeSendChat,
  type JoinInit,
} from "./net/codec";
import { Connection, type ConnectionEvents } from "./net/connection";
import { socketUrl } from "./net/url";
import type {
  BlockChanged,
  Emote,
  PlayerPose,
  ServerMessage,
  Teleport,
  Welcome,
  WorldInfo,
} from "./pb/scabbage/v1/net_pb";
import {
  aim,
  breakEdit,
  buildWorldFor,
  placeEdit,
  type BuildWorld,
  type Edit,
} from "./player/building";
import { Hotbar, hotbarBlocks } from "./player/hotbar";
import { GamepadControls } from "./player/gamepad";
import { combineIntents, type BuildActions } from "./player/input";
import { KeyboardMouse } from "./player/keyboard-mouse";
import { MoveSync, type Pose } from "./player/move-sync";
import type { SolidAt } from "./player/physics";
import { Player } from "./player/player";
import { RemotePlayers } from "./player/remote-players";
import { Avatars } from "./render/avatars";
import { blockIconPainter } from "./render/icons";
import { startScene } from "./render/scene";
import { Selection } from "./render/selection";
import { createWorldRenderer } from "./render/world-renderer";
import { defaultSettings, loadSettings, saveSettings } from "./settings/settings";
import { emojiFor } from "./social/emotes";
import { adminRouter, InvitePanel, PendingList } from "./ui/admin";
import { ChatView } from "./ui/chat-view";
import { $, el } from "./ui/dom";
import { EmoteWheel, popEmoji } from "./ui/emote-wheel";
import { mountHotbar } from "./ui/hotbar-view";
import { settingsPanel } from "./ui/settings-panel";
import { TouchControls } from "./ui/touch-controls";
import { solidQuery } from "./world/solid";
import { hideOverlay, showJoinForm, showMessage } from "./ui/screens";

const overlay = $("#overlay");
const statusEl = $("#status");
const pingEl = $("#ping");
const hintEl = $("#hint");
const crosshairEl = $("#crosshair");
const store = browserStore();
const clientName = navigator.userAgent.match(/\(([^;)]+)/)?.[1] ?? "unknown";
const inviteCode = takeInviteFromUrl();
let conn: Connection | undefined;
let lastJoin: JoinInit | undefined;
let onAdminMessage: ConnectionEvents["onMessage"] | undefined;

const touch = matchMedia("(pointer: coarse)").matches;
let settings = loadSettings(store, defaultSettings(touch));
const gameScene = startScene($("#game"), settings);
const world = createWorldRenderer(gameScene, (chunks) => conn?.send(encodeRequestChunks(chunks)));
const player = new Player();
const selection = new Selection(gameScene.scene);
const hotbar = new Hotbar([]);
const icons = blockIconPainter();
let locked = false;
const chat = new ChatView(
  $("#chat"),
  (text) => conn?.send(encodeSendChat(text)),
  (sent) =>
    sent &&
    !touch &&
    $("#game")
      .requestPointerLock()
      .catch(() => undefined),
);
const emotes = new EmoteWheel($("#emotes"), sendEmote);
const keyboardMouse = new KeyboardMouse($("#game"), player, hotbar, emotes, (l) => {
  locked = l;
  hintEl.hidden = l;
});
const gamepad = new GamepadControls(player, hotbar, () => navigator.getGamepads(), sendEmote);
const touchControls = touch ? new TouchControls($("#game"), $("#touch"), player) : undefined;
const avatars = new Avatars(gameScene.scene);
let moves: MoveSync | undefined; // set while in the world
let others: RemotePlayers | undefined; // set while in the world
let solid: SolidAt = () => true;
let building: BuildWorld | undefined;
let lastFrameMs: number | undefined;
gameScene.onFrame((timeMs) => {
  const dt = lastFrameMs === undefined ? 0 : (timeMs - lastFrameMs) / 1000;
  lastFrameMs = timeMs;
  const intent = combineIntents(
    [keyboardMouse, gamepad, touchControls].flatMap((c) => c?.poll(timeMs) ?? []),
  );
  player.update(dt, intent.move, solid);
  const eye = player.eye();
  gameScene.camera.position.set(eye.x, eye.y, eye.z);
  gameScene.camera.rotation.set(player.look.pitch, player.look.yaw, 0, "YXZ");
  world.tick(eye.x, eye.z, settings.viewDistance);
  build(intent.build, [eye.x, eye.y, eye.z]);
  crosshairEl.hidden = !(moves && (locked || touch || gamepad.active));
  if (gamepad.active) hintEl.hidden = true;
  avatars.update(others?.sample(performance.now()) ?? [], performance.now());
  const pose = moves?.due(timeMs, player.pose());
  if (pose && moves) conn?.send(encodeMove(pose, moves.teleport));
});
addHudDialog(
  "Settings",
  settingsPanel(settings, (s) => {
    settings = s;
    saveSettings(store, s);
    gameScene.apply(s);
  }),
);
addHudButton("💬", "Chat", () => chat.open());
addHudButton("😀", "Emotes", () => emotes.open());
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
        if (status !== "connected") {
          pingEl.textContent = "";
          others = undefined;
        }
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
        else if (msg.body.case === "teleport") teleport(msg.body.value);
        else if (msg.body.case === "blockChanged") blockChanged(msg.body.value);
        else if (msg.body.case === "chat") chat.add(msg.body.value);
        else if (msg.body.case === "emoteShown")
          emoteShown(msg.body.value.entity, msg.body.value.emote);
        else if (!rosterChanged(msg)) onAdminMessage?.(msg);
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
  if (w.world) enterWorld(w.world, w.tickRateHz);
}

function enterWorld(info: WorldInfo, tickRateHz: number): void {
  world.start(info);
  solid = solidQuery(
    info,
    info.blocks.map((b) => b.solid),
    (x, y, z) => world.blockAt(x, y, z),
  );
  building = buildWorldFor(info, (x, y, z) => world.blockAt(x, y, z));
  const name = (id: number) => info.blocks.find((b) => b.id === id)?.name ?? "missing";
  hotbar.slots.splice(0, Infinity, ...hotbarBlocks(info.blocks));
  hotbar.select(0);
  mountHotbar($("#hotbar"), hotbar, name, icons);
  const spawn = info.spawn;
  player.place(spawn?.x ?? 0, spawn?.y ?? 0, spawn?.z ?? 0);
  moves = new MoveSync(1000 / Math.max(1, tickRateHz));
  others = new RemotePlayers(1000 / Math.max(1, tickRateHz));
  hintEl.hidden = document.pointerLockElement !== null || touch;
  $("#touch").hidden = !touch;
  chat.enabled = true;
}

/** The server put us back somewhere, usually after rejecting a move. */
function teleport(t: Teleport): void {
  player.place(t.pos?.x ?? 0, t.pos?.y ?? 0, t.pos?.z ?? 0);
  moves?.teleported(t.id);
}

/** Outlines the block in reach and carries out this frame's building actions. */
function build(act: BuildActions, eye: [number, number, number]): void {
  const hit = building && aim(eye, player.look.yaw, player.look.pitch, building);
  selection.show(hit);
  if (!hit || !building) return;
  if (act.picking) hotbar.pick(building.blockAt(hit.x, hit.y, hit.z) ?? 0);
  const edit = act.breaking
    ? breakEdit(hit, building)
    : act.placing
      ? placeEdit(hit, hotbar.current(), player.body, building)
      : undefined;
  if (edit) sendEdit(edit);
}

/** Shows the edit straight away; the server confirms it or sends back what is really there. */
function sendEdit(edit: Edit): void {
  if (!conn || !moves) return;
  const pose = moves.flush(player.pose()); // the server checks reach from our latest position
  if (pose) conn.send(encodeMove(pose, moves.teleport));
  conn.send(encodeEditBlock(edit));
  world.setBlock(edit.x, edit.y, edit.z, edit.block);
}

function sendEmote(e: Emote): void {
  if (!moves) return;
  conn?.send(encodeEmote(e));
  popEmoji(document.body, emojiFor(e) ?? "");
}

function emoteShown(entity: number, e: Emote): void {
  const emoji = emojiFor(e);
  if (emoji) avatars.emote(entity, emoji, performance.now());
}

function blockChanged(c: BlockChanged): void {
  if (c.pos) world.setBlock(c.pos.x, c.pos.y, c.pos.z, c.block);
}

/** Applies other players arriving, moving or leaving; false if `msg` is about something else. */
function rosterChanged(msg: ServerMessage): boolean {
  const now = performance.now();
  const b = msg.body;
  if (b.case === "playersJoined")
    for (const p of b.value.players) others?.join(p, poseOf(p.pose), now);
  else if (b.case === "playersMoved")
    for (const p of b.value.poses) others?.move(p.entity, poseOf(p), now);
  else if (b.case === "playerLeft") others?.leave(b.value.entity);
  else return false;
  return true;
}

function poseOf(p: PlayerPose | undefined): Pose {
  return {
    x: p?.pos?.x ?? 0,
    y: p?.pos?.y ?? 0,
    z: p?.pos?.z ?? 0,
    yaw: p?.yaw ?? 0,
    pitch: p?.pitch ?? 0,
  };
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

function addHudButton(icon: string, label: string, onclick: () => void): void {
  $("#hud-buttons").append(el("button", { "aria-label": label, title: label, onclick }, icon));
}
