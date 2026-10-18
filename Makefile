# Run targets inside `nix-shell` (or with Go, Node 22, buf, protoc-gen-go and golangci-lint installed).
.PHONY: all gen check test lint build run dev clean

all: check build

client/node_modules: client/package.json client/package-lock.json
	cd client && npm ci
	@touch $@

gen: client/node_modules
	buf lint
	buf generate

test: client/node_modules
	cd server && go test -race ./...
	cd client && npm test

lint: client/node_modules
	buf lint
	test -z "$$(gofmt -l server)" || (gofmt -l server; exit 1)
	cd server && go vet ./... && golangci-lint run ./...
	cd client && npm run lint

check: lint test

build: client/node_modules
	cd client && npm run build
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/scabbage-server ./cmd/scabbage-server

# Serve the built client from the Go server on :8080.
run: build
	./bin/scabbage-server -web-dir client/dist

# Hot-reloading client on :5173 proxying to the Go server on :8080.
dev: client/node_modules
	cd server && go run ./cmd/scabbage-server -origins "localhost:5173,*:5173" & \
	cd client && npm run dev; kill %1

clean:
	rm -rf bin client/dist
