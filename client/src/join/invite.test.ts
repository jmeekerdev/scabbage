import { describe, expect, it } from "vitest";
import { formatCode, inviteLink, normalizeCode, parseInvite } from "./invite";

describe("invite codes", () => {
  it("normalizes case, separators and ambiguous characters away", () => {
    expect(normalizeCode(" abcd-efgh ")).toBe("ABCDEFGH");
    expect(normalizeCode("A0O1IL-B")).toBe("AB");
  });

  it("formats codes in two groups", () => {
    expect(formatCode("ABCDEFGH")).toBe("ABCD-EFGH");
    expect(formatCode("ABC")).toBe("ABC");
  });

  it("reads a join code from the URL hash", () => {
    expect(parseInvite("#join=abcd-efgh")).toBe("ABCDEFGH");
    expect(parseInvite("#other=1")).toBeUndefined();
    expect(parseInvite("")).toBeUndefined();
    expect(parseInvite("#join=")).toBeUndefined();
  });

  it("builds a shareable link", () => {
    expect(inviteLink("http://192.168.1.5:8080", "ABCDEFGH")).toBe(
      "http://192.168.1.5:8080/#join=ABCD-EFGH",
    );
  });
});
