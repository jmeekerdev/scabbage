import { Cells, type Size, type Template } from "./shape";

const solidBox = ({ w, d, h }: Size) => {
  const c = new Cells();
  c.box([0, 0, 0], [w - 1, h - 1, d - 1], "main");
  return c.list();
};

export const wall: Template = {
  about: "a solid wall",
  size: { w: 9, d: 1, h: 6 },
  min: { w: 1, d: 1, h: 1 },
  material: "stone",
  cells: solidBox,
};

export const floor: Template = {
  about: "a flat floor or platform",
  size: { w: 9, d: 9, h: 1 },
  min: { w: 1, d: 1, h: 1 },
  material: "planks",
  cells: solidBox,
};

/** A ramp of steps rising from the front to the back, with the box above the steps cleared. */
export const stairs: Template = {
  about: "steps rising from the front to the back, one block high each",
  size: { w: 3, d: 9, h: 9 },
  min: { w: 1, d: 1, h: 1 },
  material: "cobblestone",
  cells: ({ w, d, h }) => {
    const c = new Cells();
    for (let z = 0; z < d; z++) {
      const top = Math.max(1, Math.floor(((z + 1) * h) / d));
      c.box([0, 0, z], [w - 1, top - 1, z], "main");
      if (top < h) c.box([0, top, z], [w - 1, h - 1, z], "air");
    }
    return c.list();
  },
};
