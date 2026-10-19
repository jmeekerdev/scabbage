import "./style.css";
import { distanceGain, Footsteps } from "./audio/cues";
import { GameAudio } from "./audio/game-audio";
import { browserStore, loadSecret, resetSecret } from "./join/identity";
import { parseInvite } from "./join/invite";
import { screenForRejection, type Action, type Screen } from "./join/outcome";
import {
  encodeEditBlock,
  encodeEmote,
  encodeMove,
  encodeRequestChunks,
  encodeSendChat,
  encodeSubmitArt,
  type JoinInit,
} from "./net/codec";
import { Connection, type ConnectionEvents } from "./net/connection";
import { socketUrl } from "./net/url";
import {
  ArtKind,
  type BlockChanged,
  type BlockType,
  type Emote,
  type PlayerInfo,
  type PlayerPose,
  type ServerMessage,
  type Teleport,
  type Texture,
  type Welcome,
  type WorldInfo,
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
import type { BoxesAt } from "./player/physics";
import { Player } from "./player/player";
import { RemotePlayers } from "./player/remote-players";
import { Avatars, skinPixels } from "./render/avatars";
import { blockIconPainter } from "./render/icons";
import { TEXTURE_NAMES } from "./render/textures";
import { startScene } from "./render/scene";
import { Selection } from "./render/selection";
import { createWorldRenderer } from "./render/world-renderer";
import { AutoQuality } from "./settings/auto-quality";
import { deviceTier, readDevice } from "./settings/device";
import { defaultSettings, loadSettings, saveSettings, type Settings } from "./settings/settings";
import { emojiFor } from "./social/emotes";
import { adminRouter, ArtList, InvitePanel, PendingList } from "./ui/admin";
import { BlockPicker } from "./ui/block-picker";
import { ChatView } from "./ui/chat-view";
import { $, el } from "./ui/dom";
import { EmoteWheel, popEmoji } from "./ui/emote-wheel";
import { mountHotbar } from "./ui/hotbar-view";
import { openPixelEditor, type EditorMode, type Submission } from "./ui/pixel-editor";
import {
  canGoFullscreen,
  registerServiceWorker,
  toggleLandscape,
  unlockOnFullscreenExit,
} from "./ui/pwa";
import { settingsPanel } from "./ui/settings-panel";
import { TouchControls } from "./ui/touch-controls";
import { collisionBoxes, FULL_BLOCK, heldBlock, type Box } from "./world/shapes";
import { solidQuery } from "./world/solid";
import type { WorldBounds } from "./world/streamer";
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

const device = readDevice();
const touch = device.touch;
let settings = loadSettings(store, defaultSettings(deviceTier(device), touch));
const autoQuality = new AutoQuality();
const gameScene = startScene($("#game"), settings);
const world = createWorldRenderer(gameScene, (chunks) => conn?.send(encodeRequestChunks(chunks)));
const player = new Player();
const selection = new Selection(gameScene.scene);
const hotbar = new Hotbar([]);
const picker = new BlockPicker((block) => {
  hotbar.pick(block);
  picker.node.closest("dialog")?.close();
});
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
const audio = new GameAudio(settings.sound, settings.music);
const footsteps = new Footsteps();
let moves: MoveSync | undefined; // set while in the world
let others: RemotePlayers | undefined; // set while in the world
let solid: BoxesAt = () => FULL_BLOCK;
let building: BuildWorld | undefined;
let bounds: WorldBounds | undefined; // set while in the world
let me: PlayerInfo | undefined;
let read: (texture: string) => Uint8Array | undefined = () => undefined; // current block textures
let lastFrameMs: number | undefined;
gameScene.onFrame((timeMs) => {
  const dt = lastFrameMs === undefined ? 0 : (timeMs - lastFrameMs) / 1000;
  lastFrameMs = timeMs;
  const intent = combineIntents(
    [keyboardMouse, gamepad, touchControls].flatMap((c) => c?.poll(timeMs) ?? []),
  );
  player.update(dt, intent.move, solid);
  if (moves && footsteps.update(player.body)) audio.play("step");
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
  const lighter = moves && autoQuality.frame(timeMs, settings);
  if (lighter) {
    changeSettings(lighter);
    settingsView.show(lighter);
  }
});
const settingsView = settingsPanel(settings, changeSettings);
addHudDialog("Settings", settingsView.node);
addHudDialog("Blocks", picker.node);
addHudButton("🎨", "Create", () => createDialog.showModal());
addHudButton("💬", "Chat", () => chat.open());
addHudButton("😀", "Emotes", () => emotes.open());
if (touch && canGoFullscreen()) {
  addHudButton("⛶", "Fullscreen", () => void toggleLandscape().catch(() => undefined));
  unlockOnFullscreenExit();
}
const createDialog = el(
  "dialog",
  { class: "card create" },
  el("h1", {}, "Create"),
  createButton("🧑 My look", "skin"),
  createButton("🧱 New block", "block"),
  el("button", { onclick: () => createDialog.close() }, "Close"),
);
document.body.append(createDialog);
registerServiceWorker();
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
        else if (msg.body.case === "chat") {
          chat.add(msg.body.value);
          audio.play("pop", 0.6);
        } else if (msg.body.case === "emoteShown")
          emoteShown(msg.body.value.entity, msg.body.value.emote);
        else if (msg.body.case === "blocksChanged")
          applyBlocks(msg.body.value.blocks, msg.body.value.textures);
        else if (msg.body.case === "skinChanged")
          others?.setSkin(msg.body.value.entity, msg.body.value.skin);
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
  me = w.you;
  if (!w.you?.admin && onAdminMessage) return location.reload(); // admin rights were removed
  if (w.you?.admin && !onAdminMessage) mountAdminTools();
  if (w.world) enterWorld(w.world, w.tickRateHz);
}

function enterWorld(info: WorldInfo, tickRateHz: number): void {
  world.start(info);
  bounds = { radius: info.radius, height: info.height };
  hotbar.slots.splice(0, Infinity, ...hotbarBlocks(info.blocks));
  hotbar.select(0);
  applyBlocks(info.blocks, info.textures);
  const spawn = info.spawn;
  player.place(spawn?.x ?? 0, spawn?.y ?? 0, spawn?.z ?? 0);
  moves = new MoveSync(1000 / Math.max(1, tickRateHz));
  others = new RemotePlayers(1000 / Math.max(1, tickRateHz));
  hintEl.hidden = document.pointerLockElement !== null || touch;
  $("#touch").hidden = !touch;
  chat.enabled = true;
  document.body.classList.add("in-world");
}

/** Uses a new block list everywhere blocks matter: drawing, collisions, building and the hotbar. */
function applyBlocks(blocks: BlockType[], textures: Texture[]): void {
  if (!bounds) return;
  const { atlas, table } = world.setBlocks(blocks, textures);
  read = atlas.read;
  const blockAt = (x: number, y: number, z: number) => world.blockAt(x, y, z);
  const boxesById: (readonly Box[])[] = [];
  for (const b of blocks) boxesById[b.id] = collisionBoxes(b.shape, b.solid);
  solid = solidQuery(bounds, boxesById, blockAt);
  building = buildWorldFor({ ...bounds, blocks }, blockAt);
  const name = (id: number) => blocks.find((b) => b.id === id)?.name ?? "missing";
  mountHotbar($("#hotbar"), hotbar, name, blockIconPainter(atlas, table));
  picker.set(blocks, blockIconPainter(atlas, table)); // icons can only be in one place at a time
}

/** The pixel editor for one kind of art, starting from what is there now. */
function edit(kind: EditorMode["kind"]): void {
  createDialog.close();
  if (!me || !moves) return;
  const color = me.color;
  const mode: EditorMode =
    kind === "skin"
      ? { kind, color, pixels: skinPixels(me.skin, color) }
      : kind === "texture"
        ? { kind, names: TEXTURE_NAMES, read }
        : { kind };
  openPixelEditor(mode, me.admin, submitArt);
}

function submitArt(s: Submission): void {
  conn?.send(encodeSubmitArt(s.kind, s.name, s.pixels));
  if (me && s.kind === ArtKind.SKIN) me.skin = s.pixels; // keep editing from the latest look
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
  if (act.picking) {
    const block = building.blockAt(hit.x, hit.y, hit.z) ?? 0;
    hotbar.pick(heldBlock(block, building.shape[block] ?? 0));
  }
  const edit = act.breaking
    ? breakEdit(hit, building)
    : act.placing
      ? placeEdit(hit, hotbar.current(), player.body, player.look.yaw, building)
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
  audio.play(edit.block === 0 ? "break" : "place");
}

function sendEmote(e: Emote): void {
  if (!moves) return;
  conn?.send(encodeEmote(e));
  popEmoji(document.body, emojiFor(e) ?? "");
}

function emoteShown(entity: number, e: Emote): void {
  const emoji = emojiFor(e);
  if (emoji) avatars.emote(entity, emoji, performance.now());
  audio.play("pop", 0.4);
}

/** Someone's edit, or our own echoed back; only changes we haven't already shown make a sound. */
function blockChanged(c: BlockChanged): void {
  if (!c.pos) return;
  const { x, y, z } = c.pos;
  if (world.blockAt(x, y, z) === c.block) return;
  world.setBlock(x, y, z, c.block);
  const b = player.body;
  const gain = distanceGain(Math.hypot(x + 0.5 - b.x, y + 0.5 - b.y, z + 0.5 - b.z));
  audio.play(c.block === 0 ? "break" : "place", gain);
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

/** In-game admin tools: join-request and art toasts, an invite dialog and repainting. */
function mountAdminTools(): void {
  const send = (b: Parameters<Connection["send"]>[0]) => conn?.send(b);
  const pending = new PendingList(toastBox(), send);
  const art = new ArtList(toastBox(), send);
  const invite = new InvitePanel(send, location.origin);
  addHudDialog("Invite", invite.node);
  createDialog.insertBefore(createButton("🖌️ Repaint a block", "texture"), createDialog.lastChild);
  onAdminMessage = adminRouter(pending, art, invite);
}

function toastBox(): HTMLElement {
  const box = el("div", { class: "toast-group" });
  $("#toasts").append(box);
  return box;
}

function createButton(label: string, kind: EditorMode["kind"]): HTMLButtonElement {
  return el("button", { class: "primary", onclick: () => edit(kind) }, label);
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

function changeSettings(s: Settings): void {
  settings = s;
  saveSettings(store, s);
  gameScene.apply(s);
  audio.set(s.sound, s.music);
}
