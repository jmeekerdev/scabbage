import { defineConfig } from "vite";

// Dev server proxies the game socket to the Go server on :8080.
export default defineConfig({
  server: {
    host: true,
    proxy: { "/ws": { target: "ws://localhost:8080", ws: true } },
  },
  build: { target: "es2022", chunkSizeWarningLimit: 700 },
});
