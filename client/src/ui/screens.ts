import type { JoinInit } from "../net/codec";
import { formatCode, normalizeCode } from "../join/invite";
import { AVATAR_COLORS, cleanName, colorCss } from "../join/rules";
import { el } from "./dom";

export interface JoinFormOptions {
  code?: string;
  name?: string;
  color?: number;
  error?: string;
  onSubmit(join: JoinInit): void;
}

/** Shows the "ask to join" form: invite code, display name and avatar color. */
export function showJoinForm(overlay: HTMLElement, o: JoinFormOptions): void {
  const code = el("input", {
    name: "code",
    value: o.code ? formatCode(o.code) : "",
    placeholder: "ABCD-EFGH",
    autocomplete: "off",
    autocapitalize: "characters",
    required: true,
  });
  const name = el("input", {
    name: "name",
    value: o.name ?? "",
    maxlength: 16,
    autocomplete: "nickname",
    required: true,
  });
  const error = el("p", { class: "error", role: "alert" }, o.error);
  const colors = colorPicker(o.color ?? AVATAR_COLORS[0]);

  const form = el(
    "form",
    { class: "card" },
    el("h1", {}, "Join Scabbage"),
    el("label", {}, "Invite code", code),
    el("label", {}, "Your name", name),
    el("fieldset", {}, el("legend", {}, "Your color"), colors.node),
    error,
    el("button", { type: "submit", class: "primary" }, "Ask to join"),
  );
  form.addEventListener("submit", (ev) => {
    ev.preventDefault();
    const join = {
      code: normalizeCode(code.value),
      name: cleanName(name.value),
      color: colors.value(),
    };
    if (!join.code) error.textContent = "Type the invite code you were given.";
    else if (!join.name) error.textContent = "Use up to 16 letters, numbers, spaces, _ or -.";
    else o.onSubmit({ ...join, name: join.name });
  });
  overlay.replaceChildren(form);
  overlay.hidden = false;
  (o.code ? name : code).focus();
}

function colorPicker(initial: number) {
  const node = el("div", { class: "swatches" });
  for (const c of AVATAR_COLORS) {
    node.append(
      el("input", {
        type: "radio",
        name: "color",
        value: c,
        checked: c === initial,
        style: `--swatch: ${colorCss(c)}`,
        "aria-label": colorCss(c),
      }),
    );
  }
  const value = () =>
    Number(node.querySelector<HTMLInputElement>("input:checked")?.value ?? initial);
  return { node, value };
}

export interface MessageOptions {
  title: string;
  text: string;
  actionLabel?: string;
  onAction?(): void;
}

/** Shows a simple card with an optional single action button. */
export function showMessage(overlay: HTMLElement, o: MessageOptions): void {
  const button =
    o.actionLabel &&
    el("button", { class: "primary", onclick: () => o.onAction?.() }, o.actionLabel);
  overlay.replaceChildren(
    el("div", { class: "card" }, el("h1", {}, o.title), el("p", {}, o.text), button),
  );
  overlay.hidden = false;
}

export function hideOverlay(overlay: HTMLElement): void {
  overlay.replaceChildren();
  overlay.hidden = true;
}
