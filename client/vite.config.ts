/// <reference types="node" />
import { readdirSync, readFileSync } from "node:fs";
import { defineConfig, type Plugin } from "vite";

const BLOCKLY_MEDIA = "node_modules/blockly/media";

// Copies Blockly's few pictures and sounds into the build, so the editor never fetches from elsewhere.
function blocklyMedia(): Plugin {
  return {
    name: "blockly-media",
    apply: "build",
    generateBundle() {
      for (const file of readdirSync(BLOCKLY_MEDIA)) {
        const source = readFileSync(`${BLOCKLY_MEDIA}/${file}`);
        this.emitFile({ type: "asset", fileName: `assets/blockly/${file}`, source });
      }
    },
  };
}

// Dev server proxies the game socket to the Go server on :8080.
export default defineConfig({
  plugins: [blocklyMedia()],
  server: {
    host: true,
    proxy: { "/ws": { target: "ws://localhost:8080", ws: true } },
  },
  build: {
    target: "es2022",
    chunkSizeWarningLimit: 850, // the Blockly chunk, loaded only by the script editor, is ~820 kB
    rolldownOptions: {
      input: {
        main: "index.html",
        admin: "admin.html",
      },
    },
  },
});
