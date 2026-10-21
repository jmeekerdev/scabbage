# Scabbage — Project Plan

A family-friendly, browser-based, 3D voxel game for 2–6 players (ages 8+ through adults)
on Android, iOS, Linux, macOS and Windows. Sandbox building, survival, adventure and
minigames all happen in one shared, persistent world.

## Decisions so far

| Area | Decision |
|---|---|
| Genre | All-in-one: a voxel sandbox world is the base, and survival, adventure and minigames are layered on top |
| Art | Blocky voxel; textures are 16×16 pixel art the family can make |
| Players | 2–6 at once, ages 8+ |
| Client | TypeScript + Three.js (WebGL2), built with Vite, installable as a PWA |
| Server | Go, authoritative, a single binary that also serves the client's static files |
| Transport | WebSocket carrying binary Protobuf messages (WebTransport later if needed) |
| Hosting | A home server reachable from the internet, run with Docker / docker compose; budget ≤2 GB RAM and ~2 CPU cores |
| TLS | Recommended, not required. The server speaks plain HTTP; a reverse proxy (Caddy) adds TLS |
| Efficiency | As close to zero work as possible when nobody is playing; lean dependencies, no bloat |
| World | Starts bounded (a world border), designed so it can later open up to infinite |
| Login | Join code or invite link, then an admin approves each new device |
| Social | Text chat plus quick emotes and stickers |
| Input | Keyboard + mouse, touch, and gamepad, all treated as first-class |
| Creativity | Logic blocks, a minigame maker, kid-friendly scripting, custom skins and blocks |
| Plan location | This repo (`docs/`) |

## Architecture

```
  Browser (any device)                        Home server (Docker)
 ┌───────────────────────────┐   WSS    ┌──────────────────────────────────┐
 │ UI (chat, menus, HUD)     │◄────────►│ Caddy (TLS, reverse proxy)       │
 │ Input: KB/M, touch, pad   │ protobuf │   │                              │
 │ Three.js renderer         │          │   ▼                              │
 │ Web Worker: chunk meshing │          │ scabbage-server (Go)             │
 │ Client-side prediction    │          │  ├─ gateway: join/approve/auth   │
 │ IndexedDB: token, cache   │          │  ├─ sim: 20 Hz tick, world state │
 └───────────────────────────┘          │  ├─ world: chunks, gen, storage  │
                                        │  ├─ chat/emotes                  │
                                        │  ├─ (later) logic, minigames,    │
                                        │  │   scripting (sandboxed Lua)   │
                                        │  └─ admin web UI                 │
                                        │ SQLite (accounts, chunks, saves) │
                                        └──────────────────────────────────┘
```

Key principles:
- **The server is authoritative.** Clients send *intents* (move, place a block, chat), and the
  server validates them and broadcasts the results. The client predicts its own movement so
  play feels responsive, then reconciles with the server.
- **One schema.** `proto/*.proto` generates both the Go and TS types (`buf` + `protobuf-es`),
  so the two sides can't drift apart.
- **Chunks are 16×16×16 cubes.** They are sent compressed (palette + RLE) and meshed in a Web
  Worker so the main thread stays smooth on phones.
- **Pure-Go dependencies** (`modernc.org/sqlite`, `coder/websocket`, `gopher-lua`), so there is no cgo
  and the Docker images are small and simple.

## Efficiency and idle behavior

Targets: at idle, **~0% CPU and under ~30 MB RSS**. In a session with 6 players, well under 500 MB.

- **Idle mode:** when the last player leaves, the server stops its tick loop and flushes
  changed chunks to SQLite. It then unloads the world from memory and waits on network
  input only, with no timers. The first player to connect wakes it again; chunks reload
  lazily from SQLite as they are needed. (`sim.Config.OnIdle` runs after the loop stops.)
- **Runtime limits:** `GOMEMLIMIT` is set below the container limit
  (`mem_limit: 2g`, `cpus: 2`), and the Go GC returns memory to the OS after the world unloads.
- **Only active areas run:** chunks are loaded only near players. Mob AI, logic circuits
  and scripts run only in loaded chunks.
- **Compact data:** uniform chunks (all air, all stone) are stored as one block in memory; the
  palette + RLE encoding is used on the wire and in SQLite (a surface chunk is ~0.6 KB, a uniform
  one 3 bytes), with no extra compressor. The server only sends what changed each tick.
- **Few dependencies:** small, pure-Go server dependencies; the only big client dependency is
  Three.js, which is tree-shaken. Bundle size is tracked in `make check`.
- **Players can tune performance themselves:** view distance, graphics presets and a
  frame-rate cap are client settings. A weak phone can lower them without affecting anyone else.

## TLS and plain HTTP

The Go server only speaks HTTP/WS. TLS comes from Caddy (in the compose file, optional).
- **Local dev:** `http://localhost` counts as a secure context, so everything works without TLS.
- **Plain HTTP over a LAN IP or the internet** still works for playing, but the browser turns
  off some features: no PWA install or service worker, and Chrome may block the Gamepad API.
  The client feature-detects these, degrades gracefully, and shows a hint suggesting HTTPS.
- Nothing security-critical depends on `crypto.subtle` (unavailable without TLS). Device
  secrets use `crypto.getRandomValues`. Without TLS, though, tokens can be sniffed, so admins
  get a warning on the admin page.

## World bounds, from bounded to infinite

- Chunk coordinates are `int32` everywhere, and nothing in storage or the protocol assumes a size.
- The bound is just a configurable **world border** (start at e.g. 1024×1024 blocks).
  Opening the world up means raising or removing the border.
- World gen is deterministic from `(seed, generator version)`, so new land generated
  later matches up with old land without seams. If the generator algorithm changes, it gets a
  new version. Every chunk is saved once generated, so existing land never changes. A golden-hash
  test pins each generator version, and noise math avoids fused multiply-adds so it is identical on
  amd64 and arm64.
- Default border: 1024×1024 blocks (radius 512, stored per world), height 0–127, sea level 30.
  Starter blocks: stone, dirt, grass, sand, water, log, leaves, planks, cobblestone, glass, brick,
  bedrock.

## Join and approval flow

1. **First admin:** while no admin exists, the server logs a single-use admin code at startup
   (`docker compose logs scabbage`). Joining with it makes you admin immediately.
2. An admin makes an invite (in game: **Invite**, or on `/admin.html`) for 1–10 people, valid for an
   hour, a day or a week. The link looks like `http://host:8080/#join=ABCD-EFGH`. The code is in the
   URL hash, so the static server needs no routes, and the client strips it from the address bar.
3. A new player opens the link and picks a display name and an avatar color. The device makes a
   random 32-byte **device secret** (`crypto.getRandomValues`, stored in `localStorage`), and the
   server stores only its SHA-256 hash. The request is **pending**: the connection stays open
   (answering pings) and resumes after a reconnect.
4. Any online admin sees a toast with **Let in / No** (also listed on `/admin.html`). If the name
   already belongs to a player, the request is flagged "returning", and approving it adds the new
   device to that player. Approval registers the device secret; there is no separate token.
5. Limits: 5 bad codes per IP per 10 minutes (X-Forwarded-For is trusted only from private/loopback
   peers, e.g. Caddy), at most 10 pending requests, and pending requests expire after 24 h. A code
   without an approval grants nothing.
6. **iOS caveat:** Safari can wipe site storage after about 7 days without a visit unless the game
   is installed as a PWA, so we encourage installing it. Re-approving a returning player is one click.
7. Admin-only messages (`DecideJoin`, `CreateJoinCode`) are checked server-side. The admin page
   connects in *console* mode, which does not wake the world.
8. Admins can make other players admins, or remove their admin rights, from **Players** on
   `/admin.html` or with `/admin NAME` and `/unadmin NAME` in chat. Admins can't change their own
   rights, so at least one admin always remains. The player is reconnected to pick up the change.

## Repo layout (planned)

```
proto/            Protobuf schema (source of truth for the protocol)
server/           Go module: cmd/scabbage-server, internal/{gateway,sim,world,chat,store,admin}
client/           TS + Vite + Three.js: src/{net,render,input,ui,world,worker}
assets/           Block textures, sounds, default skins
docs/             Plan, design notes, protocol notes
docker/           Dockerfile(s), Caddyfile
compose.yaml      Production-ish stack (server + caddy + volume)
shell.nix         Dev tools (Go, Node, buf, linters); `make dev` hot-reloads the client
```

## Testing approach (TDD)

- **Go:** table-driven unit tests written first for world gen, chunk encoding, collision,
  rules, and the approval state machine. Simulation tests run several fake clients
  against an in-process server.
- **TS:** Vitest for meshing, input mapping, prediction/reconciliation and protocol codecs.
- **E2E:** Playwright drives two browser contexts (desktop + emulated mobile) that join,
  place blocks and chat, checking that each sees the other's changes.
- **Lint/format:** `golangci-lint`, `buf lint`, `eslint`, `tsc --noEmit`, and `prettier`, all
  run by one `make check` (also runnable inside Docker).

## Phases

Each phase ends with something the family can play. Compact the model context between phases.

### Phase 0 — Foundations
- [x] git init, `.gitignore`, `Makefile` (`check`, `test`, `dev`, `build`)
- [x] Dockerfile (multi-stage: node build → go build → distroless), `compose.yaml` with
      resource limits, optional Caddy TLS profile
- [x] Idle test: a test that fails if the server does any work (ticks or timers) with zero players
- [x] `proto/` with a first message set: Hello, Welcome, Ping/Pong; buf codegen for Go + TS
- [x] Go server skeleton: config, structured logging, health endpoint, WebSocket endpoint
- [x] Client skeleton: Vite + Three.js; renders a spinning cube and shows ping to the server
- [x] CI-ready `make check` passing with tests in place

### Phase 1 — Shared Build World (Milestone 1)
- [x] Join code → pending → admin approve → device registered; admin page (`/admin.html`) + in-game approve toasts
      (`/approve <name>` lands with chat)
- [x] World: chunk storage in SQLite, versioned seeded terrain gen, world border, ~10 starter blocks
- [x] Idle mode: stop ticking and unload the world when empty, wake when someone joins
- [x] Client settings: view distance, graphics preset, FPS cap (saved per device; touch starts lighter)
- [x] Chunk streaming by distance; meshing (greedy) in a Web Worker; texture atlas
      (procedural textures in one array texture so merged faces tile cleanly)
- [x] Player movement: walk/jump/fly toggle, AABB collision, client prediction + reconciliation
      (client simulates at 60 Hz and reports its pose; the server checks speed budgets and
      collision along the path, and sends a Teleport back on a bad move. Chunk requests are
      limited to the player's surroundings. Basic KB/M controls for now)
- [x] Block break/place with server validation and a broadcast to everyone
      (client shows its edit at once; the server checks reach, rate, replaceable target and
      player overlap, then broadcasts BlockChanged or corrects the sender. Crosshair, outline,
      9-slot hotbar, hold to repeat, middle click picks a block)
- [x] Other players visible (simple blocky avatars + name tags, interpolated)
- [x] Input: pointer-lock KB/M; touch joystick + look-drag + buttons; Gamepad API
- [x] Text chat + emote wheel
      (T, Enter or / to type; lines fade; `/approve NAME` and `/deny NAME` for admins; hold E and
      point, tap 😀, or use the D-pad to emote; emojis float over avatars)
- [x] PWA manifest + service worker; landscape fullscreen on phones
      (installable over HTTPS; ⛶ toggles fullscreen sideways on Android;
      iPhones get a turn-sideways hint, or install to the home screen; hashed assets cached for good)
- [x] Autosave + nightly world backup snapshot
      (saves every minute while anyone plays; 04:00 copies kept 7 nights in data/backups)
- **Done when:** all family devices are in one world at once, building together, and the
  world is still there after a server restart.

### Phase 2 — Look & Feel + Personalization
- [x] Better avatars (rigged blocky characters, animations), sounds, music toggle
      (synthesized in the browser, no audio files; music starts off)
- [x] Ambient occlusion + simple sky/day–night cycle (20-minute day from the wall clock, so every device agrees; nights stay playable)
- [x] In-browser pixel editor: avatar skins and custom block textures (admin-approved)
      (🎨 Create: skins in the classic 64×32 layout with a 3D preview, new blocks (up to 64, IDs from 256),
      admins repaint built-in textures; Blocks dialog picks any block into the hotbar)
- [x] Larger block palette: glass, light sources, stairs/slabs, colored wool
      (16 wool colors, lamp and torch with client-side flood-fill light, slabs and stairs in
      stone, cobblestone, planks and brick with box collision and a half-block step-up)
- [x] Graphics quality presets (auto-detect weaker phones)
      (starting preset from cores and memory; lowers the preset, then view distance, when
      frames stay under 70% of target, until the player picks settings themselves)

### Phase 3 — Survival
- [x] Health, damage, respawn; survival vs. creative game modes per area or player
      (server-side fall damage, water breaks falls, slow healing, faint → back to spawn;
      modes per player via admin page or `/survival NAME`, server default `-mode`;
      per-area modes come with protected zones)
- [x] Inventory, hotbar, item drops, mining time per block
      (survival only: 36 saved slots with the hotbar first, blocks take 0.2–1.5 s to dig and drop
      an item (stone → cobblestone, grass → dirt) that is picked up by walking near it and expires
      after 5 minutes; placing uses up the selected slot; the server times digs and checks slots;
      creative keeps its unlimited hotbar and instant breaking)
- [x] Crafting recipes (data-driven) (shapeless recipes in an embedded JSON book, replaceable by
      `data/recipes.json`; sent in WorldInfo; the Backpack's "Make" row crafts with a tap and dims
      what you can't afford; the server checks inputs and room before trading; survival only)
- [x] Mobs: passive animals + night monsters with simple server-side AI
      (sheep graze near players by day and drop wool when hit, which regrows; slimes come out at
      night only near survival players, hop after them and bite, pop after three hits and melt at
      dawn; the server steps them 10×/s, hops one-block ledges, avoids water and cliffs, caps their
      numbers and clears them when everyone leaves)
- [x] Protected zones (spawn, people's houses) so nobody gets griefed; zones can set the game mode
      (admins-only zone around spawn, half-width `-spawn-zone`, default 16; admins run
      `/zone add NAME SIZE`, `remove`, `allow`/`disallow NAME PLAYER`, `mode NAME creative|survival|any`
      and `list`; zones are columns saved in SQLite, the smallest overlapping one wins; the server
      checks every edit and the client refuses early and greets you on entering)

### Phase 4 — Logic Blocks
- [x] Signal-propagation system (server tick): wire, switch, button, pressure plate, lamp
      (`internal/logic`: each block has an off and an on ID, so power needs no extra storage.
      A circuit is re-powered only when one of its blocks changes, walking at most 4096 wires;
      the tick only lets go of buttons after 1 s and presses plates under players' feet.
      Anyone may press switches and buttons, even in protected zones.)
- [x] Doors, pistons, timers, counters, teleporters, note blocks
      (Every state is its own block ID, still with no extra storage. Doors open as a whole
      column, by hand or by power. Pistons face six ways and push up to 12 blocks one step
      when power arrives; they don't pull, and never push logic blocks or across a zone's edge.
      Loaded timers blink once a second, using a per-chunk index of ticking blocks. Counters
      count to 9 and then give power until someone resets them by hand. A teleporter sends you
      to the next pad joined to it by wire. Note blocks play a C-major pitch on the client when
      powered or retuned by hand. Logic blocks switching state no longer make place sounds.)
- [x] Deterministic tests of logic circuits
      (Hundreds of seeded random circuits must settle in one pass and change the same way every
      run. They found that doors taller than four split in two, so a door is now its whole column.
      Doors never close on a player; they wait until everyone has stepped out.)

### Phase 5 — Minigame Maker
- [x] Mark an area as an arena: spawns, teams, goal, timer, win condition
      (an arena is a saved zone's minigame setup, stored as JSON beside it and removed with it;
      admins run `/arena spawn|goal ZONE [red|blue|green|yellow]` to mark where they stand,
      `timer ZONE SECONDS`, `win ZONE last|goal|points N`, `clear ZONE` and `list`; teams are the
      colors that have spawns)
- [x] Built-in templates: race, capture-the-flag, tag, spleef, last-one-standing
      (`/arena game ZONE race|flags|tag|spleef|last|custom` sets the game, its win condition and
      timer, keeping the marks; arena descriptions say what a game still needs, such as a goal
      or a flag for every team; the rules themselves play out with round flow)
- [x] PVP enabled **only** inside minigames or opt-in arenas
      (flags, tag and last-one-standing turn it on, race and spleef off; `/arena pvp ZONE on|off`
      opts any arena in or out; a hit counts only between survival players in reach inside the
      same PVP arena, and takes a heart; the zone greeting says players can tag each other)
- [x] Lobby/portal blocks, scoreboards, round flow
      (admins place a portal block, stand on it and run `/arena portal ZONE`; stepping on it joins
      that arena's round, which counts down 10 seconds once it has a player per team, two for
      fights, else one; players are dealt teams in turn and sent to their spawns; goals win races,
      score points or take and bring home flags; fainting, a tag or walking out sends players back
      to spawn, or out to their portal in fights; the round ends on a win or when time runs out;
      a scoreboard in the corner shows status, clock, points and flags; anyone can `/leave`)

### Phase 6 — Kid-Friendly Scripting
- [x] Blockly (visual blocks) in the browser, generating Lua
      (a lazy chunk loaded when the script editor opens; "when" blocks for each event plus place, give,
      teleport, points and wait blocks, placed relative to `here`; the blocks ride along in a trailing Lua
      comment so scripts reopen as blocks; Lua edits switch the script to text)
- [x] Sandboxed `gopher-lua` on the server: CPU/memory/instruction limits, a whitelisted API
      (`on_touch`, `say`, `set_block`, `give_item`, `teleport`, timers)
      (`internal/script`: base, string, table and math only; 10 ms per run, and a script that overruns
      stops; 64 world calls per run, 16 waiting timers; no `string.rep` and capped format widths)
- [x] Script blocks attach to logic blocks and minigames; text-mode Lua for older kids/adults
      (a script block runs `on_touch` when used, `on_power` from circuits, and `on_round_start` /
      `on_round_end` inside an arena, where `add_points` scores; a text editor opens on R / 📜 / Back;
      scripts act for whoever saved them, within 16 blocks)

### Phase 7 — Adventure
- [x] Quests (data + scripts), NPCs with dialogue
      (a script calling `npc("Granny Ivy")` stands a character on its block, saved with the script so it
      stays after restarts; using it runs `on_talk(player)`; `talk` and `ask(player, text, choices, fn)`
      open a dialogue box and `ask` runs fn with the answer; `remember`/`recall` keep strings, numbers
      and booleans per player for any script (100 each); `quest`/`finish_quest` fill a quest log (Q or the
      Quests button, 50 entries) that comes in the Welcome; `take_item` takes blocks, always true in creative
      and admin-only for survival; journals save with players; a Story category in the blocks)
- [ ] Dungeons/structures in world gen, loot, a couple of bosses
- [ ] Family-authored adventure maps using phases 4–6

## Open questions

1. **Public access:** how will family outside the house reach the server: a port forward to
   Caddy, Cloudflare Tunnel, or something else? (Not needed until Phase 1 testing.)

## Resolved

- TLS is optional; a proxy terminates TLS. Plain HTTP works for local testing.
- Resource budget is ≤2 GB RAM and ~2 cores, with near-zero use at idle.
- The world starts bounded and can open up to infinite later.
- Admins approve joins from the web page or in-game.
- The name stays **Scabbage**.
- Minigame arenas live in the main world, inside zones; separate worlds can come later if needed.
