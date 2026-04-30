import { get } from "svelte/store";
import { themeControl } from "@rilldata/web-common/features/themes/theme-control";

/**
 * Bratrax theme — thin wrapper around Rill's upstream `themeControl` singleton
 * (web-common/src/features/themes/theme-control.ts). Rill is the source of
 * truth: it sets/removes the `.dark` class on <html>, persists the preference
 * to localStorage["rill:theme"], and drives both Rill's semantic-token flips
 * (logo swap, web-common/src/app.css) AND our bratrax-theme.css overrides
 * (which use :root.dark selectors).
 *
 * Having two parallel theme systems caused the bug where the new sun/moon
 * toggle flipped some tokens but not the logo (and vice-versa for the old
 * theme button). This module ensures both surfaces speak through the same
 * underlying control.
 */

export type Theme = "light" | "dark";

/** Reactive store of the current theme. Reads from themeControl.current. */
export const theme = themeControl;

/** Flip the theme. Updates the .dark class on <html> + localStorage. */
export function toggleTheme(): void {
  if (get(theme) === "dark") {
    themeControl.set.light();
  } else {
    themeControl.set.dark();
  }
}
