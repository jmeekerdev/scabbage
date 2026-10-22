import {
  FPS_CAPS,
  PRESETS,
  qualityFor,
  VIEW_DISTANCE,
  viewMetres,
  type FpsCap,
  type Preset,
  type Settings,
} from "../settings/settings";
import { lookCost, type LookCost } from "../looks/look";
import { LOOKS } from "../looks/registry";
import { el } from "./dom";

const PRESET_LABELS: Record<Preset, string> = {
  low: "Low (fastest)",
  medium: "Medium",
  high: "High (sharpest)",
};

const COST_LABELS: Record<LookCost, string> = {
  fastest: "fastest",
  light: "a little slower",
  heavier: "slower",
};

/** A look's picker label: its name, what it is like, and what it costs. */
export function lookLabel(id: string): string {
  const look = LOOKS.get(id);
  return look ? `${look.name}: ${look.about} (${COST_LABELS[lookCost(look)]})` : id;
}

export interface SettingsPanel {
  readonly node: HTMLElement;
  /** Shows settings changed elsewhere, such as by automatic quality. */
  show(s: Settings): void;
}

const metres = (chunks: number) => `${viewMetres(chunks)} m`;

/** A settings form that reports every change immediately. */
export function settingsPanel(initial: Settings, onChange: (s: Settings) => void): SettingsPanel {
  const distanceOut = el("output", {}, metres(initial.viewDistance));
  const distance = el("input", {
    type: "range",
    min: VIEW_DISTANCE.min,
    max: VIEW_DISTANCE.max,
    value: initial.viewDistance,
  });
  const preset = select(PRESETS, initial.preset, (p) => PRESET_LABELS[p]);
  const look = select([...LOOKS.keys()], initial.look, lookLabel);
  const fps = select(FPS_CAPS, initial.fpsCap, (f) => (f ? `${f} fps` : "Match screen"));
  const sound = el("input", { type: "checkbox", checked: initial.sound });
  const music = el("input", { type: "checkbox", checked: initial.music });
  const auto = el("input", { type: "checkbox", checked: initial.auto });
  const buttons = el("input", { type: "checkbox", checked: initial.buttons });
  const graphics: EventTarget[] = [distance, preset, fps];
  const reloadHint = el("p", { class: "muted", hidden: true }, "Smoothing changes after a reload.");

  const form = el(
    "form",
    { method: "dialog" },
    el("h1", {}, "Settings"),
    el("label", {}, "View distance ", distanceOut, distance),
    el("label", {}, "Look", look),
    el("label", {}, "Graphics", preset),
    el("label", {}, "Frame rate limit", fps),
    el("label", { class: "check" }, auto, "Lower graphics when the game is slow"),
    el("label", { class: "check" }, sound, "Sound effects"),
    el("label", { class: "check" }, music, "Music"),
    el("label", { class: "check" }, buttons, "Show menu buttons"),
    reloadHint,
  );
  form.addEventListener("input", (e) => {
    if (graphics.includes(e.target as EventTarget)) auto.checked = false; // the player chose
    const next: Settings = {
      viewDistance: Number(distance.value),
      preset: preset.value as Preset,
      fpsCap: Number(fps.value) as FpsCap,
      sound: sound.checked,
      music: music.checked,
      auto: auto.checked,
      look: look.value,
      buttons: buttons.checked,
    };
    distanceOut.textContent = metres(next.viewDistance);
    reloadHint.hidden = qualityFor(next.preset).antialias === qualityFor(initial.preset).antialias;
    onChange(next);
  });
  const show = (s: Settings) => {
    distance.value = String(s.viewDistance);
    distanceOut.textContent = metres(s.viewDistance);
    preset.value = s.preset;
    fps.value = String(s.fpsCap);
    auto.checked = s.auto;
    look.value = s.look;
    buttons.checked = s.buttons;
  };
  return { node: form, show };
}

function select<T extends string | number>(
  options: readonly T[],
  selected: T,
  label: (v: T) => string,
): HTMLSelectElement {
  return el(
    "select",
    {},
    ...options.map((v) => el("option", { value: v, selected: v === selected }, label(v))),
  );
}
