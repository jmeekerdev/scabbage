# Run targets inside `nix-shell` (or with Go, Node 22, buf, protoc-gen-go and golangci-lint installed).
.PHONY: all gen check test lint build web release run dev clean

all: check build

client/node_modules: client/package.json client/package-lock.json
	cd client && npm ci
	@touch $@

gen: client/node_modules
	buf lint
	buf generate

test: client/node_modules
	bash scripts/next-version_test.sh
	cd server && go test -race ./...
	cd client && npm test

lint: client/node_modules
	buf lint
	test -z "$$(gofmt -l server)" || (gofmt -l server; exit 1)
	cd server && go vet ./... && golangci-lint run ./...
	cd client && npm run lint

check: lint test

VERSION ?= dev
LDFLAGS = -s -w -X main.version=$(VERSION)
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# Builds the client and copies it into the server, which embeds it.
web: client/node_modules
	cd client && npm run build
	find server/internal/web/dist -mindepth 1 ! -name .gitkeep -delete
	cp -r client/dist/. server/internal/web/dist/

# One static binary with the client built in; copy it anywhere and run it.
build: web
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o ../bin/scabbage-server ./cmd/scabbage-server

# Release binaries for every platform plus SHA256SUMS: make release VERSION=v0.1.0
release: web
	rm -rf bin/release
	for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
	  (cd server && CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="$(LDFLAGS)" \
	    -o ../bin/release/scabbage-server-$(VERSION)-$$os-$$arch$$ext ./cmd/scabbage-server) || exit 1; \
	done
	cd bin/release && sha256sum scabbage-server-* > SHA256SUMS

# Serve the built-in client from the Go server on :8080.
run: build
	./bin/scabbage-server

# Hot-reloading client on :5173 proxying to the Go server on :8080.
dev: client/node_modules
	cd server && go run ./cmd/scabbage-server -origins "localhost:5173,*:5173" & \
	cd client && npm run dev; kill %1

clean:
	rm -rf bin client/dist
	find server/internal/web/dist -mindepth 1 ! -name .gitkeep -delete
