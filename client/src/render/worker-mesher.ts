import type { MeshRequest, MeshResult } from "../world/mesh-worker";
import type { ChunkMesh, RenderTable } from "../world/mesher";
import type { Mesher } from "../world/world-client";

const JOBS_PER_WORKER = 2;

/** Meshes chunks on background workers so the frame loop never stalls. */
export class WorkerMesher implements Mesher {
  private readonly workers: Worker[];
  private readonly busy: number[];
  private readonly jobs = new Map<number, (m: ChunkMesh) => void>();
  private nextJob = 0;

  constructor(count = navigator.hardwareConcurrency >= 4 ? 2 : 1) {
    this.workers = Array.from({ length: count }, (_, i) => {
      const w = new Worker(new URL("../world/mesh-worker.ts", import.meta.url), { type: "module" });
      w.onmessage = (ev: MessageEvent<MeshResult>) => this.finished(i, ev.data);
      return w;
    });
    this.busy = this.workers.map(() => 0);
  }

  setTable(table: RenderTable): void {
    const req: MeshRequest = { kind: "table", table };
    for (const w of this.workers) w.postMessage(req);
  }

  capacity(): number {
    return this.workers.length * JOBS_PER_WORKER - this.jobs.size;
  }

  mesh(volume: Uint16Array, done: (mesh: ChunkMesh) => void): void {
    const worker = this.busy.indexOf(Math.min(...this.busy));
    const job = this.nextJob++;
    this.jobs.set(job, done);
    this.busy[worker]++;
    const req: MeshRequest = { kind: "mesh", job, volume };
    this.workers[worker].postMessage(req, [volume.buffer]);
  }

  private finished(worker: number, { job, mesh }: MeshResult): void {
    const done = this.jobs.get(job);
    if (!done) return;
    this.jobs.delete(job);
    this.busy[worker]--;
    done(mesh);
  }
}
