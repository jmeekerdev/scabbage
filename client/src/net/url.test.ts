import { describe, expect, it } from "vitest";
import { socketUrl } from "./url";

describe("socketUrl", () => {
  it("uses ws for http pages", () => {
    expect(socketUrl({ protocol: "http:", host: "192.168.1.5:8080" })).toBe(
      "ws://192.168.1.5:8080/ws",
    );
  });

  it("uses wss for https pages", () => {
    expect(socketUrl({ protocol: "https:", host: "game.example" })).toBe("wss://game.example/ws");
  });
});
