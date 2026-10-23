# Development

## Set up

The tools come from [`shell.nix`](../../shell.nix): Go, Node 22, buf, protoc-gen-go,
golangci-lint, glslang and gcc (for `-race`). Without Nix, install those yourself.

```sh
nix-shell
make dev      # Go server on :8080 and a hot-reloading client on :5173
```

Open `http://localhost:5173`. The server logs an admin code on first start. Its world lives in
`server/data/`.

| Target                        | Does                                                                     |
| ----------------------------- | ------------------------------------------------------------------------ |
| `make dev`                    | Go server plus the Vite dev server, which proxies `/ws`                  |
| `make check`                  | everything CI runs: `lint` then `test`                                   |
| `make lint`                   | buf lint, gofmt, go vet, golangci-lint, tsc, eslint, prettier            |
| `make test`                   | Go tests with `-race`, client tests (vitest), the version script's tests |
| `make gen`                    | lint the protos and regenerate Go and TypeScript code                    |
| `make build`                  | one static binary with the client inside: `bin/scabbage-server`          |
| `make run`                    | build, then run it on :8080                                              |
| `make release VERSION=v0.1.0` | binaries for six platforms plus `SHA256SUMS`                             |
| `make clean`                  | remove build output                                                      |

## How we work

- **Test first.** Every change comes with tests, and `make check` must pass.
- **Pull requests.** Changes go through PRs, and CI runs `make check` and a Docker build on each.
- **Lean.** The server targets 2 GB and 2 cores and should do nearly nothing while idle. Think
  twice before adding a dependency.
- **US English** in code, comments, docs and the UI.

## Tests

- **Go** tests sit beside their code. Gateway tests start a real server with
  `startServer(t)` (`gateway/gateway_test.go`): a temporary database, a small seeded world,
  and helpers like `joinAdmin`, `send`, `recvWhere` and `waitPlayers`. Tests can drive the tick
  by hand by injecting a `sim.Ticker`.
- **Client** tests are vitest `*.test.ts` files beside each module. Shader tests compile every
  look with glslang. If tests see stale files, clear `client/node_modules/.vitest-cache`.

## Changing the protocol

1. Edit [`proto/scabbage/v1/net.proto`](../../proto/scabbage/v1/net.proto).
2. Run `make gen`. The generated code in `server/internal/pb` and `client/src/pb` is committed.
3. Bump `ProtocolVersion.CURRENT` if old clients can't handle the change.

## Adding a block

Blocks are defined in `server/internal/world/block.go`. The client gets the block table in
`Welcome`, and draws the textures in code in `client/src/render/` (`textures.ts`,
`machine-textures.ts`, `device-textures.ts`, `treasure-textures.ts`). Circuit behavior is in `server/internal/logic`. Survival recipes are in
`server/internal/crafting/recipes.json`.

## Looks

Each file in `client/src/looks/` named `*.look.ts` is a look, and its file name is its id. To
make one:

1. Open the game with `?tune` on the address (`http://localhost:5173/?tune`) for sliders over
   every look setting.
2. Press **Export** to get a new `.look.ts` file and save it in `client/src/looks/`.
3. `make check` validates its ranges and compiles its shaders.

## Releases

Every merge to `main` is released as the next patch version:

- GitHub release binaries for Linux, macOS and Windows on amd64 and arm64, with notes;
- the image `ghcr.io/jmeekerdev/scabbage` for amd64 and arm64, tagged with the version and
  `latest`.

The major and minor numbers come from [`VERSION`](../../VERSION). Change it, say to `0.2`, to
start the next series at `v0.2.0`. `scripts/next-version.sh` works out the next tag.
