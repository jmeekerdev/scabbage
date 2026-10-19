/** The service worker script, keyed to this build so each deploy gets fresh caches. */
export function serviceWorkerUrl(scriptUrl: string): string {
  const file = new URL(scriptUrl).pathname.split("/").pop() ?? "";
  return `/sw.js?build=${encodeURIComponent(file.replace(/\.js$/, ""))}`;
}

/** Lets the game be installed; browsers allow service workers only over HTTPS or on localhost. */
export function registerServiceWorker(): void {
  if (!import.meta.env.PROD || !window.isSecureContext || !navigator.serviceWorker) return;
  navigator.serviceWorker.register(serviceWorkerUrl(import.meta.url)).catch(() => undefined);
}

type Lockable = ScreenOrientation & {
  lock?: (o: "landscape") => Promise<void>;
  unlock?: () => void;
};

/** True in a normal browser tab, where fullscreen has to be asked for. */
export function canGoFullscreen(): boolean {
  return document.fullscreenEnabled && matchMedia("(display-mode: browser)").matches;
}

/** Fills the screen and turns sideways on phones that allow it (not iPhones). Needs a tap. */
async function goLandscape(): Promise<void> {
  if (document.fullscreenElement || !canGoFullscreen()) return;
  await document.documentElement.requestFullscreen({ navigationUI: "hide" });
  await (screen.orientation as Lockable).lock?.("landscape");
}

/** Goes fullscreen and sideways, or back to normal if already there. */
export async function toggleLandscape(): Promise<void> {
  if (!document.fullscreenElement) return goLandscape();
  (screen.orientation as Lockable).unlock?.();
  await document.exitFullscreen();
}

/** Lets the phone turn freely again whenever fullscreen ends, e.g. by the back gesture. */
export function unlockOnFullscreenExit(): void {
  document.addEventListener("fullscreenchange", () => {
    if (!document.fullscreenElement) (screen.orientation as Lockable).unlock?.();
  });
}
