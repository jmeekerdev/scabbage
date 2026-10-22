# Build context is the repo root: docker build -f docker/Dockerfile .
# The build stages run on the builder's own platform and cross-compile, so multi-arch builds skip emulation.
FROM --platform=$BUILDPLATFORM node:22-alpine AS client
WORKDIR /src/client
COPY client/package.json client/package-lock.json ./
RUN npm ci
COPY client/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=client /src/client/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" -o /out/scabbage-server ./cmd/scabbage-server \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.source=https://github.com/jmeekerdev/scabbage \
      org.opencontainers.image.description="Scabbage, a family voxel game server" \
      org.opencontainers.image.licenses=MIT
COPY --from=server /out/scabbage-server /scabbage-server
COPY --from=server --chown=nonroot:nonroot /out/data /data
ENV SCABBAGE_ADDR=:8080 SCABBAGE_DATA_DIR=/data
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/scabbage-server"]
