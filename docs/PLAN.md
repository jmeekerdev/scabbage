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
- [ ] Chunk streaming by distance; meshing (greedy) in a Web Worker; texture atlas
- [ ] Player movement: walk/jump/fly toggle, AABB collision, client prediction + reconciliation
- [ ] Block break/place with server validation and a broadcast to everyone
- [ ] Other players visible (simple blocky avatars + name tags, interpolated)
- [ ] Input: pointer-lock KB/M; touch joystick + look-drag + buttons; Gamepad API
- [ ] Text chat + emote wheel
- [ ] PWA manifest + service worker; landscape fullscreen on phones
- [ ] Autosave + nightly world backup snapshot
- **Done when:** all family devices are in one world at once, building together, and the
  world is still there after a server restart.

### Phase 2 — Look & Feel + Personalization
- [ ] Better avatars (rigged blocky characters, animations), sounds, music toggle
- [ ] Ambient occlusion + simple sky/day–night cycle
- [ ] In-browser pixel editor: avatar skins and custom block textures (admin-approved)
- [ ] Larger block palette: glass, light sources, stairs/slabs, colored wool
- [ ] Graphics quality presets (auto-detect weaker phones)

### Phase 3 — Survival
- [ ] Health, damage, respawn; survival vs. creative game modes per area or player
- [ ] Inventory, hotbar, item drops, mining time per block
- [ ] Crafting recipes (data-driven)
- [ ] Mobs: passive animals + night monsters with simple server-side AI
- [ ] Protected zones (spawn, people's houses) so nobody gets griefed

### Phase 4 — Logic Blocks
- [ ] Signal-propagation system (server tick): wire, switch, button, pressure plate, lamp
- [ ] Doors, pistons, timers, counters, teleporters, note blocks
- [ ] Deterministic tests of logic circuits

### Phase 5 — Minigame Maker
- [ ] Mark an area as an arena: spawns, teams, goal, timer, win condition
- [ ] Built-in templates: race, capture-the-flag, tag, spleef, last-one-standing
- [ ] PVP enabled **only** inside minigames or opt-in arenas
- [ ] Lobby/portal blocks, scoreboards, round flow

### Phase 6 — Kid-Friendly Scripting
- [ ] Blockly (visual blocks) in the browser, generating Lua
- [ ] Sandboxed `gopher-lua` on the server: CPU/memory/instruction limits, a whitelisted API
      (`on_touch`, `say`, `set_block`, `give_item`, `teleport`, timers)
- [ ] Script blocks attach to logic blocks and minigames; text-mode Lua for older kids/adults

### Phase 7 — Adventure
- [ ] Quests (data + scripts), NPCs with dialogue
- [ ] Dungeons/structures in world gen, loot, a couple of bosses
- [ ] Family-authored adventure maps using phases 4–6

## Open questions

1. **Public access:** how will family outside the house reach the server: a port forward to
   Caddy, Cloudflare Tunnel, or something else? (Not needed until Phase 1 testing.)
2. **Extra worlds:** should minigame and adventure maps live in the main world, or in separate
   worlds reached through portals? (Decide by Phase 5; the storage layer will support multiple worlds.)

## Resolved

- TLS is optional; a proxy terminates TLS. Plain HTTP works for local testing.
- Resource budget is ≤2 GB RAM and ~2 cores, with near-zero use at idle.
- The world starts bounded and can open up to infinite later.
- Admins approve joins from the web page or in-game.
- The name stays **Scabbage**.
