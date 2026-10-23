# Architecture

```
  Browser                                      Server (one Go binary)
 ┌────────────────────────────┐   WebSocket   ┌───────────────────────────────────┐
 │ TypeScript + Three.js      │  /ws, binary  │ gateway: sessions, rules, routing │
 │  net ─ world ─ render      │◄────────────► │  sim (tick)  world  logic  mobs   │
 │  player  ui  autopilot     │   Protobuf    │  script (Lua)  rounds  zones  …   │
 │  mesh workers              │               │ store: SQLite (WAL)               │
 └────────────────────────────┘               │ web: embedded client  agent: MCP  │
                                              └───────────────────────────────────┘
```

The server is authoritative: it checks every edit, move, hit and script call. The client
predicts movement and building so they feel instant, and it draws the world.

## Server

`server/cmd/scabbage-server` wires everything up. The packages in `server/internal/`:

| Package                                                        | Does                                                                                                             |
| -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `gateway`                                                      | accepts WebSockets, admits players and routes every message. Most game rules meet here                           |
| `sim`                                                          | the tick loop. It runs only while a player is in the world                                                       |
| `world`                                                        | chunks, terrain generation, the border, chunk storage and the block table                                        |
| `store`                                                        | opens SQLite and runs its migrations                                                                             |
| `access`                                                       | device secrets, join codes and approving join requests                                                           |
| `edit`                                                         | which breaks and placements are allowed: reach, brushes, digging time                                            |
| `zones`                                                        | protected areas and who may build in them                                                                        |
| `logic`                                                        | circuits: wire networks, sources, gates and machines                                                             |
| `mobs`                                                         | sheep, slimes and bosses                                                                                         |
| `vitals`                                                       | health, falls and fainting                                                                                       |
| `inventory`, `crafting`, `drops`, `loot`                       | survival items                                                                                                   |
| `arenas`, `rounds`                                             | minigame setups and the rounds played in them                                                                    |
| `script`, `scripts`                                            | the Lua sandbox, and storing and loading the scripts that blocks run                                             |
| `journal`                                                      | per-player script memories and quests                                                                            |
| `maps`                                                         | adventure maps: a zone's blocks and scripts, saved so they can be put back or shared                             |
| `art`                                                          | painted skins and blocks, and their approval                                                                     |
| `agent`                                                        | agent tokens, the tool table, and the MCP and JSON servers                                                       |
| `chat`, `rate`, `motion`, `weather`, `backup`, `config`, `web` | chat text and commands, rate limits, move checks, weather from the clock, backups, settings, the embedded client |
| `pb`                                                           | generated Protobuf code                                                                                          |

### A session

1. The browser opens `/ws`. The origin must match the host or `-origins`.
2. It sends `Hello` with the protocol version and its device secret. A version mismatch gets a
   `Disconnect`, and the player sees "Update needed" with a reload button.
3. `admit` checks the secret, or takes a join code. A pending player waits on the open socket
   until an admin decides.
4. The server sends `Welcome` (world info, vitals, inventory, position and quests). It adds the
   player to the world and joins the hub, which starts the tick if they're the first.
5. `serve` reads messages and writes the outbound queue. Only one goroutine writes.
6. On close, the player is removed and saved. When the last one leaves, the world is saved and
   unloaded.

The admin page connects in console mode: it's a session without an avatar that doesn't wake
the world.

### A tick

20 times a second by default (`roster.go`):

1. Step the world: healing, picking up drops, mobs, rounds.
2. Broadcast `PlayersMoved`.
3. Save what changed.
4. Step circuits: scheduled waits, timers, sensors, doors, teleporters.

Autosave and dropping far-off chunks run once a minute.

### Storage

SQLite in WAL mode, through the pure-Go `modernc.org/sqlite` (no CGO), with one connection.
Migrations in `store/store.go` only add. `PRAGMA user_version` tracks them.

- **Chunks** are 32³ blocks, keyed by `(world_id, x, y, z)`. Only changed chunks are stored;
  the rest are generated from the seed. A chunk is a palette plus run lengths, and the same bytes
  are stored and sent.
- Other tables: players, devices, join codes and requests, worlds, art, zones and builders,
  arenas, scripts, memories, quests, maps and agent tokens.

### Agents

`/mcp` and `/api/agent/v1/` share one tool table (`agent/tools.go`). A tool call finds the
token owner's open tab and relays an `AgentCommand` to it. The tab runs the job with its
autopilot, under the same rules as the player's own hands, and answers with `AgentReply` and
`AgentProgress`. The plan is [design/agent-api-plan.md](../design/agent-api-plan.md).

## Protocol

[`proto/scabbage/v1/net.proto`](../../proto/scabbage/v1/net.proto) is the source of truth. Each
WebSocket frame is one `ClientMessage` or `ServerMessage`, a `oneof` of every message. The
socket uses permessage-deflate with no context takeover, so idle players hold no compressor
memory.

| Group             | Messages                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Joining           | `Hello`, `Welcome`, `JoinPending`, `Disconnect`, `Ping`/`Pong`                                                                 |
| World             | `RequestChunks`/`ChunkBatch` (up to 64), `RequestFar`/`FarLand` (distant land), `ChunksStale`, `BlocksChanged`, `ZonesChanged` |
| Building          | `EditBlock`, `UseBlock`, `BlockChanged`, `BlockBatch`                                                                          |
| Players           | `Move`, `Teleport`, `PlayersJoined`/`Moved`/`Left`, `SetAway`/`PlayerAway`, `SkinChanged`                                      |
| Social            | `SendChat`/`Chat`, `SendEmote`/`EmoteShown`                                                                                    |
| Survival          | `Vitals`, `Inventory`, `MoveItem`, `Craft`, `ItemsChanged`, `MobsChanged`, `HitMob`, `HitPlayer`                               |
| Minigames         | `Scoreboard`                                                                                                                   |
| Scripts and story | `OpenScript`/`SaveScript`/`ScriptShown`, `NpcsChanged`, `TalkTo`, `Dialogue`/`Answer`, `QuestLog`                              |
| Admin             | `PendingRequests`, `DecideJoin`, `PlayerList`, `SetAdmin`, `CreateJoinCode`, `SetGameMode`, `ArtRequests`/`DecideArt`          |
| Maps              | `ImportMap`/`MapExported` ([`map.proto`](../../proto/scabbage/v1/map.proto) is the file format)                                |
| Agents            | `AgentCommand`, `AgentReply`, `AgentProgress`, `AgentLinked`                                                                   |

Bump `ProtocolVersion.CURRENT` for any change old clients can't handle. Old tabs are told to
reload. Tabs also reload by themselves when a reconnect finds a newer build.

## Client

`client/src/main.ts` boots the game. The modules:

| Folder                                               | Does                                                                                                                                                   |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `net/`                                               | the socket (ping every 2 s, reconnect after 3 s), encoding and decoding, latency, noticing a stale build                                               |
| `world/`                                             | the chunk store, streaming (nearest first), meshing, light, ray casts, collision, shapes, far land                                                     |
| `render/`                                            | the scene and frame pacer, chunk materials, the texture atlas, procedural textures, look shaders and after-effects, sky, weather, avatars, mobs, drops |
| `looks/`                                             | one file per look, plus the registry and range checks                                                                                                  |
| `player/`                                            | physics, input (keyboard and mouse, gamepad, touch), building and digging, inventory, other players (drawn 2 ticks behind)                             |
| `autopilot/`                                         | `/goto`, `/fill` and `/clear` jobs, path finding, undo, and running agent commands                                                                     |
| `ui/`                                                | DOM views: chat, hotbar, backpack, settings, the pixel and script editors (Blockly loads lazily), dialogue, quests, scoreboard, admin cards            |
| `join/`, `settings/`, `social/`, `editor/`, `audio/` | identity and invites, settings and auto quality, emotes, pixel helpers, sound                                                                          |
| `pb/`                                                | generated Protobuf code                                                                                                                                |

### Drawing

- **Meshing** runs in Web Workers: two if the device has 4 or more cores, otherwise one.
- **The frame pacer** draws at full speed for a second after the view changes. Then it drops to
  30 fps while anything moves, and otherwise checks for changes ten times a second. Hidden and
  paused tabs draw nothing.
- **Looks.** A look's numbers are shared uniforms, so changing them is instant. Its features are
  shader defines, so turning one on or off recompiles. Changing the mesh shape (rounded edges or
  soft corners) fades the world into fog and remeshes every chunk.

### Sound

All sound is synthesized with Web Audio: effects, note blocks and a quiet background song. There
are no audio files. The audio context suspends after 3 s of silence, and whenever the tab is
hidden or another window has focus.

## Design

[design/PLAN.md](../design/PLAN.md) has the decisions behind all this and the plan phase by
phase.
