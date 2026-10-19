import "./style.css";
import { browserStore, loadSecret } from "./join/identity";
import { Connection } from "./net/connection";
import { socketUrl } from "./net/url";
import type { ServerMessage } from "./pb/scabbage/v1/net_pb";
import { adminRouter, ArtList, InvitePanel, PendingList } from "./ui/admin";
import { $, el } from "./ui/dom";

// Console mode authenticates this device's admin account without waking the game world.
const page = $("#admin");
const statusEl = $("#status");
const insecure = !window.isSecureContext;
const INSECURE_WARNING =
  "This connection isn't encrypted (no HTTPS). Use the TLS setup for play over the internet.";
let onMessage: ((msg: ServerMessage) => void) | undefined;

const conn: Connection = new Connection(
  socketUrl(location),
  { clientName: "admin page", secret: loadSecret(browserStore()), console: true },
  {
    onStatus: (status, detail) => {
      statusEl.textContent = detail ? `${status}: ${detail}` : status;
      statusEl.dataset.status = status;
    },
    onWelcome: (w) => mount(w.you?.name ?? "admin"),
    onMessage: (msg) => onMessage?.(msg),
    onRejected: (d) =>
      page.replaceChildren(
        el("h1", {}, "Admins only"),
        el("p", {}, d.reason),
        el(
          "p",
          {},
          "Join the game from this device with an admin account first. ",
          el("a", { href: "/" }, "Open the game"),
        ),
      ),
  },
);
conn.start();

function mount(name: string): void {
  const send = conn.send.bind(conn);
  const pendingRoot = el("div", { class: "requests" });
  const pending = new PendingList(pendingRoot, send, "Nobody is waiting right now.");
  const artRoot = el("div", { class: "requests" });
  const art = new ArtList(artRoot, send, "No new art to look at.");
  const invite = new InvitePanel(send, location.origin);
  page.replaceChildren(
    el("h1", {}, "Scabbage admin"),
    el(
      "p",
      { class: "muted" },
      `Signed in as ${name}. `,
      el("a", { href: "/" }, "Back to the game"),
    ),
    insecure ? el("p", { class: "warning" }, INSECURE_WARNING) : "",
    el("section", {}, el("h2", {}, "Waiting to join"), pendingRoot),
    el("section", {}, el("h2", {}, "New art"), artRoot),
    invite.node,
  );
  onMessage = adminRouter(pending, art, invite);
}
