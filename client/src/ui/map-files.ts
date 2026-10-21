/** Saving exported adventure maps as .scabmap files, and picking one to import. */

/** The largest file the server accepts, as maps.MaxFileBytes. */
export const MAX_MAP_BYTES = 4 << 20;

/** The NAME in "/map import NAME", or undefined for any other text. */
export function mapImportName(text: string): string | undefined {
  const words = text.trim().split(/\s+/);
  if (
    words.length !== 3 ||
    words[0].toLowerCase() !== "/map" ||
    words[1].toLowerCase() !== "import"
  )
    return undefined;
  return words[2];
}

/** A download name for a map, safe on every system. */
export function mapFileName(name: string): string {
  return `${name.replace(/[/\\:*?"<>|]/g, "_") || "map"}.scabmap`;
}

/** Saves a map file through the browser's download. */
export function downloadMap(name: string, data: Uint8Array): void {
  const url = URL.createObjectURL(
    new Blob([data as BlobPart], { type: "application/octet-stream" }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = mapFileName(name);
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 10_000);
}

/** Asks for a .scabmap file; call while handling a tap or key so the browser allows it. */
export function pickMapFile(): Promise<Uint8Array | undefined> {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".scabmap";
  return new Promise((resolve) => {
    input.addEventListener("change", () => {
      const file = input.files?.[0];
      if (!file) return resolve(undefined);
      void file.arrayBuffer().then((b) => resolve(new Uint8Array(b)));
    });
    input.addEventListener("cancel", () => resolve(undefined));
    input.click();
  });
}
