import { coordsText } from "./ui/coords";
import { BossBar, bossNear } from "./ui/boss-bar";
import "./style.css";
import { distanceGain, Footsteps } from "./audio/cues";
import { noteTable, noteToPlay } from "./audio/notes";
import { GameAudio } from "./audio/game-audio";
import { browserStore, loadSecret, resetSecret } from "./join/identity";
import { parseInvite } from "./join/invite";
import { screenForRejection, type Action, type Screen } from "./join/outcome";
import {
  encodeEditBlock,
  encodeHitMob,
  encodeHitPlayer,
  encodeTalkTo,
  encodeAnswer,
  encodeUseBlock,
  encodeOpenScript,
  encodeSaveScript,
  encodeEmote,
  encodeMove,
  encodeCraft,
  encodeMoveItem,
  encodeRequestChunks,
  encodeRequestFar,
  encodeImportMap,
  encodeSendChat,
  encodeSubmitArt,
  type Bytes,
  type JoinInit,
} from "./net/codec";
import { Connection, type ConnectionEvents } from "./net/connection";
import { socketUrl } from "./net/url";
import {
  ArtKind,
  GameMode,
  Harm,
  type BlockChanged,
  type BlockType,
  type Emote,
  type PlayerInfo,
  type PlayerPose,
  type ServerMessage,
  type Teleport,
  type Texture,
  type Vitals,
  type Quest,
  type Welcome,
  type WorldInfo,
} from "./pb/scabbage/v1/net_pb";
import {
  aim,
  breakEdit,
  buildWorldFor,
  digMs,
  placeEdit,
  PressFilter,
  UseHold,
  usesBlock,
  type BuildWorld,
  type Edit,
} from "./player/building";
import { brushCube, MAX_BRUSH, nextBrush, type Cell } from "./player/brush";
import { Hotbar, hotbarBlocks } from "./player/hotbar";
import { Inventory } from "./player/inventory";
import { Npcs } from "./player/npcs";
import { Miner } from "./player/mining";
import { MOB_REACH, MOB_STEP_MS, type MobState, Mobs, pickMob } from "./player/mobs";
import { GamepadControls } from "./player/gamepad";
import { combineIntents, type BuildActions } from "./player/input";
import { KeyboardMouse } from "./player/keyboard-mouse";
import { MoveSync, type Pose } from "./player/move-sync";
import type { BoxesAt } from "./player/physics";
import { Player } from "./player/player";
import { pickPlayer, RemotePlayers } from "./player/remote-players";
import { Avatars, skinPixels } from "./render/avatars";
import { ArenaMarks } from "./render/arena-marks";
import { Drops } from "./render/drops";
import { MobView } from "./render/mob-view";
import { blockIconPainter } from "./render/icons";
import { TEXTURE_NAMES } from "./render/textures";
import { DEBUG_FRAMES, startScene } from "./render/scene";
import { Selection } from "./render/selection";
import { createWorldRenderer } from "./render/world-renderer";
import { AutoQuality } from "./settings/auto-quality";
import { deviceTier, readDevice } from "./settings/device";
import { defaultSettings, loadSettings, saveSettings, type Settings } from "./settings/settings";
import { emojiFor } from "./social/emotes";
import { HealthView } from "./ui/health-view";
import { ScoreboardView } from "./ui/scoreboard-view";
import { ScriptEditor } from "./ui/script-editor";
import { scriptBlockNames } from "./ui/script-names";
import { adminRouter, ArtList, InvitePanel, PendingList } from "./ui/admin";
import { BlockPicker } from "./ui/block-picker";
import { ChatView } from "./ui/chat-view";
import { $, el } from "./ui/dom";
import { EmoteWheel, popEmoji } from "./ui/emote-wheel";
import { mountHotbar } from "./ui/hotbar-view";
import { DialogueView } from "./ui/dialogue-view";
import { isMenuKey } from "./ui/hotkey";
import { QuestLogView } from "./ui/quest-log-view";
import { InventoryView } from "./ui/inventory-view";
import { openPixelEditor, type EditorMode, type Submission } from "./ui/pixel-editor";
import {
  canGoFullscreen,
  registerServiceWorker,
  toggleLandscape,
  unlockOnFullscreenExit,
} from "./ui/pwa";
import { settingsPanel } from "./ui/settings-panel";
import { lookById } from "./looks/registry";
import { TouchControls } from "./ui/touch-controls";
import { collisionBoxes, FULL_BLOCK, type Box } from "./world/shapes";
import { lookDirection, type RayHit, type Vec3 } from "./world/raycast";
import { solidQuery } from "./world/solid";
import { zoneGreeting, Zones } from "./world/zones";
import type { WorldBounds } from "./world/streamer";
import { hideOverlay, showJoinForm, showMessage } from "./ui/screens";
import { downloadMap, mapImportName, MAX_MAP_BYTES, pickMapFile } from "./ui/map-files";

const overlay = $("#overlay");
const statusEl = $("#status");
const pingEl = $("#ping");
const coordsEl = $("#coords");
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
const world = createWorldRenderer(
  gameScene,
  (chunks) => conn?.send(encodeRequestChunks(chunks)),
  (columns) => conn?.send(encodeRequestFar(columns)),
  lookById(settings.look),
);
const player = new Player();
const selection = new Selection(gameScene.scene);
const hotbar = new Hotbar([]);
const picker = new BlockPicker((block) => {
  hotbar.pick(block);
  picker.node.closest("dialog")?.close();
});
const inventory = new Inventory();
const backpack = new InventoryView(
  inventory,
  (from, to) => conn?.send(encodeMoveItem(from, to)),
  (recipe) => conn?.send(encodeCraft(recipe)),
);
const miner = new Miner();
let brush = MAX_BRUSH; // blocks a side we break and place at once
const presses = new PressFilter(250); // input repeats held buttons every 250 ms
const useHold = new UseHold(250);
const npcs = new Npcs();
const dialogue = new DialogueView((id, choice) => conn?.send(encodeAnswer(id, choice)));
const questLog = new QuestLogView();
let survival = false;
const zones = new Zones();
const arenaMarks = new ArenaMarks(gameScene.scene);
let zoneHere: string | undefined; // the zone we stand in, to greet on entering
let refusedAtMs = -Infinity; // when we last said a zone is protected
inventory.onChange = () => {
  if (survival) hotbar.carry(inventory.slots);
  backpack.draw();
};
let locked = false;
const chat = new ChatView(
  $("#chat"),
  (text) => {
    const importing = mapImportName(text);
    if (importing === undefined) conn?.send(encodeSendChat(text));
    else void importMap(importing);
  },
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
const npcAvatars = new Avatars(gameScene.scene); // characters, apart so their ids never meet players'
const audio = new GameAudio(settings.sound, settings.music);
const footsteps = new Footsteps();
const health = new HealthView($("#health"), $("#hurt"), $("#faint"));
const scoreboard = new ScoreboardView($("#scoreboard"));
const drops = new Drops(gameScene.scene);
const scriptEditor = new ScriptEditor(
  (at, source) => conn?.send(encodeSaveScript(at, source)),
  () =>
    !touch &&
    $("#game")
      .requestPointerLock()
      .catch(() => undefined),
);
let scriptBlocks = new Set<number>(); // ids of the script block and its powered state
let toldAboutScripts = false; // whether we said how to open a script block
const mobView = new MobView(gameScene.scene);
const bossBar = new BossBar();
document.body.append(bossBar.node);
let moves: MoveSync | undefined; // set while in the world
let others: RemotePlayers | undefined; // set while in the world
let mobs: Mobs | undefined; // set while in the world
let solid: BoxesAt = () => FULL_BLOCK;
let building: BuildWorld | undefined;
let notes = noteTable([]);
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
  if (moves) greetZone();
  const eye = player.eye();
  gameScene.camera.position.set(eye.x, eye.y, eye.z);
  gameScene.camera.rotation.set(player.look.pitch, player.look.yaw, 0, "YXZ");
  world.tick(eye.x, eye.z, settings.viewDistance);
  const near = mobs?.sample(performance.now()) ?? [];
  mobView.update(near, dt * 1000);
  bossBar.update(moves && bossNear(near, player.body));
  showCoords(moves ? coordsText(player.body, player.look.yaw) : "");
  build(intent.build, intent.move.descend, [eye.x, eye.y, eye.z], dt, near);
  drops.update(timeMs);
  crosshairEl.hidden = !(moves && (locked || touch || gamepad.active));
  if (gamepad.active) hintEl.hidden = true;
  avatars.update(others?.sample(performance.now()) ?? [], performance.now());
  npcAvatars.update(npcs.figures(player.body), performance.now());
  const pose = moves?.due(timeMs, player.pose());
  if (pose && moves) conn?.send(encodeMove(pose, moves.teleport));
});
const drawn = { view: 0, objects: 0 };
if (DEBUG_FRAMES)
  window.setInterval(() => {
    $("#fps").textContent = `${drawn.view + drawn.objects} fps drawn (${drawn.view} view)`;
    drawn.view = drawn.objects = 0;
  }, 1000);
gameScene.onDrawn((timeMs, steady, change) => {
  if (change !== "none") drawn[change]++;
  const lighter = moves && autoQuality.frame(timeMs, settings, steady);
  if (lighter) {
    changeSettings(lighter);
    settingsView.show(lighter);
  }
});
const settingsView = settingsPanel(settings, changeSettings);
addHudDialog("Settings", settingsView.node);
addHudDialog("Blocks", el("div", {}, picker.node, backpack.node), "KeyB");
addHudDialog("Quests", questLog.node, "KeyQ");
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
          mobs = undefined;
          scoreboard.hide();
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
        else if (msg.body.case === "farLand") world.receiveFar(msg.body.value);
        else if (msg.body.case === "chunksStale") world.refetch(msg.body.value.chunks);
        else if (msg.body.case === "mapExported")
          downloadMap(msg.body.value.name, msg.body.value.data);
        else if (msg.body.case === "teleport") teleport(msg.body.value);
        else if (msg.body.case === "vitals") applyVitals(msg.body.value);
        else if (msg.body.case === "scoreboard") scoreboard.show(msg.body.value);
        else if (msg.body.case === "scriptShown") scriptEditor.show(msg.body.value);
        else if (msg.body.case === "npcsChanged")
          npcs.change(msg.body.value.npcs, msg.body.value.gone);
        else if (msg.body.case === "dialogue") {
          dialogue.show(msg.body.value);
          audio.play("pop", 0.6);
        } else if (msg.body.case === "questLog") questNews(msg.body.value.quests);
        else if (msg.body.case === "blockChanged") blockChanged(msg.body.value);
        else if (msg.body.case === "blockBatch") blocksChanged(msg.body.value.blocks);
        else if (msg.body.case === "inventory") {
          if (inventory.set(msg.body.value.slots)) audio.play("pop", 0.5);
        } else if (msg.body.case === "itemsChanged")
          drops.change(msg.body.value.drops, msg.body.value.gone);
        else if (msg.body.case === "zonesChanged") {
          zones.set(msg.body.value.zones);
          arenaMarks.set(msg.body.value.zones);
        } else if (msg.body.case === "mobsChanged")
          mobs?.change(msg.body.value.mobs, msg.body.value.gone, performance.now());
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

/** Picks a .scabmap file and sends it to be placed as the zone `name` around us. */
async function importMap(name: string): Promise<void> {
  const data = await pickMapFile();
  if (!data) return;
  if (data.length > MAX_MAP_BYTES) chat.note(`Map files can be at most ${MAX_MAP_BYTES >> 20} MB.`);
  else conn?.send(encodeImportMap(name, data));
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
  if (w.inventory) inventory.set(w.inventory.slots);
  if (w.vitals) applyVitals(w.vitals);
  if (w.start) teleport(w.start); // after vitals, which decide whether we may fly
  if (w.quests) questLog.start(w.quests.quests);
}

/** Shows the changed quest log and tells us what is new in it. */
function questNews(quests: Quest[]): void {
  const news = questLog.update(quests);
  for (const line of news) chat.note(line);
  if (news.length) audio.play("pop", 0.8);
}

/** Our own health and game mode: survival walks, carries its blocks and shows hearts. */
function applyVitals(v: Vitals): void {
  survival = v.mode === GameMode.SURVIVAL;
  player.allowFlying(!survival);
  hotbar.carry(survival ? inventory.slots : undefined);
  picker.node.hidden = survival;
  backpack.node.hidden = !survival;
  health.show(v);
  if (v.hurt !== Harm.UNSPECIFIED) audio.play("hurt");
}

function enterWorld(info: WorldInfo, tickRateHz: number): void {
  world.start(info);
  drops.clear(); // the server lists every item again on joining
  npcs.clear(); // and every character
  bounds = { radius: info.radius, height: info.height };
  hotbar.slots.splice(0, Infinity, ...hotbarBlocks(info.blocks));
  hotbar.select(0);
  applyBlocks(info.blocks, info.textures);
  backpack.setRecipes(info.recipes.map((r) => ({ make: stack(r.make), from: r.from.map(stack) })));
  const spawn = info.spawn;
  player.place(spawn?.x ?? 0, spawn?.y ?? 0, spawn?.z ?? 0);
  moves = new MoveSync(1000 / Math.max(1, tickRateHz));
  others = new RemotePlayers(1000 / Math.max(1, tickRateHz));
  mobs = new Mobs(MOB_STEP_MS); // the server lists every mob again on joining
  zones.set([]);
  arenaMarks.set([]);
  zoneHere = undefined;
  hintEl.hidden = document.pointerLockElement !== null || touch;
  $("#touch").hidden = !touch;
  chat.enabled = true;
  document.body.classList.add("in-world");
}

const stack = (s?: { block: number; count: number }) => ({
  block: s?.block ?? 0,
  count: s?.count ?? 0,
});

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
  notes = noteTable(blocks);
  scriptBlocks = new Set(blocks.filter((b) => b.name.startsWith("script_block")).map((b) => b.id));
  scriptEditor.setBlockNames(scriptBlockNames(blocks));
  const name = (id: number) => blocks.find((b) => b.id === id)?.name ?? "missing";
  const icon = blockIconPainter(atlas, table);
  mountHotbar($("#hotbar"), hotbar, name, icon);
  picker.set(blocks, icon);
  backpack.setBlocks(icon, name);
  drops.setIcons(icon);
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

/** The server put us back somewhere, usually after rejecting a move, or where we were saved. */
function teleport(t: Teleport): void {
  player.place(t.pos?.x ?? 0, t.pos?.y ?? 0, t.pos?.z ?? 0);
  if (t.restore) player.restore(t.restore);
  moves?.teleported(t.id);
}

/** Outlines the block in reach and carries out this frame's building actions, or hits a mob or player. */
function showCoords(text: string): void {
  if (coordsEl.textContent !== text) coordsEl.textContent = text;
}

function build(
  act: BuildActions,
  sneaking: boolean,
  eye: [number, number, number],
  dt: number,
  near: readonly MobState[],
): void {
  if (act.resizing) resizeBrush();
  const hit = building && aim(eye, player.look.yaw, player.look.pitch, building);
  const aimed = aimedAt(eye, near);
  const npc = npcs.pick(eye, lookDirection(player.look.yaw, player.look.pitch), MOB_REACH);
  if (npc && npc.distance < Math.min(hit?.distance ?? Infinity, aimed?.distance ?? Infinity)) {
    miner.stop();
    selection.show(undefined);
    if (act.placing && presses.fresh(performance.now())) talkTo(npc.id);
    return;
  }
  if (aimed && (!hit || aimed.distance < hit.distance)) {
    miner.stop();
    selection.show(undefined);
    if (act.breaking) strike(aimed.hit);
    return;
  }
  const target = hit && building?.blockAt(hit.x, hit.y, hit.z);
  const onScript = target !== undefined && scriptBlocks.has(target);
  touchControls?.aimingAtScript(!!onScript);
  if (onScript) tellAboutScripts();
  if (hit && onScript && act.scripting) conn?.send(encodeOpenScript(hit));
  if (hit && act.placing && building && usesBlock(target, sneaking, building)) {
    miner.stop();
    selection.show(hit);
    if (presses.fresh(performance.now())) sendUse(hit);
    useHold.used(performance.now());
    return;
  }
  if (hit && !zones.canBuild(hit.x, hit.z)) {
    miner.stop();
    selection.show(hit);
    if (act.breaking || act.digging || act.placing) refuse(hit.x, hit.z);
    return;
  }
  const dug = survival ? dig(act.digging, hit, dt) : 0;
  selection.show(hit && brushCube(hit, brush, false), dug);
  if (!hit || !building) return;
  if (act.picking) {
    const block = building.blockAt(hit.x, hit.y, hit.z) ?? 0;
    hotbar.pick(building.item[block] ?? block);
  }
  const breaking = survival ? dug === 1 : act.breaking;
  const edit = breaking
    ? breakEdit(hit, building, brush)
    : act.placing && !useHold.blocks(performance.now())
      ? placeEdit(hit, hotbar.current(), player.body, player.look, building, brush)
      : undefined;
  if (edit) sendEdit(edit);
}

/** Says once how to open a script block, the first time we aim at one. */
function tellAboutScripts(): void {
  if (toldAboutScripts) return;
  toldAboutScripts = true;
  const how = touch ? "Tap 📜" : gamepad.active ? "Press Back" : "Press R";
  chat.note(`${how} to read or write this script block's script. Using it runs on_touch.`);
}

/** Survival mining: how far the held brush is dug, from 0 to 1 when it breaks. */
function dig(held: boolean, hit: RayHit | undefined, dt: number): number {
  const edit = held && hit && building ? breakEdit(hit, building, brush) : undefined;
  if (!edit || !building) {
    miner.stop();
    return 0;
  }
  return miner.dig(dt, edit, digMs(edit, building));
}

/** Switches to the next brush size and says what it is. */
function resizeBrush(): void {
  brush = nextBrush(brush);
  miner.stop();
  const wide = brush === MAX_BRUSH ? " (a big block)" : "";
  chat.note(`Building ${brush} block${brush === 1 ? "" : "s"} wide${wide}. G changes it.`);
}

/** Says, now and then, that this zone is protected. */
function refuse(x: number, z: number): void {
  const now = performance.now();
  if (now - refusedAtMs < 3000) return;
  refusedAtMs = now;
  chat.note(`Only builders of ${zones.at(x, z)?.name ?? "this zone"} can change blocks here.`);
}

/** Welcomes us into each zone we walk into. */
function greetZone(): void {
  const zone = zones.at(Math.floor(player.body.x), Math.floor(player.body.z));
  if (zone?.name === zoneHere) return;
  zoneHere = zone?.name;
  if (zone) chat.note(zoneGreeting(zone));
}

/** The nearest mob, or player we may tag, in reach along our look, with the message that hits it. */
function aimedAt(
  eye: Vec3,
  near: readonly MobState[],
): { distance: number; hit: Bytes } | undefined {
  const dir = lookDirection(player.look.yaw, player.look.pitch);
  const mob = pickMob(eye, dir, near, MOB_REACH);
  const rival = mayTag()
    ? pickPlayer(eye, dir, others?.sample(performance.now()) ?? [], MOB_REACH)
    : undefined;
  if (rival && (!mob || rival.distance < mob.distance))
    return { distance: rival.distance, hit: encodeHitPlayer(rival.entity) };
  return mob && { distance: mob.distance, hit: encodeHitMob(mob.id) };
}

/** Whether we stand where survival players may hit each other. */
function mayTag(): boolean {
  return survival && (zones.at(Math.floor(player.body.x), Math.floor(player.body.z))?.pvp ?? false);
}

function strike(hit: Bytes): void {
  if (!conn || !moves) return;
  const pose = moves.flush(player.pose()); // the server checks reach from our latest position
  if (pose) conn.send(encodeMove(pose, moves.teleport));
  conn.send(hit);
  audio.play("bonk");
}

/** Asks a character's script block to run its on_talk. */
function talkTo(npc: number): void {
  if (!conn || !moves) return;
  const pose = moves.flush(player.pose()); // the server checks reach from our latest position
  if (pose) conn.send(encodeMove(pose, moves.teleport));
  conn.send(encodeTalkTo(npc));
}

/** Presses a switch or button; anyone may, even in a protected zone. */
function sendUse(hit: RayHit): void {
  if (!conn || !moves) return;
  const pose = moves.flush(player.pose()); // the server checks reach from our latest position
  if (pose) conn.send(encodeMove(pose, moves.teleport));
  conn.send(encodeUseBlock(hit));
  audio.play("place");
}

/** Shows the edit straight away; the server confirms it or sends back what is really there. */
function sendEdit(edit: Edit): void {
  if (!conn || !moves) return;
  if (!zones.canBuild(edit.x, edit.z)) return refuse(edit.x, edit.z);
  const pose = moves.flush(player.pose()); // the server checks reach from our latest position
  if (pose) conn.send(encodeMove(pose, moves.teleport));
  conn.send(encodeEditBlock(edit, hotbar.selected));
  for (const c of affordable(edit)) world.setBlock(c.x, c.y, c.z, edit.block);
  audio.play(edit.block === 0 ? "break" : "place");
}

/** The blocks of an edit we can pay for, spending them; survival placements use up the held slot. */
function affordable(edit: Edit): Cell[] {
  if (!survival || edit.block === 0) return edit.cells;
  let n = 0;
  while (n < edit.cells.length && inventory.spend(hotbar.selected)) n++;
  return edit.cells.slice(0, n);
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

/** Many blocks changing at once, like a brush or a circuit, playing each sound only once. */
function blocksChanged(changes: readonly BlockChanged[]): void {
  const played = new Set<string>();
  for (const c of changes) blockChanged(c, played);
}

/**
 * Someone's edit, or our own echoed back; only changes we haven't already shown make a sound.
 * Logic blocks switching state are quiet, except note blocks, which play.
 */
function blockChanged(c: BlockChanged, played = new Set<string>()): void {
  if (!c.pos) return;
  const { x, y, z } = c.pos;
  const was = world.blockAt(x, y, z);
  if (was === c.block) return;
  world.setBlock(x, y, z, c.block);
  const b = player.body;
  const gain = distanceGain(Math.hypot(x + 0.5 - b.x, y + 0.5 - b.y, z + 0.5 - b.z));
  const note = noteToPlay(notes, was, c.block);
  const switched = was !== undefined && building?.item[was] === building?.item[c.block];
  const sound =
    note !== undefined ? `note ${note}` : switched ? "" : c.block === 0 ? "break" : "place";
  if (!sound || played.has(sound)) return;
  played.add(sound);
  if (note !== undefined) audio.playNote(note, gain);
  else audio.play(c.block === 0 ? "break" : "place", gain);
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

/** Adds a HUD button, and optionally a key that toggles it, that opens `content` in a modal dialog. */
function addHudDialog(label: string, content: Node, key?: string): void {
  const dialog = el(
    "dialog",
    { class: "card" },
    content,
    el("button", { onclick: () => dialog.close() }, "Close"),
  );
  document.body.append(dialog);
  $("#hud-buttons").append(el("button", { onclick: () => dialog.showModal() }, label));
  if (key) document.addEventListener("keydown", (e) => toggleByKey(e, key, dialog));
}

/** Opens or closes `dialog` on its key, unless another dialog or the join screen is up. */
function toggleByKey(e: KeyboardEvent, key: string, dialog: HTMLDialogElement): void {
  if (!isMenuKey(e, key) || !document.body.classList.contains("in-world")) return;
  const other = document.querySelector("dialog[open]");
  if (other && other !== dialog) return;
  e.preventDefault();
  if (dialog.open) return dialog.close();
  if (document.pointerLockElement) document.exitPointerLock();
  dialog.showModal();
}

function addHudButton(icon: string, label: string, onclick: () => void): void {
  $("#hud-buttons").append(el("button", { "aria-label": label, title: label, onclick }, icon));
}

function changeSettings(s: Settings): void {
  settings = s;
  saveSettings(store, s);
  gameScene.apply(s);
  world.setLook(lookById(s.look));
  audio.set(s.sound, s.music);
}
