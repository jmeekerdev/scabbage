# Scabbage

A family voxel game that runs in the browser. A small group (2–6 players) shares one persistent
world that you host yourself. You can build in it, survive, wire up logic circuits, make
minigames, write scripts with Blockly or Lua, and play adventures.

The server is one small Go binary with SQLite and the client built in. It runs comfortably on a
home server with 2 GB RAM and 2 cores, and does almost nothing while nobody is playing.

![A lake shore with trees, terraced hills and a stone tower](docs/screenshots/world.jpg)

## Features

- **[Building](docs/play/building.md):** a seeded world of small blocks, brushes, glass, lights,
  stairs, slabs, wool, family-painted blocks and an autopilot.
- **[Survival](docs/play/survival.md):** health, digging, crafting, animals and night monsters.
  Zones protect builds, and each zone or player can be creative or survival.
- **[Logic](docs/play/logic.md):** wire, switches, sensors, gates, pistons, doors, timers,
  teleporters and note blocks.
- **[Minigames](docs/play/minigames.md):** arenas with teams, goals and timers, plus race,
  capture-the-flag, tag, spleef and last-one-standing.
- **[Scripting](docs/play/scripting.md):** Blockly for kids and Lua for older players, run in a
  sandbox on the server.
- **[Adventure](docs/play/adventure.md):** characters, quests, dungeons, bosses and shareable
  adventure maps.
- **[Looks](docs/play/looks.md):** each player picks how the world is drawn, from crisp pixels
  to watercolor or felt.
- **[Agents](docs/play/agents.md):** let an AI agent such as Claude Code build as you, over MCP or
  JSON.

![The block menu open over a rainy beach](docs/screenshots/blocks.jpg)

## Quick start

```sh
git clone https://github.com/jmeekerdev/scabbage.git && cd scabbage
docker compose up -d            # pulls the latest released image
docker compose logs scabbage   # shows the one-time admin code
```

Open `http://<host>:8080` and join with the admin code. Then type `/invite` in game to make a
link for each family member, and let them in from the toast when they ask to join.

Without Docker, download `scabbage-server` from
[Releases](https://github.com/jmeekerdev/scabbage/releases) and run it.
[Install](docs/host/install.md) covers HTTPS, settings, updates and backups.

## Docs

- **Playing:** [Getting started](docs/play/getting-started.md) has the controls and menus.
- **Hosting:** [Install](docs/host/install.md), [Running a server](docs/host/admin.md) and
  [Commands](docs/host/commands.md).
- **Developing:** [Development](docs/dev/development.md) and
  [Architecture](docs/dev/architecture.md).

Everything is listed in [docs/](docs/README.md).

## Development

```sh
nix-shell
make dev      # Go server on :8080 and a hot-reloading client on :5173
make check    # formatting, lint, type checks, and Go and client tests
```

Changes are test-first and go through pull requests. Every merge to `main` is released
automatically. See [Development](docs/dev/development.md).

## License

[MIT](LICENSE)
