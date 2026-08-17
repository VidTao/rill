/**
 * Shopify embedded-context detection and the break-out helper.
 *
 * When Bratrax is opened from inside the Shopify admin, the whole app runs in
 * an iframe on admin.shopify.com. Two things behave differently there:
 *
 *   1. Anything that must be top-level — Lemon Squeezy checkout, "Open in
 *      Bratrax" — has to open a NEW tab. LS cannot render in a nested iframe
 *      (payment providers set frame-ancestors against exactly that), and 3-D
 *      Secure breaks in frames even where framing is allowed.
 *   2. Onboarding finishes on /embed/canvas/campaign_deep_dive rather than the
 *      full app shell.
 *
 * Detection is sticky: the flag is written to sessionStorage on first
 * observation so it survives the OAuth and checkout round-trips, which can
 * momentarily land in a context where the frame check alone would say "no".
 */

const EMBED_FLAG = "bratrax_shopify_embedded";

/** True when this document is running inside a frame we didn't create. */
function inIframe(): boolean {
  try {
    return window.top !== window.self;
  } catch {
    // Cross-origin parent throws on access — which itself means we're framed.
    return true;
  }
}

/**
 * Record that this session is running embedded. Called from the entry points
 * Shopify sends merchants to (they carry `host` / `embedded=1`).
 */
export function markShopifyEmbedded(): void {
  try {
    sessionStorage.setItem(EMBED_FLAG, "1");
  } catch {
    // Private mode / storage disabled — fall back to the live frame check.
  }
}

/**
 * True when the app should behave as an embedded Shopify app.
 *
 * Checks the sticky flag first, then Shopify's own URL params, then the frame
 * position. The param check also sets the flag, so a single embedded entry is
 * enough for the rest of the session.
 */
export function isShopifyEmbedded(): boolean {
  if (typeof window === "undefined") return false;
  try {
    if (sessionStorage.getItem(EMBED_FLAG) === "1") return true;
  } catch {
    /* ignore */
  }

  const params = new URLSearchParams(window.location.search);
  if (params.get("embedded") === "1" || params.has("host")) {
    markShopifyEmbedded();
    return true;
  }

  // Bare frame check last: it's true for any framing, so on its own it would
  // also fire for unrelated embeds of our dashboards.
  return inIframe() && params.has("shop");
}

/**
 * Open a URL in a new top-level tab, breaking out of the Shopify iframe.
 *
 * Only safe to call SYNCHRONOUSLY inside a click handler. If you need to fetch
 * something first, use reserveTopLevelTab() — see the note there.
 *
 * Returns false when the popup was blocked, so the caller can render a plain
 * clickable link instead of failing silently — losing a merchant at the
 * checkout step because a popup blocker ate the window is not acceptable.
 */
export function openTopLevel(url: string): boolean {
  try {
    const win = window.open(url, "_blank", "noopener,noreferrer");
    if (win) return true;
  } catch {
    /* fall through */
  }
  return false;
}

/**
 * Replace the top-level window, rather than opening a tab beside it.
 *
 * Use this for destinations that live inside the Shopify admin itself — the
 * App Pricing plan-selection page being the one that matters. A new tab would
 * render the admin a second time next to the one the merchant came from, and
 * loading it in the iframe would nest the admin inside itself, which Shopify
 * refuses to be framed into.
 *
 * Like openTopLevel this must be called synchronously from a click handler:
 * a cross-origin child may only navigate its top window with user activation,
 * which does not survive an await.
 *
 * Falls back to navigating this frame when `window.top` is unreachable. That
 * only happens outside an iframe, where the two are the same window anyway.
 */
export function navigateTopLevel(url: string): void {
  try {
    if (window.top) {
      window.top.location.href = url;
      return;
    }
  } catch {
    /* cross-origin access denied — fall through */
  }
  window.location.href = url;
}

// --------------------------------------------------------------------------
// Embedded bearer token
// --------------------------------------------------------------------------
// The bratrax_auth cookie is SameSite=Lax, so it is not sent from inside the
// Shopify admin iframe. That breaks the account-creation flow specifically:
// onboard_start needs an authenticated caller, and a Shopify session token
// cannot stand in for one — it resolves shop -> client, and the client is
// exactly what onboard_start is there to create. Chicken and egg.
//
// POST /bratrax/auth/login already returns the JWT in its body, and the Go
// middleware already accepts it as `Authorization: Bearer` (a fallback added
// for popup OAuth flows "where cookies may not be sent due to browser
// restrictions" — the same problem). So in the iframe we keep the token and
// send it explicitly.
//
// sessionStorage, not localStorage: scoped to this tab, gone when it closes.

const TOKEN_KEY = "bratrax_embedded_token";

/** Remember the JWT from a login performed inside the iframe. */
export function setEmbeddedToken(token: string | undefined | null): void {
  if (!token) return;
  try {
    sessionStorage.setItem(TOKEN_KEY, token);
  } catch {
    /* storage unavailable — requests fall back to the cookie */
  }
}

export function getEmbeddedToken(): string | null {
  try {
    return sessionStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function clearEmbeddedToken(): void {
  try {
    sessionStorage.removeItem(TOKEN_KEY);
  } catch {
    /* ignore */
  }
}

/** Handle to a tab opened ahead of knowing its destination. */
export interface ReservedTab {
  /** False when the popup was blocked and nothing was opened. */
  readonly ok: boolean;
  /** Send the reserved tab to `url`. No-op when the popup was blocked. */
  navigate(url: string): void;
  /** Close the reserved tab — call this if the URL never arrives. */
  close(): void;
}

/**
 * Reserve a blank top-level tab NOW, to navigate once a URL is available.
 *
 * Browsers grant `window.open` off a user gesture, and that grant does not
 * survive an `await`. Both places we break out of the iframe — Lemon Squeezy
 * checkout and "Open in Bratrax" — have to call the backend first, so opening
 * afterwards is blocked essentially every time rather than occasionally. The
 * fallback would become the normal path.
 *
 * So: open synchronously on the click, then point the tab at the URL when it
 * arrives. Deliberately no `noopener` — that makes window.open return null in
 * several browsers, and we need the handle. `opener` is nulled after
 * navigating, which gets the same protection.
 */
export function reserveTopLevelTab(): ReservedTab {
  let win: Window | null = null;
  try {
    win = window.open("about:blank", "_blank");
  } catch {
    win = null;
  }

  return {
    ok: !!win,
    navigate(url: string) {
      if (!win) return;
      try {
        win.opener = null;
        win.location.replace(url);
      } catch {
        /* tab closed by the user mid-flight */
      }
    },
    close() {
      try {
        win?.close();
      } catch {
        /* already gone */
      }
    },
  };
}

/**
 * Deep link back into this app inside the Shopify admin, e.g. from the tab
 * that Lemon Squeezy redirected to after checkout.
 *
 * `shop` is the myshopify domain; the admin path uses the store handle (the
 * part before .myshopify.com).
 */
export function shopifyAdminAppUrl(shop: string, appHandle: string): string {
  const handle = shop.replace(/\.myshopify\.com$/i, "");
  return `https://admin.shopify.com/store/${encodeURIComponent(handle)}/apps/${encodeURIComponent(appHandle)}`;
}
