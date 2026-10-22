import { HALF_WIDTH, HEIGHT } from "../player/physics";
import type { Vec } from "./route";

/** Whether the body may pass through the block at x,y,z. */
export type Passable = (x: number, y: number, z: number) => boolean;

const NODE = 3; // blocks a side: a meter, just wider than the body
const TALL = 2; // nodes the body stands in, 6 blocks for its 5.4
const MAX_NODES = 8000; // nodes searched before giving up, a few milliseconds' work
const SIDES = [
  [1, 0, 0],
  [-1, 0, 0],
  [0, 1, 0],
  [0, -1, 0],
  [0, 0, 1],
  [0, 0, -1],
] as const;

const CORNERS = [
  [-1, -1],
  [-1, 1],
  [1, -1],
  [1, 1],
] as const;

type Node = [number, number, number];

/**
 * Feet points to fly through, in open air, from `from` until `near` is satisfied; `goal` steers the
 * search. Searches a meter grid, so it finds gaps a meter or more wide. Undefined if there's no way.
 */
export function flyPath(
  from: Vec,
  goal: Vec,
  near: (feet: Vec) => boolean,
  passable: Passable,
  maxNodes = MAX_NODES,
): Vec[] | undefined {
  const fits = fitter(passable);
  const start = nodeOf(from);
  const key = (n: Node) => n.join();
  const cameFrom = new Map<string, Node | undefined>([[key(start), undefined]]);
  const cost = new Map([[key(start), 0]]);
  const open = new Heap<Node>();
  open.push(start, 0);
  for (let searched = 0; searched < maxNodes && open.size; searched++) {
    const node = open.pop()!;
    if (near(feetOf(node))) return smooth(from, walk(cameFrom, node, key), fits);
    for (const [dx, dy, dz] of SIDES) {
      const next: Node = [node[0] + dx, node[1] + dy, node[2] + dz];
      const g = cost.get(key(node))! + 1;
      if (g >= (cost.get(key(next)) ?? Infinity) || !fits(next)) continue;
      cost.set(key(next), g);
      cameFrom.set(key(next), node);
      open.push(next, g + distance(feetOf(next), goal) / NODE);
    }
  }
  return undefined;
}

const nodeOf = (p: Vec): Node => [
  Math.floor(p.x / NODE),
  Math.floor(p.y / NODE),
  Math.floor(p.z / NODE),
];

const feetOf = ([i, j, k]: Node): Vec => ({
  x: (i + 0.5) * NODE,
  y: j * NODE,
  z: (k + 0.5) * NODE,
});

const distance = (a: Vec, b: Vec) => Math.hypot(a.x - b.x, a.y - b.y, a.z - b.z);

/** Whether the body fits in a node, remembering the answers. */
function fitter(passable: Passable): (n: Node) => boolean {
  const known = new Map<string, boolean>();
  return (n) => {
    const k = n.join();
    let fits = known.get(k);
    if (fits === undefined) known.set(k, (fits = bodyFits(n, passable)));
    return fits;
  };
}

function bodyFits([i, j, k]: Node, passable: Passable): boolean {
  for (let y = j * NODE; y < (j + TALL) * NODE; y++)
    for (let x = i * NODE; x < (i + 1) * NODE; x++)
      for (let z = k * NODE; z < (k + 1) * NODE; z++) if (!passable(x, y, z)) return false;
  return true;
}

function walk(
  cameFrom: Map<string, Node | undefined>,
  end: Node,
  key: (n: Node) => string,
): Node[] {
  const nodes: Node[] = [];
  for (let n: Node | undefined = end; n; n = cameFrom.get(key(n))) nodes.unshift(n);
  return nodes;
}

/** Drops each waypoint the body can skip, flying straight on to a later one. */
function smooth(from: Vec, nodes: Node[], fits: (n: Node) => boolean): Vec[] {
  const out: Vec[] = [];
  let at = from;
  let i = 0;
  while (i < nodes.length) {
    let j = nodes.length - 1;
    while (j > i && !clear(at, feetOf(nodes[j]!), fits)) j--;
    at = feetOf(nodes[j]!);
    out.push(at);
    i = j + 1;
  }
  return out;
}

/** Whether every node along a straight line fits the body. */
function clear(a: Vec, b: Vec, fits: (n: Node) => boolean): boolean {
  const steps = Math.ceil((distance(a, b) / NODE) * 3);
  for (let s = 1; s <= steps; s++) {
    const t = s / steps;
    const p = { x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t, z: a.z + (b.z - a.z) * t };
    // The body is wider and taller than a point: check the nodes its feet and head corners are in.
    for (const [ox, oz] of CORNERS) {
      const corner = { x: p.x + ox * HALF_WIDTH, y: p.y, z: p.z + oz * HALF_WIDTH };
      const head = { ...corner, y: p.y + HEIGHT - NODE }; // the node from here up covers the head
      if (!fits(nodeOf(corner)) || !fits(nodeOf(head))) return false;
    }
  }
  return true;
}

/** A binary min-heap by priority. */
class Heap<T> {
  private readonly items: { item: T; priority: number }[] = [];

  get size(): number {
    return this.items.length;
  }

  push(item: T, priority: number): void {
    const a = this.items;
    a.push({ item, priority });
    for (let i = a.length - 1; i > 0;) {
      const up = (i - 1) >> 1;
      if (a[up]!.priority <= a[i]!.priority) break;
      [a[up], a[i]] = [a[i]!, a[up]!];
      i = up;
    }
  }

  pop(): T | undefined {
    const a = this.items;
    const top = a[0];
    const last = a.pop();
    if (a.length && last) {
      a[0] = last;
      for (let i = 0; ;) {
        const [l, r] = [2 * i + 1, 2 * i + 2];
        let m = i;
        if (l < a.length && a[l]!.priority < a[m]!.priority) m = l;
        if (r < a.length && a[r]!.priority < a[m]!.priority) m = r;
        if (m === i) break;
        [a[m], a[i]] = [a[i]!, a[m]!];
        i = m;
      }
    }
    return top?.item;
  }
}
