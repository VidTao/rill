/**
 * Shopify App Bridge loader and session-token accessor.
 *
 * App Bridge is what mints the short-lived JWT that authenticates us inside
 * the Shopify admin iframe. The Go proxy verifies it (bratrax/shopify_session.go)
 * and resolves shop -> client -> user, so an embedded request ends up with the
 * exact same X-Bratrax-* headers a cookie session produces.
 *
 * Loaded LAZILY and only when embedded, rather than as a tag in app.html.
 * app.html is served to every visitor including the marketing pages, and
 * pulling a third-party script into all of them to serve the small fraction of
 * sessions that run inside Shopify is a cost — and a failure mode — the rest of
 * the app shouldn't carry. Non-embedded sessions never touch this module.
 *
 * The API key is hardcoded for the same reason the GA4 measurement ID is: there
 * is no working env-plumbing route into the frontend here. SvelteKit's env
 * placeholders only accept a PUBLIC_ prefix and hard-error otherwise,
 * `import.meta.env` isn't available in app.html, and the rill repo has no .env
 * while adapter-static substitutes at build time. It is a public client id, not
 * a secret — it appears in every OAuth URL already.
 */

import { isShopifyEmbedded } from "./shopify-embed";

/** Shopify app client_id. Matches `client_id` in bratrax/shopify.app.toml. */
const SHOPIFY_API_KEY = "b1afe70db24dd0e683b49ec1f5ccd325";

const APP_BRIDGE_SRC = "https://cdn.shopify.com/shopifycloud/app-bridge.js";

declare global {
  interface Window {
    shopify?: { idToken?: () => Promise<string> };
  }
}

let loadPromise: Promise<boolean> | null = null;

/**
 * Inject App Bridge once. Resolves true when `window.shopify.idToken` is
 * usable, false on any failure — callers fall back to cookie auth rather than
 * breaking, which is the right behaviour if we somehow mis-detected the
 * embedded context.
 */
function loadAppBridge(): Promise<boolean> {
  if (loadPromise) return loadPromise;

  loadPromise = new Promise<boolean>((resolve) => {
    if (typeof document === "undefined") return resolve(false);
    if (typeof window.shopify?.idToken === "function") return resolve(true);

    const script = document.createElement("script");
    script.src = APP_BRIDGE_SRC;
    script.setAttribute("data-api-key", SHOPIFY_API_KEY);
    script.onload = () =>
      resolve(typeof window.shopify?.idToken === "function");
    script.onerror = () => {
      console.warn(
        "[bratrax] App Bridge failed to load; falling back to cookie auth",
      );
      resolve(false);
    };
    document.head.appendChild(script);
  });

  return loadPromise;
}

/**
 * A fresh Shopify session token, or null when unavailable.
 *
 * Not cached: these expire in about a minute, and App Bridge mints a new one
 * per call cheaply. Caching would trade a negligible saving for intermittent
 * 401s that are miserable to debug.
 */
export async function getShopifySessionToken(): Promise<string | null> {
  if (!isShopifyEmbedded()) return null;
  try {
    if (!(await loadAppBridge())) return null;
    const token = await window.shopify?.idToken?.();
    return token || null;
  } catch (e) {
    console.warn("[bratrax] session token unavailable", e);
    return null;
  }
}

/**
 * Authorization header for an embedded request, or an empty object.
 *
 * Spread into a fetch init. Returns nothing when not embedded, so the cookie
 * path is completely untouched for every existing user.
 */
export async function shopifyAuthHeader(): Promise<Record<string, string>> {
  const token = await getShopifySessionToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
