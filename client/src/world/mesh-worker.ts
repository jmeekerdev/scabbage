import { meshChunk, type ChunkMesh, type MeshData, type RenderTable } from "./mesher";

export type MeshRequest =
  { kind: "table"; table: RenderTable } | { kind: "mesh"; job: number; volume: Uint16Array };

export interface MeshResult {
  job: number;
  mesh: ChunkMesh;
}

let table: RenderTable | undefined;

self.onmessage = (ev: MessageEvent<MeshRequest>) => {
  const req = ev.data;
  if (req.kind === "table") {
    table = req.table;
    return;
  }
  if (!table) throw new Error("mesh worker has no render table");
  const mesh = meshChunk(req.volume, table);
  const result: MeshResult = { job: req.job, mesh };
  // The options form type-checks under both DOM and worker typings.
  self.postMessage(result, { transfer: [...buffers(mesh.opaque), ...buffers(mesh.translucent)] });
};

function buffers(m: MeshData): ArrayBuffer[] {
  return [m.positions, m.uvs, m.faces, m.layers, m.ao, m.indices].map(
    (a) => a.buffer as ArrayBuffer,
  );
}
