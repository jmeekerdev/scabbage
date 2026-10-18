import { describe, expect, it } from "vitest";
import { loadSecret, memoryStore, resetSecret, SECRET_BYTES } from "./identity";

describe("device secret", () => {
  it("is created once and then reused", () => {
    const store = memoryStore();
    const first = loadSecret(store);
    expect(first).toHaveLength(SECRET_BYTES);
    expect(loadSecret(store)).toEqual(first);
  });

  it("uses the supplied random source", () => {
    const secret = loadSecret(memoryStore(), (b) => b.fill(7));
    expect([...secret].every((v) => v === 7)).toBe(true);
  });

  it("is replaced after reset", () => {
    const store = memoryStore();
    const first = loadSecret(store, (b) => b.fill(1));
    resetSecret(store);
    expect(loadSecret(store, (b) => b.fill(2))).not.toEqual(first);
  });

  it("ignores a corrupt stored value", () => {
    const store = memoryStore();
    store.setItem("scabbage.deviceSecret", "not base64!!");
    expect(loadSecret(store)).toHaveLength(SECRET_BYTES);
  });
});
