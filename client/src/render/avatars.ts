import {
  BoxGeometry,
  CanvasTexture,
  DataTexture,
  Float32BufferAttribute,
  Group,
  Mesh,
  type MeshBasicMaterial,
  NearestFilter,
  type Scene,
  Sprite,
  SpriteMaterial,
  SRGBColorSpace,
} from "three";
import type { RemoteState } from "../player/remote-players";
import { litMesh } from "./entity-material";
import { Gait, type Stance } from "./gait";
import { boxUVs, defaultSkin, PARTS, SKIN_H, SKIN_W, type Part } from "./skin";
import { METER } from "../world/constants";

const PX = 1.8 / 32; // meters per skin pixel: the figure is 32 pixels tall
const TAG_HEIGHT = 0.3; // meters
const EMOTE_SIZE = 0.7; // meters
const EMOTE_MS = 3000;

/** A box for one body part, with its skin pixels mapped onto each face. */
function partBox(part: Part): BoxGeometry {
  const { w, h, d } = PARTS[part];
  const g = new BoxGeometry(w * PX, h * PX, d * PX);
  const { position, normal } = g.attributes;
  g.setAttribute("uv", new Float32BufferAttribute(boxUVs(part, position.array, normal.array), 2));
  return g;
}

// Every figure shares these; only the skin texture differs.
const shapes = {
  head: partBox("head"),
  body: partBox("body"),
  arm: partBox("arm"),
  leg: partBox("leg"),
};
function skinTexture(pixels: Uint8Array): DataTexture {
  const tex = new DataTexture(pixels, SKIN_W, SKIN_H);
  tex.colorSpace = SRGBColorSpace;
  tex.magFilter = tex.minFilter = NearestFilter;
  tex.needsUpdate = true;
  return tex;
}

/** Skin pixels to draw: the player's own, or the default look from their color. */
export function skinPixels(skin: Uint8Array, color: number): Uint8Array {
  return skin.length === SKIN_W * SKIN_H * 4 ? skin : defaultSkin(color);
}

/** A part hanging from a pivot, so rotating the pivot swings it from the hip, shoulder or neck. */
function pivoted(mesh: Mesh, drop: number): Group {
  const pivot = new Group();
  pivot.add(at(mesh, 0, drop * PX, 0));
  return pivot;
}

/** One blocky player figure with a name tag; feet at the origin, facing -z. */
export class Avatar {
  readonly root = new Group();
  private readonly head: Group;
  private readonly legs: [Group, Group];
  private readonly arms: [Group, Group];
  private readonly gait = new Gait();
  private readonly tag?: Sprite;
  private readonly material: MeshBasicMaterial;
  private bubble?: { sprite: Sprite; untilMs: number };
  private shown?: Uint8Array;

  constructor(name: string | undefined, color: number, skin: Uint8Array) {
    this.root.scale.setScalar(METER); // the figure is modeled in meters
    this.material = litMesh({ alphaTest: 0.5 });
    this.setSkin(skin, color);
    const m = this.material;
    const leg = () => pivoted(new Mesh(shapes.leg, m), -6);
    const arm = () => pivoted(new Mesh(shapes.arm, m), -4);
    this.legs = [at(leg(), -2 * PX, 12 * PX, 0), at(leg(), 2 * PX, 12 * PX, 0)];
    this.arms = [at(arm(), -6 * PX, 22 * PX, 0), at(arm(), 6 * PX, 22 * PX, 0)];
    this.head = at(pivoted(new Mesh(shapes.head, m), 4), 0, 24 * PX, 0);
    this.root.add(
      ...this.legs,
      ...this.arms,
      at(new Mesh(shapes.body, m), 0, 18 * PX, 0),
      this.head,
    );
    if (name !== undefined) this.root.add(at((this.tag = nameTag(name)), 0, 2.05, 0));
  }

  /** Re-uploads the skin after its pixels were changed in place, e.g. by the editor. */
  refreshSkin(): void {
    if (this.material.map) this.material.map.needsUpdate = true;
  }

  /** Shows new skin pixels (or the default look when empty); does nothing if they're unchanged. */
  setSkin(skin: Uint8Array, color: number): void {
    if (skin === this.shown) return;
    this.shown = skin;
    this.material.map?.dispose();
    this.material.map = skinTexture(skinPixels(skin, color));
    this.material.needsUpdate = true;
  }

  pose(p: RemoteState["pose"], dtMs: number, nowMs: number): void {
    this.root.position.set(p.x, p.y, p.z);
    this.root.rotation.y = p.yaw;
    this.head.rotation.x = p.pitch;
    this.animate(this.gait.update(p, dtMs), nowMs);
  }

  /** Swings legs and arms opposite ways; spreads arms in the air and waves while emoting. */
  private animate({ swing, airborne }: Stance, nowMs: number): void {
    const [leftLeg, rightLeg] = this.legs;
    const [leftArm, rightArm] = this.arms;
    leftLeg.rotation.x = airborne ? 0.35 : swing;
    rightLeg.rotation.x = airborne ? -0.35 : -swing;
    leftArm.rotation.set(-swing * 0.8, 0, airborne ? -0.5 : 0);
    rightArm.rotation.set(swing * 0.8, 0, airborne ? 0.5 : 0);
    if (this.bubble) rightArm.rotation.set(0, 0, 2.6 + Math.sin(nowMs / 110) * 0.3);
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
    this.material.map?.dispose();
    this.material.dispose();
    this.tag?.material.map?.dispose();
    this.tag?.material.dispose();
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
  private lastMs?: number;

  constructor(private readonly scene: Scene) {}

  /** Shows an emote over a player, if they are drawn. */
  emote(entity: number, emoji: string, nowMs: number): void {
    this.shown.get(entity)?.emote(emoji, nowMs);
  }

  update(players: readonly RemoteState[], nowMs: number): void {
    const dtMs = nowMs - (this.lastMs ?? nowMs);
    this.lastMs = nowMs;
    const seen = new Set<number>();
    for (const p of players) {
      seen.add(p.entity);
      let a = this.shown.get(p.entity);
      if (!a) {
        a = new Avatar(p.name, p.color, p.skin);
        a.root.name = `player ${p.name ?? p.entity}`;
        this.shown.set(p.entity, a);
        this.scene.add(a.root);
      }
      a.setSkin(p.skin, p.color);
      a.expire(nowMs);
      a.pose(p.pose, dtMs, nowMs);
    }
    for (const [entity, a] of this.shown) {
      if (seen.has(entity)) continue;
      this.scene.remove(a.root);
      a.dispose();
      this.shown.delete(entity);
    }
  }
}
