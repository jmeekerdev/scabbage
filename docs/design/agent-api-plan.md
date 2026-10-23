# Agent API: plan

Any agent (Claude Code, another LLM client, a script) can drive a player's **own avatar**. The
player watches it fly, dig and build in the web UI. Scabbage has no LLM or API key inside: it
offers a documented API, and the agent brings its own brain.

```
 agent ──MCP / JSON──► scabbage-server ──AgentCommand──► player's open tab
   ▲                      │  (token → player)                  │ autopilot drives
   └──── result ──────────┴──────────── AgentReply ◄───────────┘ the avatar
```

## Ground rules

- **Same rules as hands.** Every edit goes through the normal `EditBlock` path: reach, zones,
  creative/survival, bedrock and rate limits. The autopilot only presses buttons a player could.
- **Watchable.** It moves with player physics and paces its edits (about 8 brushes a second,
  well under the server's 30/s). It never teleports, even for admins.
- **The player stays in charge.** Any movement or build key, or `/stop`, cancels the job. `/undo`
  reverses the last job.
- **Idle ≈ 0.** There's no polling while no agent is connected, no new dependencies, and nothing
  runs on the server unless a request arrives.
- **Units are blocks** (3 per meter), the same numbers the 📍 coordinates show.

## Phase 1: autopilot and in-game commands (client only)

There are no protocol changes. The commands run in the chat box:

| Command                         | Does                                                                               |
| ------------------------------- | ---------------------------------------------------------------------------------- |
| `/goto x y z`                   | flies to x,y,z                                                                     |
| `/clear x1 y1 z1 x2 y2 z2`      | breaks every breakable block in the box, top layer first                           |
| `/fill x1 y1 z1 x2 y2 z2 block` | places `block` into the box's empty cells (air, water, plants), bottom layer first |
| `/stop`                         | cancels the job                                                                    |
| `/undo`                         | puts back what the last job changed                                                |

- Coordinates accept `~` and `~n`, relative to the player, so `/clear ~-5 ~ ~-5 ~5 ~10 ~5` works.
  Block names autocomplete from the world's block list.
- Phase 1 is **creative only**. In survival it says that survival support is coming (phase 4).
- Anyone who may build may use it, because it adds no power beyond holding the mouse button.
  Jobs are capped at 32,768 blocks (32³).
- Progress shows as chat notes ("Clearing… 40%") and ends with "Done: 1,240 blocks".

**How it works.**

- **Plan.** A job box is tiled into the largest brush cubes (size 3, 2, 1) that fit inside it, so
  no brush spills over the edge. The cubes are ordered layer by layer in a serpentine path:
  top-down for clear, bottom-up for fill.
- **Route.** An A* search over a one-meter grid of open air finds a way to a spot where the
  eye is in reach of the next cube, so the avatar flies in under tree canopies and around walls.
  It steers yaw toward each waypoint and uses forward, jump and descend. If it makes no progress
  for 0.5 s, it tries going over, then around either side, then back, and searches again. After
  8 tries it skips that cube and says so at the end. The player's normal physics does the
  rest, so the server's motion tracker accepts every step.
- **Schedule.** Once the eye is within `REACH` of a cube, it sends that cube's edit and moves on
  to the next cube still in reach. It flies again only when nothing is left in reach.
- **Undo.** Before each edit it records what each cell held. Undo is a job of its own: it clears
  the cells it placed, then refills the old blocks.

**Files.**

- `client/src/autopilot/job.ts` (+ test): parses the command args (`~`, block names, the size
  cap) into a `Job` and tiles it into ordered cubes.
- `client/src/autopilot/route.ts` (+ test): `Steering.steer(dt, body, target)`, which returns
  `MoveInput` and yaw, and the stuck check with its escape moves.
- `client/src/autopilot/path.ts` (+ test): `flyPath`, the A* search through open air.
- `client/src/autopilot/jobs.ts` (+ test): runs the commands, keeps undo history and writes the
  chat notes.
- `client/src/autopilot/autopilot.ts` (+ test): the state machine
  (`idle → flying → editing → done`). `tick(dt, body, eye)` returns `{ intent, edits }`.
  `cancel()`. Progress callbacks.
- `client/src/autopilot/undo.ts` (+ test): records cell history and makes the undo job.
- `client/src/player/building.ts`: a new `cubeEdit(cube, block, w)`, the raycast-free twin of
  `breakEdit`/`placeEdit`, sharing their cell filters.
- `client/src/main.ts`: adds the autopilot as an intent source in `combineIntents`, sends its
  edits through `sendEdit`, cancels it on player input, and registers the commands.

**Tests.**

- Tiling covers the box exactly, with no overlap or spill.
- Clear runs top-down and fill bottom-up.
- `~` parsing works, unknown blocks are refused, and so are oversized jobs.
- Steering points at the target and tries escape moves when stuck.
- The route search finds gaps in walls and gets in under roofs.
- With real physics, the autopilot flies around a wall to fill a cube behind it.
- The autopilot sends edits only in reach and stops on cancel.
- Undo restores a fill and a clear.
- e2e: `/fill` a 3×3×3 cube on the dev server and check the blocks.

## Phase 2: agent link (server relay, MCP and JSON, docs)

Built; [agents.md](../play/agents.md) is the user guide. Where it differs from this plan:

- The token is shown in a dialog with Copy buttons (`ServerMessage.agent_linked`), not in chat,
  since chat text can't be selected. The client builds the setup lines from its own address.
- There's no ban in the game yet, so tokens are dropped by `/agent deny`, `/agent revoke` and
  deleting the player.
- `inventory` waits for survival (phase 4). `say` is answered by the server, which marks the
  line with 🤖.
- `agent_progress` carries `{ id, state, percent, note }`.
- `place` clears what's in a listed cell before placing, so it can replace blocks, and `"air"`
  breaks one.

**Pairing.**

- `/agent link [label]` creates a token, shows it once in chat to that player only, and shows
  the MCP URL with it. The server keeps only a hash, in a new `agent_tokens` table
  (hash, player_id, label, created_ms, last_used_ms).
- `/agent list` shows the player's tokens, and `/agent revoke <label|all>` removes them.
  Revoking a device or banning the player also drops the player's tokens.
- **Access.** Admins may link by default. Admins switch other players on or off with
  `/agent allow <name>` and `/agent deny <name>`, stored as a new players column
  `agent_allowed`.

**Endpoints** (on the game server, behind the same Caddy/TLS):

- `POST /mcp`: MCP Streamable HTTP (JSON-RPC 2.0: `initialize`, `tools/list`, `tools/call`).
  It is hand-written, about 200 lines of Go, with no SDK dependency.
- `POST /api/agent/v1/{tool}`: the same tools as plain JSON, for scripts and agents without MCP.
- `GET /api/agent/v1/openapi.json`: the spec, generated from the same tool table.
- Every request needs `Authorization: Bearer <token>`, and each token is limited to 5 requests
  a second.

**Tools.**

- **Answered by the server from its own state** (works without a browser round trip):
  - `whereami`: position, facing, mode, zone.
  - `blocks`: names you can place.
  - `look`: block counts and a heightmap for a box, capped.
  - `inventory`.
  - `players`: who's online and where.
  - `status`: the current job.
- **Relayed to the player's tab and run by the phase 1 autopilot:**
  - `goto`, `clear`, `fill`, `stop`, `undo`.
  - `place`: an explicit list of `{x,y,z,block}`, capped at 4,096. This lets an agent design
    anything.
  - `say`: chat as the player, marked 🤖 so everyone knows an agent said it.
  - Job tools return a job id at once. Agents follow up with `status` (or `wait: true` up to
    30 s).
- **If the player has no tab open**, the tools return a clear error: "open Scabbage in a browser
  first".

**Protocol.**

- New `ServerMessage.agent_command { id, tool, json }`.
- New `ClientMessage.agent_reply { id, json }`.
- New `ClientMessage.agent_progress { id, done, total, state }`.
- The server keeps one pending request map per session, with a timeout.

**Files.**

- `proto/scabbage/v1/*.proto`: the messages above.
- `server/internal/agent/`, with tests for each part:
  - `tokens.go`: create, hash and revoke tokens.
  - `tools.go`: the tool table with JSON schemas.
  - `mcp.go`: JSON-RPC.
  - `rest.go`.
  - `openapi.go`.
- `server/internal/gateway/agent.go` (+ test): the relay, the `/agent` chat commands, the
  `agent_allowed` checks and the server-side read-only tools.
- `server/internal/store`: a migration for `agent_tokens` and `players.agent_allowed`.
- `server/cmd/scabbage-server/main.go`: mounts `/mcp` and `/api/agent/v1/`.
- `client/src/autopilot/agent.ts` (+ test): turns `agent_command` into autopilot jobs and
  reports replies and progress.

**Docs.**

- `docs/play/agents.md`: setup (link, then add the MCP server), the tool reference, examples
  such as "build a 9×9 log cabin at my position", and safety notes.
- Ready-to-paste config:
  `claude mcp add --transport http scabbage https://<host>/mcp --header "Authorization: Bearer <token>"`.

**Tests.**

- Token hashing, revocation, and refusal of tokens without `agent_allowed`.
- MCP `initialize`/`tools/list`/`tools/call` golden responses.
- Relay round trip with a fake session, and the timeout.
- Rate limit.
- e2e: link a token, call `fill` over HTTP and see the blocks.

## Phase 3: templates and ghost preview

Built; [Building](../play/building.md#templates) and [agents.md](../play/agents.md) are the user
guides. Where it differs from this plan:

- The first templates are `cabin`, `tower`, `wall`, `floor` and `stairs`. `house` and `bridge`
  can follow. Stairs are full blocks, one up per step, so they don't need turned stair blocks.
- The "always confirm" switch is per admin, for their own agents: `/agent confirm on|off`,
  stored in `players.agent_skip_confirm`. Non-admins' agent jobs always wait, so a switch for
  them would do nothing. The server sends `AgentCommand.confirm`, and the protocol went to 25
  so old tabs, which would ignore it, reload.
- `goto` never waits, and `/undo` doesn't either. Moving doesn't drop a waiting job, so the
  player can fly around the preview; `/no`, `/stop`, `/undo` or a new job does.
- `status` with `wait: true` also returns when the player answers.

- **`build` command and tool.** `/build cabin|house|tower|bridge|wall|floor|stairs [w d h]
[material] [facing]`, placed at the player or at x,y,z. Templates are small TS functions that
  return cells, so agents and players share them.
- **Ghost preview.** Before a job over 500 blocks, or any job from an agent (an admin can set
  this per player), the client shows a translucent mesh of the changes and a "Build it? /yes
  /no" prompt. The agent sees `awaiting_confirmation` in `status`. Agent jobs for players who
  aren't admins always wait for this.
- **Files.**
  - `client/src/autopilot/templates/*.ts` (+ tests checking sizes and doorways).
  - `client/src/render/ghost.ts`: one merged, transparent mesh.
  - A confirm prompt in the chat UI.
- **Tests.** Template cell counts and bounds, facing rotation, preview created and removed,
  jobs held until `/yes`.

## Phase 4: survival

- **Walking routes.** An A* search over standable cells: 6 blocks of headroom, steps up to 1.5
  blocks, jumps up to 3, and no fall drops over 9 blocks. It is bounded to 64 blocks per leg,
  then searches again. The route is followed with the same steering, plus jump.
- **Digging** uses the existing `Miner` and `digMs` times, so the avatar works at the same
  speed as a player.
- **Materials.** Before `fill`/`place`/`build`, it counts the blocks needed against the
  inventory and offers a partial build or refuses ("needs 140 planks, you have 64"). The edits
  spend the hotbar slot as usual. Clearing picks up the drops.
- **Safety.** It pauses when health is low or a mob is near, and reports it in `status`.
- **Files.**
  - Walking routes in `client/src/autopilot/path.ts`, next to the flying search (+ tests on
    small fixed worlds: stairs, gaps, walls).
  - Survival branches in `autopilot.ts`.
  - A materials check in `job.ts`.

## Phase 5: plans

Built; [agents.md](../play/agents.md#plans) and [Building](../play/building.md#plans) are the
user guides. Where it differs from this plan:

- A palette entry names a block only. `material` swaps the block the plan uses most, so there
  are no template parts to name.
- Rows read left to right as seen from the front. The size comes from the layers, so a plan
  carries no `w d h`.
- `propose` answers like any job, with its job id, not a separate proposal id. `save_plan` takes
  just an optional `name` and saves the last plan proposed in the tab.
- The `/no` reason comes back in `status` as the `note` ("The player said no: …").
- Added after trying it: `chat` takes `after` and `wait` to long-poll for the player's answer,
  `/ai MESSAGE` sends a private message to the player's own agent, which loops on a new `listen`
  tool, and `say` takes `private: true`.
- `propose` and `build` reach the tab as `build` with the whole plan in its arguments, so
  `AgentCommand` needed no change. Protocol 27 adds only `PlanRequest` and `Plan`.

Agents design new buildings, show them as a ghost, revise them from the player's answers, and
save them as **plans** that `/build NAME` can build later, like the built-in templates.

- **The format.** A plan has:
  - a `name` and an `about` line
  - its size `w d h`, each side at most 32
  - a `palette` of one-character keys, such as `{"p": "planks", "g": "glass"}`, where `.`
    means "leave alone" and `_` means air
  - its `layers`, bottom first, each a list of `d` rows (front row first) of `w` characters

  The coordinates are the plan's own: x across the front, y up, z back from the front. That
  makes plans turn and place the same way templates do. A palette entry may name a template part
  instead of a block (`"p": "main"`), so `material` swaps work as they do for templates.

- **Who sees what.**
  - A saved plan is its owner's.
  - The owner can share it with `/plan share NAME` and take it back with `/plan unshare NAME`.
    No admin approval is needed.
  - A shared name must not match a built-in template or another shared plan. Sharing refuses
    such a name and says to rename the plan first.
  - Private names only need to be unique for their owner.
  - Admins can delete any shared plan.
  - `/build NAME` looks in this order: the built-in templates, the player's own plans, then
    shared plans.
  - A player can have at most 50 plans, each at most 32,768 cells.
- **Agent tools.**
  - **`propose`** takes `plan` (a whole plan, or the name of a saved one), `x y z` and
    `facing?`. It shows the ghost, always waiting for the player, even with `/agent confirm
off`. It answers with a proposal `id`. A new `propose` replaces the ghost, so the agent can
    revise without starting over. `/yes` builds it.
  - **`/no [reason]`**: the reason comes back in `status` as `answer`.
  - **`chat`**: recent chat lines the player can see, so the agent hears "make the roof
    steeper". It reads only; the agent already talks with `say`.
  - **`save_plan`** takes a proposal `id` and a `name`, and saves the plan the player has seen.
    It doesn't ask again, because the plan is private until the player shares it.
  - **`plans`** lists the player's own plans and the shared ones, with sizes and owners.
    **`plan`** returns one in full, so an agent can start from it.
  - **`build`** also takes plan names.
- **In game.**
  - `/plans` lists them.
  - `/plan save NAME` saves the ghost on screen.
  - `/plan share`, `/plan unshare`, `/plan rename OLD NEW` and `/plan delete NAME`.
- **Server.** The `plan` package checks the format, the limits and the names. It stores plans
  in a `plans` table (owner, name, shared, the plan as JSON, created). Proposals are held in
  memory per player, and only the latest one is kept.
- **Protocol 27.**
  - `AgentCommand` can carry a plan.
  - The client asks for a saved plan by name for `/build` (`PlanRequest`), and the server
    answers with `Plan`.
  - Plans aren't sent at login.
- **Files.**
  - `server/internal/plan/` (+ tests): the format, the checks and the queries.
  - `server/internal/gateway/plans.go`: the `/plan` commands, `PlanRequest`, and the tools.
  - `client/src/autopilot/templates/plan.ts`: turns a plan into cells that `placeTemplate`
    places.
- **Tests.**
  - Parsing and refusing bad plans: wrong row lengths, unknown keys, too big.
  - The name rules for sharing.
  - Rotation matching the built-in templates.
  - A rejected proposal's reason reaching `status`.
  - `/build` finding a shared plan.

## Order and size

| Phase                  | Rough size                     | Ships as                             |
| ---------------------- | ------------------------------ | ------------------------------------ |
| 1 Autopilot + commands | ~600 lines TS + tests          | one PR                               |
| 2 Agent link           | ~900 lines Go + ~200 TS + docs | one or two PRs (server, then client) |
| 3 Templates + preview  | ~500 lines TS                  | one PR                               |
| 4 Survival             | ~500 lines TS                  | one PR                               |
| 5 Plans                | ~600 lines Go + ~250 TS + docs | one PR                               |

Each phase is usable on its own. Phase 1 alone gives players `/fill` and `/clear`.
