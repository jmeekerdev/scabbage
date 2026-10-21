import { describe, expect, it } from "vitest";
import { scriptStatus } from "./script-status";

const shown = (over: Partial<Parameters<typeof scriptStatus>[0]>) => ({
  editable: true,
  error: "",
  saved: false,
  ...over,
});

describe("scriptStatus", () => {
  it("says a save worked, or what is still wrong", () => {
    expect(scriptStatus(shown({ saved: true }))).toEqual({ text: "Saved!", ok: true });
    expect(scriptStatus(shown({ saved: true, error: "line 2: oops" }))).toEqual({
      text: "Saved, but it has a problem: line 2: oops",
      ok: false,
    });
  });

  it("shows the last error when opened", () => {
    expect(scriptStatus(shown({ error: "boom" }))).toEqual({
      text: "Last time it ran: boom",
      ok: false,
    });
    expect(scriptStatus(shown({}))).toEqual({ text: "", ok: true });
  });

  it("explains why someone can only read it", () => {
    expect(scriptStatus(shown({ editable: false })).text).toBe(
      "You can read this script, but only builders here can change it.",
    );
  });
});
