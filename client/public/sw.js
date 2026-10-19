/* global self, caches, location, fetch, URL */
// Keeps this build's files so the game starts fast and shows its own page offline.
// Pages always try the network first, so a deploy shows up on the next load.
const build = new URL(location.href).searchParams.get("build") ?? "dev";
const CACHE = `scabbage-${build}`;

self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (e) =>
  e.waitUntil(dropOldCaches().then(() => self.clients.claim())),
);

self.addEventListener("fetch", (e) => {
  const url = new URL(e.request.url);
  if (e.request.method !== "GET" || url.origin !== location.origin) return;
  if (url.pathname.startsWith("/assets/")) e.respondWith(cacheFirst(e.request));
  else if (e.request.mode === "navigate") e.respondWith(networkFirst(e.request));
});

async function dropOldCaches() {
  const names = await caches.keys();
  await Promise.all(names.filter((n) => n !== CACHE).map((n) => caches.delete(n)));
}

async function cacheFirst(request) {
  return (await caches.match(request)) ?? fetchAndKeep(request);
}

async function networkFirst(request) {
  try {
    return await fetchAndKeep(request);
  } catch (err) {
    const kept = await caches.match(request);
    if (kept) return kept;
    throw err;
  }
}

async function fetchAndKeep(request) {
  const response = await fetch(request);
  if (response.ok) {
    const copy = response.clone();
    void caches.open(CACHE).then((c) => c.put(request, copy));
  }
  return response;
}
