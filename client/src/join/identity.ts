export const SECRET_BYTES = 32;
const KEY = "scabbage.deviceSecret";

/** The subset of Web Storage used here, so tests and private-mode fallbacks can supply their own. */
export interface KeyValueStore {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

type RandomFill = (bytes: Uint8Array<ArrayBuffer>) => Uint8Array<ArrayBuffer>;

// crypto.getRandomValues works without HTTPS, unlike crypto.subtle.
const cryptoFill: RandomFill = (b) => crypto.getRandomValues(b);

/** Returns this device's secret, creating and storing one on first use. */
export function loadSecret(store: KeyValueStore, random: RandomFill = cryptoFill): Uint8Array {
  const stored = decode(store.getItem(KEY));
  if (stored) return stored;
  const fresh = random(new Uint8Array(SECRET_BYTES));
  store.setItem(KEY, btoa(String.fromCharCode(...fresh)));
  return fresh;
}

/** Forgets the secret so the next load creates a new identity (e.g. after a denied request). */
export function resetSecret(store: KeyValueStore): void {
  store.removeItem(KEY);
}

function decode(value: string | null): Uint8Array | undefined {
  if (!value) return undefined;
  try {
    const bytes = Uint8Array.from(atob(value), (c) => c.charCodeAt(0));
    return bytes.length === SECRET_BYTES ? bytes : undefined;
  } catch {
    return undefined;
  }
}

/** In-memory store for tests and browsers that block localStorage. */
export function memoryStore(): KeyValueStore {
  const m = new Map<string, string>();
  return {
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => void m.set(k, v),
    removeItem: (k) => void m.delete(k),
  };
}

/** localStorage when usable, otherwise a memory store (the device is then forgotten on reload). */
export function browserStore(): KeyValueStore {
  try {
    localStorage.setItem("scabbage.probe", "1");
    localStorage.removeItem("scabbage.probe");
    return localStorage;
  } catch {
    return memoryStore();
  }
}
