# Agent API

An agent (Claude Code, another MCP client, or a script) can fly and build as **your own
avatar** while you watch in the browser. Scabbage has no AI inside it: it offers the tools, and
the agent brings its own brain.

## Set up

1. In game, type `/agent link` (or `/agent link laptop` to name it). A dialog shows a token
   once, with ready-to-copy setup.
2. For Claude Code, paste the command it shows:

   ```sh
   claude mcp add --transport http scabbage https://your.server/mcp \
     --header "Authorization: Bearer scab_…"
   ```

   Other MCP clients use the URL `https://your.server/mcp` with the same header. Agents without
   MCP can use plain HTTP: tell them to start by reading `https://your.server/api/agent/v1/`,
   which explains the API and lists the tools.

3. Keep the Scabbage tab open and visible. The agent drives that tab's avatar, and browsers
   pause hidden tabs. The game pauses itself and goes quiet when another window has focus,
   but it keeps running while an agent's job is under way.

Admins can link agents right away. Other players need an admin to type `/agent allow NAME`
first. `/agent deny NAME` turns agents off again and revokes that player's tokens.

| Command                           | Does                                                                         |
| --------------------------------- | ---------------------------------------------------------------------------- |
| `/agent link [label]`             | makes a token (up to 5 per player; linking a label again replaces its token) |
| `/agent list`                     | shows your linked agents and when each was last used                         |
| `/agent revoke LABEL` / `all`     | turns a token off                                                            |
| `/agent allow NAME` / `deny NAME` | admins: turn agents on or off for a player                                   |
| `/agent confirm on` / `off`       | admins: whether your own agents' jobs wait for your `/yes` (on at first)     |

## MCP or plain HTTP?

Both reach the same tools, and agents are told this when they connect (MCP's instructions, and
the guide at `/api/agent/v1/`):

- **MCP** suits building by conversation. The tools show up as native tool calls, the agent gets
  the instructions on connecting, and the token stays in the config.
- **HTTP** suits big generated designs. A short script can compute thousands of cells and POST
  them to `place` in batches of up to 4,096, instead of the agent writing every cell into a tool
  call.

To use MCP without saving it to your Claude Code config, put the server in a file and load it
for one session:

```sh
echo '{"mcpServers":{"scabbage":{"type":"http","url":"https://your.server/mcp",
  "headers":{"Authorization":"Bearer scab_…"}}}}' > scabbage-mcp.json
claude --mcp-config scabbage-mcp.json
```

## Rules

- **Same rules as your hands.** Every edit goes through the normal building path: reach,
  zones, bedrock and the edit rate limit. The avatar flies there with normal physics and never
  teleports.
- **You say yes first.** Before an agent's job changes a block, you see a see-through preview
  (blue for blocks it builds, red for blocks it breaks) and chat asks you to say `/yes` or
  `/no`. Jobs for players who aren't admins always wait like this. An admin can let their own
  agents build without asking with `/agent confirm off`. Flying with `goto` never asks.
- **You stay in charge.** Moving or building yourself, or `/stop`, stops the agent's job.
  `/undo` takes back the last job. Jobs an agent starts show 🤖 in chat, and so do its lines
  from `say`.
- **Players who aren't admins** fly and build just like their own hands would: in creative
  everyone can fly, and a job that reaches into a zone they can't build in is refused up front,
  naming the zone.
- **Creative only for now.** In survival, jobs are refused; survival support is planned.
- **Limits.** 5 calls a second per token. Boxes up to 32,768 blocks (a 32-block cube) and
  `place` lists up to 4,096 cells.

## Units

Coordinates are whole blocks, the same numbers as the 📍 line in game. There are 3 blocks to a
meter, the avatar is about 5 blocks tall, and y is up. `whereami` gives the avatar's feet.

## Tools

| Tool       | Arguments                                               | Returns                                                                                                                                                                                        |
| ---------- | ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `whereami` |                                                         | `x y z` (feet), `block` `[x,y,z]`, `facing`, `flying`, `mode`, `zone`                                                                                                                          |
| `blocks`   |                                                         | `blocks`: each block `fill` and `place` take, with `solid`, `see_through` (glass), `glows` (lamps), `shape` when not a cube (`slab`, `stairs_east`, `torch`, …) and `usable` (switches, doors) |
| `look`     | `from`, `to`                                            | `counts` by block name; `heights` and `tops`: per column (rows z, columns x), the top non-air block's y and name, or null                                                                      |
| `players`  |                                                         | `players`: who is online, where, their mode, and `away` when their game is paused                                                                                                              |
| `status`   | `wait?`                                                 | the last job: `job`, `tool`, `state` (`awaiting_confirmation`, `running`, `done`, `stopped`), `percent`, `note`; `wait: true` waits up to 20 s for it to end, or for the player to answer      |
| `say`      | `text`                                                  | chats as you, marked 🤖                                                                                                                                                                        |
| `goto`     | `x y z`                                                 | flies there; starts a job                                                                                                                                                                      |
| `clear`    | `from`, `to`                                            | breaks every breakable block in the box, top layer first; starts a job                                                                                                                         |
| `fill`     | `from`, `to`, `block`                                   | fills the box's empty cells (air, water, plants), bottom layer first; `"air"` clears the box instead, e.g. for a doorway; starts a job                                                         |
| `place`    | `blocks: [{x, y, z, block}]`                            | sets each cell to its block (`"air"` breaks it), clearing what's in the way; starts a job                                                                                                      |
| `build`    | `template`, `x y z`, `w? d? h?`, `material?`, `facing?` | builds a [template](building.md#templates) with its front's bottom middle at `x y z`, facing `north`, `east`, `south` or `west` (default `south`), the rest behind it; starts a job            |
| `stop`     |                                                         | stops the job                                                                                                                                                                                  |
| `undo`     |                                                         | puts back what the last job changed; starts a job                                                                                                                                              |

`from` and `to` are `{x, y, z}` corners, both included. A job tool answers at once with
`{"job": id, "started": "…", "brushes": n, "seconds": s}`, plus `"awaiting": true` when it waits
for the player's `/yes`. Follow up with `status` and `wait: true` until `state` is `done` or
`stopped`. A good agent tells the player in chat what it's about to build, so they know to
answer. One job runs at a time, and a new one stops the last. The avatar flies to the work by
itself: it searches for a way through, and when there's none, it rises above everything on the
way, crosses and drops down.

Like a player, the avatar edits with a brush: a cube 3, 2 or 1 blocks on a side, about 8
brushes a second. Jobs pick the biggest brushes that fit, so one big `fill` is much quicker
than many small calls, and boxes whose sides are multiples of 3 go fastest (a 9×9×3 wall is 9
brushes). Job tools answer with `brushes` and `seconds`, the time spent editing once there.

A `place` batch is all or nothing. If any cell is wrong, nothing is placed, and the error lists
the cells by index with what's wrong with each, so the agent can fix just those and resend.

`air` isn't in the `blocks` list but works in `fill` and `place`. An unknown block name is
refused with close names, for example "There's no block named wooden_stairs. Close names:
planks_stairs, …".

Errors come back as MCP tool errors (`isError`), or as `{"error": "…"}` from the JSON API. They
say what to fix, for example "open Scabbage in a browser first", "There's no block named lava."
or "slow down".

## Plain JSON

`GET /api/agent/v1/` explains the API and lists every tool with its arguments; no token needed.
Every tool is also `POST /api/agent/v1/{tool}` with the arguments as the JSON body and the
same `Authorization` header. The OpenAPI spec is at `GET /api/agent/v1/openapi.json`.

```sh
curl -s https://your.server/api/agent/v1/whereami -H "Authorization: Bearer $TOKEN" -d '{}'
curl -s https://your.server/api/agent/v1/fill -H "Authorization: Bearer $TOKEN" \
  -d '{"from":{"x":10,"y":64,"z":10},"to":{"x":18,"y":64,"z":18},"block":"planks"}'
curl -s https://your.server/api/agent/v1/status -H "Authorization: Bearer $TOKEN" -d '{"wait":true}'
```

## Example prompts

- "Build a 9×9 log cabin with a door and glass windows next to me."
- "Clear the trees in a 20-block square around me, then lay a planks floor."
- "Look at the hill to the north and carve stairs up it."

Agents usually start with `whereami` and `blocks`, use `look` to see the ground, and build
walls and floors with `fill` and the details with `place`.

## Safety

- A token acts as you. Keep it secret, and revoke it with `/agent revoke` if it leaks.
  The server keeps only a hash of each token.
- Turning agents off for a player (`/agent deny`) or deleting the player revokes their tokens.
- Nothing runs while no agent is calling. The server doesn't poll, and with no tab open every
  tool but `blocks` and `players` says to open the game first.
