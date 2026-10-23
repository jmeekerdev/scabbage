import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { SkySchema, Weather as PbWeather } from "../pb/scabbage/v1/net_pb";
import { setSkyClock, skyClock } from "./sky-clock";
import { Weather } from "./weather";

describe("setSkyClock", () => {
  it("takes the clock offset and held weather, holding for good without an end", () => {
    setSkyClock(create(SkySchema, { dayOffsetMs: 5000n, held: PbWeather.STORM, heldSinceMs: 7n }));
    expect(skyClock).toEqual({
      dayOffsetMs: 5000,
      held: { kind: Weather.STORM, sinceMs: 7, untilMs: Infinity },
    });
  });

  it("goes back to the wall clock without a message", () => {
    setSkyClock(undefined);
    expect(skyClock).toEqual({ dayOffsetMs: 0, held: undefined });
  });
});
