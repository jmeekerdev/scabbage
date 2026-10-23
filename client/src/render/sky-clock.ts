import { Weather as PbWeather, type Sky } from "../pb/scabbage/v1/net_pb";
import type { Held, Weather } from "./weather";

/** The time and weather admins set; both otherwise follow the wall clock. */
export const skyClock: { dayOffsetMs: number; held?: Held } = { dayOffsetMs: 0 };

/** Takes the server's Sky message, or the plain wall clock without one. */
export function setSkyClock(m: Sky | undefined): void {
  skyClock.dayOffsetMs = Number(m?.dayOffsetMs ?? 0);
  skyClock.held =
    m && m.held !== PbWeather.UNSPECIFIED
      ? {
          kind: (m.held - 1) as Weather, // the proto counts from 1, leaving 0 for none
          sinceMs: Number(m.heldSinceMs),
          untilMs: Number(m.heldUntilMs) || Infinity,
        }
      : undefined;
}
