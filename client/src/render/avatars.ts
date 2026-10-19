import {
  BoxGeometry,
  CanvasTexture,
  Color,
  Float32BufferAttribute,
  Group,
  Mesh,
  MeshBasicMaterial,
  type Scene,
  Sprite,
  SpriteMaterial,
} from "three";
import type { RemoteState } from "../player/remote-players";

const SKIN = 0xf0c8a0;
const TAG_HEIGHT = 0.3; // world units
const EMOTE_SIZE = 0.7; // world units
const EMOTE_MS = 3000;

/** A box with fixed per-face shading like the chunks: bright top, darker sides and bottom. */
function shadedBox(w: number, h: number, d: number): BoxGeometry {
  const g = new BoxGeometry(w, h, d);
  const faceLight = [0.8, 0.8, 1, 0.5, 0.7, 0.7]; // +x -x +y -y +z -z, 4 vertices each
  g.setAttribute(
    "color",
    new Float32BufferAttribute(
      faceLight.flatMap((l) => Array<number>(12).fill(l)),
      3,
    ),
  );
  return g;
}

const shapes = {
  legs: shadedBox(0.5, 0.75, 0.3),
  torso: shadedBox(0.6, 0.6, 0.35),
  head: shadedBox(0.5, 0.5, 0.5),
  eye: shadedBox(0.08, 0.08, 0.02),
};
const skin = new MeshBasicMaterial({ color: SKIN, vertexColors: true });
const eyes = new MeshBasicMaterial({ color: 0x202020 });

/** One blocky player figure with a name tag; feet at the origin, facing -z. */
class Avatar {
  readonly root = new Group();
  private readonly head = new Group();
  private readonly tag: Sprite;
  private readonly clothes: MeshBasicMaterial[];
  private bubble?: { sprite: Sprite; untilMs: number };

  constructor(name: string, color: number) {
    const shirt = new MeshBasicMaterial({ color, vertexColors: true });
    const trousers = new MeshBasicMaterial({
      color: new Color(color).multiplyScalar(0.55),
      vertexColors: true,
    });
    this.clothes = [shirt, trousers];
    this.tag = nameTag(name);
    this.root.add(
      at(new Mesh(shapes.legs, trousers), 0, 0.375, 0),
      at(new Mesh(shapes.torso, shirt), 0, 1.05, 0),
      at(this.head, 0, 1.3, 0),
      at(this.tag, 0, 2.05, 0),
    );
    this.head.add(
      at(new Mesh(shapes.head, skin), 0, 0.225, 0),
      at(new Mesh(shapes.eye, eyes), -0.11, 0.27, -0.26),
      at(new Mesh(shapes.eye, eyes), 0.11, 0.27, -0.26),
    );
  }

  pose(p: RemoteState["pose"]): void {
    this.root.position.set(p.x, p.y, p.z);
    this.root.rotation.y = p.yaw;
    this.head.rotation.x = p.pitch;
  }

  /** Floats an emoji over the head for a few seconds, replacing any earlier one. */
  emote(emoji: string, nowMs: number): void {
    this.clearBubble();
    const sprite = at(new Sprite(emojiMaterial(emoji)), 0, 2.6, 0);
    sprite.scale.set(EMOTE_SIZE, EMOTE_SIZE, 1);
    sprite.renderOrder = 1;
    this.root.add(sprite);
    this.bubble = { sprite, untilMs: nowMs + EMOTE_MS };
  }

  expire(nowMs: number): void {
    if (this.bubble && nowMs >= this.bubble.untilMs) this.clearBubble();
  }

  private clearBubble(): void {
    if (this.bubble) this.root.remove(this.bubble.sprite);
    this.bubble = undefined;
  }

  dispose(): void {
    for (const m of this.clothes) m.dispose();
    this.tag.material.map?.dispose();
    this.tag.material.dispose();
  }
}

function at<T extends Group | Mesh | Sprite>(o: T, x: number, y: number, z: number): T {
  o.position.set(x, y, z);
  return o;
}

/** A name label that always faces the camera and shows through walls so family can find each other. */
function nameTag(name: string): Sprite {
  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  const font = "bold 24px system-ui, sans-serif";
  if (ctx) ctx.font = font;
  canvas.width = Math.ceil((ctx?.measureText(name).width ?? 0) + 16);
  canvas.height = 32;
  if (ctx) {
    ctx.fillStyle = "rgba(0, 0, 0, 0.45)";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.font = font;
    ctx.fillStyle = "#fff";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(name, canvas.width / 2, canvas.height / 2 + 1);
  }
  const material = new SpriteMaterial({ map: new CanvasTexture(canvas), depthTest: false });
  const sprite = new Sprite(material);
  sprite.renderOrder = 1;
  sprite.scale.set((TAG_HEIGHT * canvas.width) / canvas.height, TAG_HEIGHT, 1);
  return sprite;
}

// One material per emoji, kept for the session; there are only a handful.
const emojiMaterials = new Map<string, SpriteMaterial>();

function emojiMaterial(emoji: string): SpriteMaterial {
  let m = emojiMaterials.get(emoji);
  if (m) return m;
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 64;
  const ctx = canvas.getContext("2d");
  if (ctx) {
    ctx.font = "52px system-ui, sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(emoji, 32, 36);
  }
  m = new SpriteMaterial({ map: new CanvasTexture(canvas), depthTest: false });
  emojiMaterials.set(emoji, m);
  return m;
}

/** Draws other players, adding and removing figures to match each frame's list. */
export class Avatars {
  private readonly shown = new Map<number, Avatar>();

  constructor(private readonly scene: Scene) {}

  /** Shows an emote over a player, if they are drawn. */
  emote(entity: number, emoji: string, nowMs: number): void {
    this.shown.get(entity)?.emote(emoji, nowMs);
  }

  update(players: readonly RemoteState[], nowMs: number): void {
    const seen = new Set<number>();
    for (const p of players) {
      seen.add(p.entity);
      let a = this.shown.get(p.entity);
      if (!a) {
        a = new Avatar(p.name, p.color);
        this.shown.set(p.entity, a);
        this.scene.add(a.root);
      }
      a.pose(p.pose);
      a.expire(nowMs);
    }
    for (const [entity, a] of this.shown) {
      if (seen.has(entity)) continue;
      this.scene.remove(a.root);
      a.dispose();
      this.shown.delete(entity);
    }
  }
}
