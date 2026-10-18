import {
  FPS_CAPS,
  PRESETS,
  qualityFor,
  VIEW_DISTANCE,
  type FpsCap,
  type Preset,
  type Settings,
} from "../settings/settings";
import { el } from "./dom";

const PRESET_LABELS: Record<Preset, string> = {
  low: "Low (fastest)",
  medium: "Medium",
  high: "High (sharpest)",
};

/** A settings form that reports every change immediately. */
export function settingsPanel(initial: Settings, onChange: (s: Settings) => void): HTMLElement {
  const distanceOut = el("output", {}, String(initial.viewDistance));
  const distance = el("input", {
    type: "range",
    min: VIEW_DISTANCE.min,
    max: VIEW_DISTANCE.max,
    value: initial.viewDistance,
  });
  const preset = select(PRESETS, initial.preset, (p) => PRESET_LABELS[p]);
  const fps = select(FPS_CAPS, initial.fpsCap, (f) => (f ? `${f} fps` : "Match screen"));
  const reloadHint = el("p", { class: "muted", hidden: true }, "Smoothing changes after a reload.");

  const form = el(
    "form",
    { method: "dialog" },
    el("h1", {}, "Settings"),
    el("label", {}, "View distance ", distanceOut, distance),
    el("label", {}, "Graphics", preset),
    el("label", {}, "Frame rate limit", fps),
    reloadHint,
  );
  form.addEventListener("input", () => {
    const next: Settings = {
      viewDistance: Number(distance.value),
      preset: preset.value as Preset,
      fpsCap: Number(fps.value) as FpsCap,
    };
    distanceOut.textContent = distance.value;
    reloadHint.hidden = qualityFor(next.preset).antialias === qualityFor(initial.preset).antialias;
    onChange(next);
  });
  return form;
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
