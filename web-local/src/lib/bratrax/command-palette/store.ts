import { writable } from "svelte/store";

/** Open state of the ⌘K palette, shared by the palette and its header trigger. */
export const commandPaletteOpen = writable(false);

// ⌘ on macOS, Ctrl elsewhere. Mac Ctrl+K is left alone on purpose: it's the
// native "delete to end of line" in every text field. Same UA sniff as
// web-common's MetaKey.svelte.
export const isMac =
  typeof navigator !== "undefined" && navigator.userAgent.includes("Macintosh");

export const shortcutLabel = isMac ? "⌘K" : "Ctrl K";
