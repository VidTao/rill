/**
 * Google Analytics 4 (gtag.js) event helpers.
 * Spec: bratrax/docs/ANALYTICS_IMPLEMENTATION.md.
 *
 * The tag itself lives in web-local/src/app.html so it is live before any Svelte
 * code runs, and it self-suppresses off *.bratrax.com. window.gtag is therefore
 * legitimately absent in dev, under Playwright, and for anyone running an ad
 * blocker — every function here no-ops silently in that case. GA4 is
 * fire-and-forget: we never retry. Under-counting beats over-counting.
 *
 * Deliberately NOT built on Rill's behaviourEvent/metricsService (its schema is
 * locked to Rill's own domain — medium/space/screen_name/entity_id, no signup
 * action — and it transports to /local/track behind Rill's intake credentials)
 * nor on PostHog (which points at an unimplemented /ph proxy). Both are inert
 * in this fork.
 */

type GtagParams = Record<string, string | number | boolean>;

declare global {
  interface Window {
    gtag?: (...args: unknown[]) => void;
  }
}

export function trackEvent(name: string, params?: GtagParams): void {
  try {
    window.gtag?.("event", name, params ?? {});
  } catch {
    // Analytics must never break a user flow.
  }
}

/**
 * Fire at most once per browser per guard key. Same localStorage idiom as
 * autoMarkOnce() in lib/bratrax/onboarding/checklist.ts. Guard keys must derive
 * from a stable per-account id (user.id, client_id), never from page or session.
 *
 * The gtag presence check comes FIRST: if the tag is absent there is nothing to
 * dedupe, and burning the guard would permanently suppress a later legitimate
 * send (an ad-blocked user who returns with the blocker off).
 */
export function trackOnce(
  guardKey: string,
  name: string,
  params?: GtagParams,
): void {
  if (typeof window.gtag !== "function") return;
  const key = `bratrax_ga_${guardKey}`;
  try {
    if (localStorage.getItem(key)) return;
    localStorage.setItem(key, "1");
  } catch {
    // No storage → no dedupe available. Fire anyway; every call site is also
    // structurally single-shot (see their comments).
  }
  trackEvent(name, params);
}

function sourcesKey(clientId: string): string {
  return `bratrax_ga_sources_${clientId}`;
}

/**
 * Record "this client starts with zero connected sources", so the first real
 * connection is never mistaken for a pre-existing one. Called at onboard_start,
 * where we know the client was just created.
 */
export function seedConnectedSourcesBaseline(clientId: string): void {
  try {
    const key = sourcesKey(clientId);
    if (!localStorage.getItem(key)) localStorage.setItem(key, "[]");
  } catch {
    // ignore — a missing baseline just means the first observation is silent.
  }
}

/**
 * Diff the server's connected-platform list against the last list observed for
 * this client and fire `data_source_connected` for anything new.
 *
 * Why a diff rather than a hook per connect path: connections land via ~6
 * mechanisms (the Shopify OAuth callback page, popup OAuth + the account
 * selection modal, four full-page redirect-OAuth returns, WooCommerce's
 * server-to-server callback, Funnelish's snippet modal). All of them terminate
 * in a refreshFromServer(), so one diff placed there is inherited by future
 * connectors for free.
 *
 * The stored set is the UNION of seen + current, so disconnect → reconnect does
 * not re-fire. A missing baseline means "first observation": record it and stay
 * silent, so an established workspace never retro-emits one event per source it
 * connected months ago.
 */
export function trackConnectedSources(
  clientId: string,
  platforms: string[],
): void {
  if (typeof window.gtag !== "function" || !clientId) return;
  const key = sourcesKey(clientId);
  let seen: string[] | null;
  try {
    const raw = localStorage.getItem(key);
    seen = raw ? (JSON.parse(raw) as string[]) : null;
    localStorage.setItem(
      key,
      JSON.stringify(Array.from(new Set([...(seen ?? []), ...platforms]))),
    );
  } catch {
    return; // No reliable dedupe → send nothing rather than risk a flood.
  }
  if (seen === null) return; // Baseline only.
  for (const platform of platforms) {
    if (!seen.includes(platform)) {
      trackEvent("data_source_connected", { source: platform });
    }
  }
}
