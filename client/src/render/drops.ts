import {
  CanvasTexture,
  NearestFilter,
  type Scene,
  Sprite,
  SpriteMaterial,
  SRGBColorSpace,
} from "three";

export interface DropInfo {
  id: bigint;
  block: number;
  pos?: { x: number; y: number; z: number };
}

const SIZE = 0.3;

/** Items lying in the world, drawn as small bobbing block icons. */
export class Drops {
  private readonly sprites = new Map<bigint, Sprite>();
  private readonly materials = new Map<number, SpriteMaterial>();
  private icon: (block: number) => HTMLCanvasElement = () => document.createElement("canvas");

  constructor(private readonly scene: Scene) {}

  /** Uses new block icons, e.g. after a block was repainted. */
  setIcons(icon: (block: number) => HTMLCanvasElement): void {
    this.icon = icon;
    for (const m of this.materials.values()) {
      m.map?.dispose();
      m.dispose();
    }
    this.materials.clear();
    for (const s of this.sprites.values()) s.material = this.material(s.userData.block as number);
  }

  change(dropped: readonly DropInfo[], gone: readonly bigint[]): void {
    for (const id of gone) this.remove(id);
    for (const d of dropped) {
      const s = this.sprites.get(d.id) ?? this.add(d);
      s.position.set(d.pos?.x ?? 0, (d.pos?.y ?? 0) + SIZE / 2, d.pos?.z ?? 0);
      s.userData.baseY = s.position.y;
    }
  }

  clear(): void {
    for (const id of [...this.sprites.keys()]) this.remove(id);
  }

  /** Bobs every item gently; call once per frame. */
  update(nowMs: number): void {
    for (const [id, s] of this.sprites)
      s.position.y = (s.userData.baseY as number) + 0.08 * Math.sin(nowMs / 400 + Number(id));
  }

  private add(d: DropInfo): Sprite {
    const s = new Sprite(this.material(d.block));
    s.scale.set(SIZE, SIZE, SIZE);
    s.userData.block = d.block;
    s.name = `drop ${d.id}`;
    this.sprites.set(d.id, s);
    this.scene.add(s);
    return s;
  }

  private remove(id: bigint): void {
    const s = this.sprites.get(id);
    if (s) this.scene.remove(s);
    this.sprites.delete(id);
  }

  private material(block: number): SpriteMaterial {
    let m = this.materials.get(block);
    if (!m) {
      const map = new CanvasTexture(this.icon(block));
      map.magFilter = NearestFilter;
      map.colorSpace = SRGBColorSpace;
      m = new SpriteMaterial({ map, transparent: true, alphaTest: 0.5 });
      this.materials.set(block, m);
    }
    return m;
  }
}
