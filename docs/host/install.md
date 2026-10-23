# Install and run a server

Scabbage is one Go binary with SQLite and the game built in. It runs comfortably on a home
server with 2 GB of RAM and 2 cores, and does almost nothing while nobody is playing.

## Docker

```sh
git clone https://github.com/jmeekerdev/scabbage.git && cd scabbage
docker compose up -d            # pulls the latest released image
docker compose logs scabbage   # shows the one-time admin code
```

Open `http://<host>:8080` and join with the admin code to become the first admin. Then see
[Running a server](admin.md) to invite your family.

The image is `ghcr.io/jmeekerdev/scabbage`, for amd64 and arm64. It runs as a non-root user
(uid 65532) in a distroless container, with the world in the `scabbage-data` volume.

### HTTPS

HTTPS is optional but recommended for play away from home. Browsers need it to install the game
as an app and to use gamepads, and the admin page warns without it. Point a domain at the
machine, open ports 80 and 443, and run:

```sh
SCABBAGE_DOMAIN=game.example.org docker compose --profile tls up -d
```

This adds Caddy, which fetches a certificate and proxies to the game. Put `SCABBAGE_DOMAIN` in
`.env` to keep it. Set `SCABBAGE_PORT=127.0.0.1:8080` to stop plain HTTP from reaching the game
from other machines.

With your own proxy or tunnel, forward `/` to port 8080, including the WebSocket at `/ws`. If
the address the browser sees differs from the host the server sees, add it to
`SCABBAGE_ORIGINS`. The server trusts `X-Forwarded-For` only from loopback and private
addresses.

### Updating

```sh
docker compose pull && docker compose up -d
```

Every merge to `main` is released as the next version. To stay on one release, set
`SCABBAGE_VERSION` (such as `v0.1.3`) in `.env`. To run your own changes, use
`docker compose up -d --build`.

## Without Docker

Download `scabbage-server` for Linux, macOS or Windows (amd64 or arm64) from
[Releases](https://github.com/jmeekerdev/scabbage/releases), check it against `SHA256SUMS`, and
run it. It keeps the world in `./data` and serves the game on port 8080. Stop it with Ctrl-C or
SIGTERM, and it saves the world first.

## Settings

With Docker, copy [`.env.example`](../../.env.example) to `.env`, edit it, and run
`docker compose up -d` again. Every setting is also a flag (`scabbage-server -help`). A flag
wins over its environment variable.

| Flag           | Variable               | Default    | Meaning                                                                                 |
| -------------- | ---------------------- | ---------- | --------------------------------------------------------------------------------------- |
| `-name`        | `SCABBAGE_NAME`        | `Scabbage` | name shown in the game and browser tab                                                  |
| `-mode`        | `SCABBAGE_MODE`        | `creative` | `creative` or `survival`, for players without their own mode                            |
| `-spawn-zone`  | `SCABBAGE_SPAWN_ZONE`  | `48`       | half-width in blocks of the admins-only zone around spawn, up to 1024; `0` turns it off |
| `-tick-rate`   | `SCABBAGE_TICK_RATE`   | `20`       | simulation ticks a second, 1–120. 20 is the tested rate                                 |
| `-backups`     | `SCABBAGE_BACKUPS`     | `7`        | nightly backups to keep; `0` turns them off                                             |
| `-backup-hour` | `SCABBAGE_BACKUP_HOUR` | `4`        | hour of the day (0–23) backups run, in `TZ`                                             |
| `-origins`     | `SCABBAGE_ORIGINS`     |            | extra hosts allowed to open the game connection, comma-separated                        |
| `-addr`        | `SCABBAGE_ADDR`        | `:8080`    | listen address (fixed in Docker; use `SCABBAGE_PORT`)                                   |
| `-data-dir`    | `SCABBAGE_DATA_DIR`    | `data`     | where the world lives (fixed in Docker; use `SCABBAGE_DATA`)                            |
| `-web-dir`     | `SCABBAGE_WEB_DIR`     |            | serve the client from this folder instead of the built-in one                           |

Docker-only settings:

| Variable                                    | Default         | Meaning                                                             |
| ------------------------------------------- | --------------- | ------------------------------------------------------------------- |
| `SCABBAGE_VERSION`                          | `latest`        | image tag to run                                                    |
| `SCABBAGE_PORT`                             | `8080`          | host port for plain HTTP                                            |
| `SCABBAGE_DATA`                             | `scabbage-data` | a volume name, or a host folder (`sudo chown 65532:65532` it first) |
| `SCABBAGE_DOMAIN`                           |                 | domain for the `tls` profile                                        |
| `SCABBAGE_HTTP_PORT`, `SCABBAGE_HTTPS_PORT` | `80`, `443`     | Caddy's ports                                                       |
| `SCABBAGE_MEMORY`, `SCABBAGE_CPUS`          | `2g`, `2`       | container limits                                                    |
| `GOMEMLIMIT`                                | `1800MiB`       | keep it about 200 MiB under the memory limit                        |
| `TZ`                                        | `UTC`           | time zone, such as `America/Denver`                                 |

### Recipes

To change the crafting recipes, put a `recipes.json` in the data folder and restart. It
replaces the built-in book, [`server/internal/crafting/recipes.json`](../../server/internal/crafting/recipes.json),
which is a good place to start. Each recipe looks like
`{"make": "planks", "count": 4, "from": {"log": 1}}`.

## Data and backups

The data folder holds:

| File                                  | What                                                                                             |
| ------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `scabbage.db`                         | the whole world and every player, in SQLite. `-wal` and `-shm` files sit beside it while it runs |
| `backups/scabbage-YYYYMMDD-HHMMSS.db` | nightly backups; the newest `SCABBAGE_BACKUPS` are kept                                          |
| `backups/before-wipe-….db`            | taken before `/world reset` or `/world new`; never pruned                                        |
| `recipes.json`                        | optional recipe book                                                                             |

Only land that has been changed is stored. The rest grows back from the world's seed.

Each backup is a complete, consistent copy of the database. To restore one:

1. Stop the server (`docker compose stop scabbage`).
2. Copy the backup over `scabbage.db`, and delete `scabbage.db-wal` and `scabbage.db-shm`.
3. Start it again.

With the Docker volume, reach the files with a helper container, for example:

```sh
docker run --rm -v scabbage-data:/data alpine sh -c \
  'cp /data/backups/scabbage-20261022-040000.db /data/scabbage.db && rm -f /data/scabbage.db-wal /data/scabbage.db-shm'
```

A newer server can open an older database, but not the other way round.

## Endpoints

| Path                     | What                                      |
| ------------------------ | ----------------------------------------- |
| `/`                      | the game                                  |
| `/admin.html`            | the [admin page](admin.md#the-admin-page) |
| `/ws`                    | the game's WebSocket                      |
| `/mcp`, `/api/agent/v1/` | the [agent API](../play/agents.md)        |
| `/healthz`               | answers `ok`, for health checks           |

## Logs and idle

Logs go to stderr (`docker compose logs scabbage`): startup, players joining and leaving,
backups, admin changes and the admin code while there's no admin.

The simulation only runs while someone is in the world. When the last player leaves, the server
saves, unloads the world and frees its memory. While players are on, it saves every minute and
drops land nobody is near.

## Locked out?

There's no password to reset: each device holds a secret key in its browser storage. If every
admin has lost their device:

1. Stop the server and open `scabbage.db` with `sqlite3`.
2. Run `UPDATE players SET admin = 0;`.
3. Start the server. With no admin left, it logs a new admin code again.
4. Join with that code under your old name. This device is linked to your old player, which
   becomes an admin again. Make the others admins again with `/admin NAME`.
