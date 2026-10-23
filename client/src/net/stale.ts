const TRIED_KEY = "scabbage.reloadedFor"; // the build we last reloaded for, this tab only

/** The module script a page loads; its name changes with every build. */
export function entryScript(html: string): string | undefined {
  return /<script[^>]*type="module"[^>]*src="([^"]+)"/.exec(html)?.[1];
}

/** The newer build to reload for, unless this tab already tried it or either build is unknown. */
export function staleBuild(
  running: string | undefined,
  served: string,
  tried: string | undefined,
): string | undefined {
  const fresh = entryScript(served);
  return running && fresh && fresh !== running && fresh !== tried ? fresh : undefined;
}

/** Reloads the page when the server now serves a newer build, e.g. after a reconnect. */
export async function reloadIfStale(): Promise<void> {
  const running = document
    .querySelector<HTMLScriptElement>('script[type="module"]')
    ?.getAttribute("src");
  const served = await fetch(location.pathname, { cache: "no-store" }).then((r) => r.text());
  const fresh = staleBuild(
    running ?? undefined,
    served,
    sessionStorage.getItem(TRIED_KEY) ?? undefined,
  );
  if (!fresh) return;
  sessionStorage.setItem(TRIED_KEY, fresh);
  location.reload();
}
