import { describe, expect, it } from "vitest";
import { serviceWorkerUrl } from "./pwa";

describe("serviceWorkerUrl", () => {
  it("names the build after the hashed main script", () => {
    expect(serviceWorkerUrl("http://h:8080/assets/main-D5RFZvVA.js")).toBe(
      "/sw.js?build=main-D5RFZvVA",
    );
  });
});
