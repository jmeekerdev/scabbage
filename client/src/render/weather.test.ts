import { describe, expect, it } from "vitest";
import { lookById } from "../looks/registry";
import { DAY_MS, MIN_LIGHT, skyAt } from "./sky";
import { FADE_MS, flashAt, SPELL_MS, snowsAt, Weather, weatherAt, weatherSky } from "./weather";

const spell = (n: number) => n * SPELL_MS + 1;
const { CLEAR, RAIN, STORM } = Weather;

describe("weatherAt", () => {
  it("picks the same spells as the server's weather package", () => {
    const want = [CLEAR, CLEAR, CLEAR, RAIN, CLEAR, RAIN, RAIN, CLEAR, CLEAR, RAIN, RAIN, CLEAR];
    want.push(CLEAR, CLEAR, RAIN, CLEAR, CLEAR, STORM, CLEAR, RAIN);
    expect(want.map((_, i) => weatherAt(spell(i)).kind)).toEqual(want);
  });

  it("fades in over the start of a spell from the one before", () => {
    expect(weatherAt(spell(3)).wet).toBeCloseTo(0);
    expect(weatherAt(spell(3) + FADE_MS / 2).wet).toBeCloseTo(0.5);
    expect(weatherAt(spell(4) - 2).wet).toBe(1);
    expect(weatherAt(spell(17) + FADE_MS).storm).toBe(1);
    expect(weatherAt(spell(18) + FADE_MS).storm).toBe(0);
  });
});

describe("weatherAt with held weather", () => {
  // Spell 3 is rain and 4 clear, so holding a storm over them shows every change.
  const held = { kind: STORM, sinceMs: spell(3) + FADE_MS, untilMs: spell(4) + FADE_MS };

  it("fades from the weather it found into the held weather", () => {
    expect(weatherAt(held.sinceMs, held)).toMatchObject({ kind: STORM, wet: 1, storm: 0 });
    expect(weatherAt(held.sinceMs + FADE_MS / 2, held).storm).toBeCloseTo(0.5);
    expect(weatherAt(spell(4) + 1, held)).toMatchObject({ kind: STORM, wet: 1, storm: 1 });
  });

  it("fades back to the wall clock's weather once the hold ends", () => {
    expect(weatherAt(held.untilMs + FADE_MS / 2, held).wet).toBeCloseTo(0.5);
    expect(weatherAt(held.untilMs + FADE_MS, held)).toEqual(weatherAt(held.untilMs + FADE_MS));
  });

  it("holds for good with no end", () => {
    const always = { ...held, untilMs: Infinity };
    expect(weatherAt(spell(40), always).kind).toBe(STORM);
  });
});

describe("snowsAt", () => {
  it("snows only high up", () => {
    expect(snowsAt(40)).toBe(false);
    expect(snowsAt(400)).toBe(true);
  });
});

describe("flashAt", () => {
  it("lights up storms now and then, briefly, and never clear skies", () => {
    const storm = spell(17) + FADE_MS;
    const lit = Array.from({ length: 6000 }, (_, i) => flashAt(storm + i * 10)).filter(
      (f) => f > 0,
    );
    expect(lit.length).toBeGreaterThan(5);
    expect(lit.length).toBeLessThan(600);
    const clear = Array.from({ length: 6000 }, (_, i) => flashAt(spell(18) + FADE_MS + i * 10));
    expect(clear.every((f) => f === 0)).toBe(true);
  });
});

describe("weatherSky", () => {
  const noon = skyAt(0.25 * DAY_MS, lookById("today").sky);

  it("leaves a clear sky as it is", () => {
    expect(weatherSky(noon, { kind: CLEAR, wet: 0, storm: 0 }, 0)).toEqual(noon);
  });

  it("darkens and grays the sky while it rains, more in storms, never below night", () => {
    const rain = weatherSky(noon, { kind: RAIN, wet: 1, storm: 0 }, 0);
    const storm = weatherSky(noon, { kind: STORM, wet: 1, storm: 1 }, 0);
    expect(rain.daylight).toBeLessThan(noon.daylight);
    expect(storm.daylight).toBeLessThan(rain.daylight);
    expect(storm.daylight).toBeGreaterThan(MIN_LIGHT);
    const blueness = (c: { r: number; b: number }) => c.b - c.r;
    expect(blueness(rain.color)).toBeLessThan(blueness(noon.color));
  });

  it("brightens everything for a lightning flash", () => {
    const storm = { kind: STORM, wet: 1, storm: 1 };
    expect(weatherSky(noon, storm, 1).daylight).toBeGreaterThan(
      weatherSky(noon, storm, 0).daylight,
    );
  });
});
