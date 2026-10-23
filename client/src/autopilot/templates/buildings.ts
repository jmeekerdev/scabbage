import { Cells, type Size, type Template } from "./shape";

const DOOR_TALL = 6; // the avatar is about 5 blocks tall

/** Cuts a doorway, centered in the front wall, just above the floor. */
function doorway(c: Cells, w: number): void {
  const wide = w % 2 ? 3 : 2;
  const x = Math.floor((w - wide) / 2);
  c.box([x, 1, 0], [x + wide - 1, DOOR_TALL, 0], "air");
}

/** Hollow walls from y 1 to `top` around a cleared inside. */
function walls(c: Cells, { w, d }: Size, top: number): void {
  c.box([0, 1, 0], [w - 1, top, d - 1], "main");
  if (w > 2 && d > 2) c.box([1, 1, 1], [w - 2, top, d - 2], "air");
}

const roofRise = (w: number) => Math.ceil(w / 2);
const cabinWallTop = ({ w, h }: Size) => h - 1 - roofRise(w);

/** A log cabin: floor, walls with a doorway and side windows, and a gable roof. */
export const cabin: Template = {
  about: "a log cabin with a doorway, windows and a gable roof",
  size: { w: 9, d: 9, h: 13 },
  min: { w: 7, d: 7, h: 12 },
  material: "log",
  floor: "planks",
  roof: "planks",
  check: (size) => {
    const tall = roofRise(size.w) + DOOR_TALL + 2;
    return cabinWallTop(size) < DOOR_TALL + 1
      ? `A cabin ${size.w} wide needs to be at least ${tall} tall.`
      : undefined;
  },
  cells: (size) => {
    const { w, d } = size;
    const top = cabinWallTop(size);
    const c = new Cells();
    c.box([0, 0, 0], [w - 1, 0, d - 1], "floor");
    walls(c, size, top);
    for (let k = 0; k < roofRise(w); k++) {
      const y = top + 1 + k;
      c.box([k, y, 0], [w - 1 - k, y, d - 1], "main"); // the gable ends
      if (d > 2) c.box([k, y, 1], [w - 1 - k, y, d - 2], "air");
      c.box([k, y, 0], [k, y, d - 1], "roof");
      c.box([w - 1 - k, y, 0], [w - 1 - k, y, d - 1], "roof");
    }
    doorway(c, w);
    const wide = Math.min(3, d - 4);
    const z = Math.floor((d - wide) / 2);
    for (const x of [0, w - 1]) c.box([x, 3, z], [x, 5, z + wide - 1], "window");
    return c.list();
  },
};

const SLIT_EVERY = 7; // blocks between the tower's window slits, going up

/** A tower: hollow walls with a doorway and window slits, a flat top and battlements. */
export const tower: Template = {
  about: "a tower with a doorway, window slits, a flat top and battlements",
  size: { w: 9, d: 9, h: 21 },
  min: { w: 5, d: 5, h: DOOR_TALL + 3 },
  material: "cobblestone",
  cells: (size) => {
    const { w, d, h } = size;
    const c = new Cells();
    c.box([0, 0, 0], [w - 1, 0, d - 1], "main");
    walls(c, size, h - 3);
    c.box([0, h - 2, 0], [w - 1, h - 2, d - 1], "main");
    for (let x = 0; x < w; x++)
      for (let z = 0; z < d; z++) {
        const edge = x === 0 || z === 0 || x === w - 1 || z === d - 1;
        if (edge && (x + z) % 2 === 0) c.set(x, h - 1, z, "main");
      }
    doorway(c, w);
    const [mx, mz] = [Math.floor(w / 2), Math.floor(d / 2)];
    for (let y = DOOR_TALL + 3; y + 2 <= h - 4; y += SLIT_EVERY) {
      c.box([mx, y, d - 1], [mx, y + 2, d - 1], "window");
      c.box([0, y, mz], [0, y + 2, mz], "window");
      c.box([w - 1, y, mz], [w - 1, y + 2, mz], "window");
    }
    return c.list();
  },
};
