import { describe, expect, it } from "vitest";
import { entryScript, staleBuild } from "./stale";

describe("entryScript", () => {
  it("finds the page's module script", () => {
    const html = '<head><script type="module" crossorigin src="/assets/main-cJIrs7VX.js"></script>';
    expect(entryScript(html)).toBe("/assets/main-cJIrs7VX.js");
    expect(entryScript("<p>no scripts</p>")).toBeUndefined();
  });
});

describe("staleBuild", () => {
  const page = (src: string) => `<script type="module" src="${src}"></script>`;

  it("says when the server serves another build than the one running", () => {
    expect(staleBuild("/assets/main-old.js", page("/assets/main-new.js"), undefined)).toBe(
      "/assets/main-new.js",
    );
    expect(
      staleBuild("/assets/main-new.js", page("/assets/main-new.js"), undefined),
    ).toBeUndefined();
  });

  it("reloads once per build, so a stubborn cache can't loop", () => {
    expect(
      staleBuild("/assets/main-old.js", page("/assets/main-new.js"), "/assets/main-new.js"),
    ).toBeUndefined();
  });

  it("ignores pages it can't read", () => {
    expect(staleBuild("/assets/main-old.js", "<html>oops", undefined)).toBeUndefined();
    expect(staleBuild(undefined, page("/assets/main-new.js"), undefined)).toBeUndefined();
  });
});
