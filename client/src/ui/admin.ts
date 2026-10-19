import type { Bytes } from "../net/codec";
import { encodeCreateJoinCode, encodeDecideArt, encodeDecideJoin } from "../net/codec";
import {
  ArtKind,
  type ArtRequest,
  type JoinCodeCreated,
  type PendingRequest,
  type ServerMessage,
} from "../pb/scabbage/v1/net_pb";
import { formatCode, inviteLink } from "../join/invite";
import { colorCss } from "../join/rules";
import { paintPixels } from "../render/icons";
import { FRONT_H, FRONT_W, frontView } from "../render/skin";
import { TEXTURE_SIZE } from "../render/textures";
import { el } from "./dom";

type Send = (bytes: Bytes) => void;

/** Cards for pending join requests with Approve / Deny buttons. */
export class PendingList {
  constructor(
    private readonly root: HTMLElement,
    private readonly send: Send,
    private readonly emptyText = "",
  ) {}

  set(requests: PendingRequest[]): void {
    const cards = requests.map((r) => this.card(r));
    const empty = this.emptyText ? [el("p", { class: "muted" }, this.emptyText)] : [];
    this.root.replaceChildren(...(cards.length ? cards : empty));
  }

  private card(r: PendingRequest): HTMLElement {
    const decide = (approve: boolean) => this.send(encodeDecideJoin(r.id, approve));
    return el(
      "div",
      { class: "request" },
      el("span", { class: "dot", style: `background: ${colorCss(r.color)}` }),
      el(
        "div",
        { class: "who" },
        el("strong", {}, r.name),
        " wants to join",
        r.returning && el("small", {}, "Already has an account. Only approve if it's really them."),
      ),
      el("button", { class: "approve", onclick: () => decide(true) }, "Let in"),
      el("button", { class: "deny", onclick: () => decide(false) }, "No"),
    );
  }
}

/** Cards for pixel art waiting for approval, each with a picture of it. */
export class ArtList {
  constructor(
    private readonly root: HTMLElement,
    private readonly send: Send,
    private readonly emptyText = "",
  ) {}

  set(requests: ArtRequest[]): void {
    const cards = requests.map((r) => this.card(r));
    const empty = this.emptyText ? [el("p", { class: "muted" }, this.emptyText)] : [];
    this.root.replaceChildren(...(cards.length ? cards : empty));
  }

  private card(r: ArtRequest): HTMLElement {
    const decide = (approve: boolean) => this.send(encodeDecideArt(r.id, approve));
    const what = r.kind === ArtKind.SKIN ? " made a new look" : ` made a block: ${r.name}`;
    return el(
      "div",
      { class: "request" },
      artPicture(r),
      el("div", { class: "who" }, el("strong", {}, r.author), what),
      el("button", { class: "approve", onclick: () => decide(true) }, "Looks good"),
      el("button", { class: "deny", onclick: () => decide(false) }, "No"),
    );
  }
}

function artPicture(r: ArtRequest): HTMLCanvasElement {
  const canvas =
    r.kind === ArtKind.SKIN
      ? paintPixels(frontView(r.pixels), FRONT_W, FRONT_H)
      : paintPixels(r.pixels, TEXTURE_SIZE, TEXTURE_SIZE);
  canvas.className = "art";
  return canvas;
}

const USE_CHOICES = [1, 5, 10];
const TTL_CHOICES: [string, number][] = [
  ["1 hour", 1],
  ["1 day", 24],
  ["1 week", 168],
];

/** Lets an admin make an invite code and share its link. */
export class InvitePanel {
  readonly node: HTMLElement;
  private readonly result = el("div", { class: "invite-result" });

  constructor(
    private readonly send: Send,
    private readonly origin: string,
  ) {
    const uses = el(
      "select",
      {},
      ...USE_CHOICES.map((n) =>
        el("option", { value: n }, `${n} ${n === 1 ? "person" : "people"}`),
      ),
    );
    const ttl = el(
      "select",
      {},
      ...TTL_CHOICES.map(([label, h]) => el("option", { value: h, selected: h === 24 }, label)),
    );
    const make = () => this.send(encodeCreateJoinCode(Number(uses.value), Number(ttl.value)));
    this.node = el(
      "section",
      { class: "invite" },
      el("h2", {}, "Invite someone"),
      el("div", { class: "row" }, el("label", {}, "For ", uses), el("label", {}, "Good for ", ttl)),
      el("button", { class: "primary", onclick: make }, "Make invite"),
      this.result,
    );
  }

  show(c: JoinCodeCreated): void {
    const link = inviteLink(this.origin, c.code);
    const input = el("input", { readonly: true, value: link, onfocus: () => input.select() });
    const copy = el("button", { onclick: () => void copyText(input, copy) }, "Copy link");
    const share =
      "share" in navigator &&
      el(
        "button",
        { onclick: () => void navigator.share({ title: "Join me in Scabbage", url: link }) },
        "Share",
      );
    this.result.replaceChildren(
      el("p", { class: "code" }, formatCode(c.code)),
      el("div", { class: "row" }, input, copy, share),
      el(
        "p",
        { class: "muted" },
        `Expires ${new Date(c.expiresMs).toLocaleString()}. You'll still approve each person.`,
      ),
    );
  }
}

// The Clipboard API needs HTTPS; over plain HTTP we select the text for a manual copy.
async function copyText(input: HTMLInputElement, button: HTMLButtonElement): Promise<void> {
  try {
    await navigator.clipboard.writeText(input.value);
    button.textContent = "Copied!";
  } catch {
    input.focus();
    input.select();
    button.textContent = "Press Ctrl+C / hold to copy";
  }
}

/** Routes admin-related server messages to the widgets that show them. */
export function adminRouter(pending: PendingList, art: ArtList, invite: InvitePanel) {
  return (msg: ServerMessage) => {
    if (msg.body.case === "pendingRequests") pending.set(msg.body.value.requests);
    if (msg.body.case === "artRequests") art.set(msg.body.value.requests);
    if (msg.body.case === "joinCodeCreated") invite.show(msg.body.value);
  };
}
