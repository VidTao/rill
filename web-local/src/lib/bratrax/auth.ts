import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import {
  getEmbeddedToken,
  isShopifyEmbedded,
  setEmbeddedToken,
} from "./shopify-embed";

export interface BratraxUser {
  id: number;
  email: string;
  name: string;
  role: "super_admin" | "admin" | "viewer";
  project_id: string | null;
  // Bratrax client the user is bound to (null for super_admins who can switch).
  // Already serialized by the Go proxy's User struct; surfaced here so frontend
  // code can gate features per-client without an extra round trip.
  client_id?: string | null;
  // For super_admins: the most recent client they were active on, used to
  // resolve the active client when no bratrax_active_client cookie is set.
  last_client_id?: string | null;
  // For multi-store admins/viewers: the rill_multi_clients parent that owns
  // the user's sub-stores. NULL for legacy single-store users. When set, the
  // client switcher dropdown + "Add store" header button are rendered.
  multi_client_id?: string | null;
}

function getBaseUrl(): string {
  return get(runtime).host;
}

/**
 * Keep the JWT when we're inside the Shopify admin iframe.
 *
 * Both login and signup set bratrax_auth, but that cookie is SameSite=Lax and
 * is never sent from a cross-site frame — so in the embedded app a successful
 * login leaves the merchant looking logged out, and /login bounces them
 * straight back to itself.
 *
 * Done here rather than at each call site so no entry point can miss it:
 * /login, /signup, /accept-invite, /join and /shopify/connect all route
 * through these two functions.
 *
 * Skipped entirely when not embedded — a normal browser session has a working
 * cookie and does not need a second copy of the credential in sessionStorage.
 */
function rememberTokenIfEmbedded(token?: string): void {
  if (token && isShopifyEmbedded()) setEmbeddedToken(token);
}

/**
 * `token` is the same JWT that gets set as the bratrax_auth cookie. Normally
 * the cookie is all a caller needs and this is ignored — but inside the
 * Shopify admin iframe the cookie is SameSite=Lax and never sent, so the
 * embedded flow keeps this and passes it as `Authorization: Bearer`.
 * See lib/bratrax/shopify-embed.ts::setEmbeddedToken.
 */
export async function bratraxLogin(
  email: string,
  password: string,
): Promise<{ user: BratraxUser; token?: string }> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Login failed (${res.status})`);
  }
  const data = await res.json();
  rememberTokenIfEmbedded(data?.token);
  return data;
}

export async function bratraxSignup(
  email: string,
  password: string,
  companyName: string,
): Promise<{ user: BratraxUser }> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password, company_name: companyName }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Signup failed (${res.status})`);
  }
  const data = await res.json();
  rememberTokenIfEmbedded(data?.token);
  return data;
}

export async function bratraxLogout(): Promise<void> {
  await fetch(`${getBaseUrl()}/bratrax/auth/logout`, {
    method: "POST",
    credentials: "include",
  });
}

// Public, unauthenticated. Validate a store signup link before rendering the
// /join form. Returns { valid, store_platform, remaining } — valid:false for a
// full / expired / unknown token (no enumeration detail).
export interface StoreSignupLinkInfo {
  valid: boolean;
  store_platform?: "shopify" | "woocommerce";
  remaining?: number;
}

export async function getStoreSignupLink(
  token: string,
): Promise<StoreSignupLinkInfo> {
  const res = await fetch(`${getBaseUrl()}/bratrax/signup-link/${token}`);
  if (!res.ok) return { valid: false };
  return res.json();
}

// Public, unauthenticated. Atomically reserves one slot AND creates the account
// server-side (Flask) — the account is NOT created via /auth/signup because that
// path is gated by ONLY_INVITATION_LINK. The caller then logs in. On failure
// throws an Error whose message is a stable status string: "already_user" or
// "link_full_or_expired".
export async function consumeStoreSignupLink(
  token: string,
  email: string,
  password: string,
  name: string,
): Promise<{ store_platform: "shopify" | "woocommerce" }> {
  const res = await fetch(
    `${getBaseUrl()}/bratrax/signup-link/${token}/consume`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, name }),
    },
  );
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(
      body.status ?? body.error ?? `Request failed (${res.status})`,
    );
  }
  return body;
}

export interface BratraxAuthConfig {
  invite_only: boolean;
  // Gates the WooCommerce store connector on /onboard/store + /connectors.
  // Driven by the ALLOW_WOOCOMMERCE env flag on the Go proxy.
  allow_woocommerce: boolean;
}

// Public, unauthenticated. Used by /signup to decide whether to render the
// form or the invite-only message. Falls back to invite_only=false on any
// error so a transient failure doesn't lock new users out.
export async function getAuthConfig(
  fetchFn: typeof fetch = fetch,
): Promise<BratraxAuthConfig> {
  try {
    const res = await fetchFn(`${getBaseUrl()}/bratrax/auth/config`);
    if (!res.ok) return { invite_only: false, allow_woocommerce: false };
    return await res.json();
  } catch {
    return { invite_only: false, allow_woocommerce: false };
  }
}

// Public, unauthenticated. Submits an access request from the /signup
// invite-only screen. Always returns a known status string — the backend
// deliberately returns 2xx in all "already known" cases so the UI shows the
// same friendly message and doesn't leak which emails are already in the
// system.
export interface AccessRequestResult {
  status: "pending" | "already_pending" | "already_approved" | "already_user";
}

export async function requestAccess(
  email: string,
  firstName?: string,
): Promise<AccessRequestResult> {
  const res = await fetch(`${getBaseUrl()}/bratrax/access-requests`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, first_name: firstName ?? "" }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed (${res.status})`);
  }
  return res.json();
}

// Public, unauthenticated. Submits a self-serve "Try demo" request from the
// marketing landing page. Creates (or recognises) a viewer on the Bratrax Demo
// Account and returns a single-use hand-off URL that signs them in. Returns 2xx
// in all known cases (no enumeration oracle):
//   "ok"           — follow `handoff_url` to land in the demo, already signed in
//   "already_user" — the email belongs to a real account (point them at login)
//
// `handoff_url` must be followed with a full navigation, not a client-side
// goto(): it's a server redirect on the Go proxy whose whole job is to set the
// bratrax_auth cookie. The token is single-use and expires in ~2 minutes.
export interface DemoRequestResult {
  status: "ok" | "already_user";
  handoff_url?: string;
  already_had_account?: boolean;
}

export async function requestDemo(
  email: string,
  firstName?: string,
): Promise<DemoRequestResult> {
  const res = await fetch(`${getBaseUrl()}/bratrax/demo-request`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, first_name: firstName ?? "" }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed (${res.status})`);
  }
  return res.json();
}

export async function bratraxGetMe(
  fetchFn: typeof fetch = fetch,
): Promise<BratraxUser | null> {
  // The root layout calls this before rendering anything. Inside the Shopify
  // iframe the bratrax_auth cookie is SameSite=Lax and never sent, so without
  // the Bearer this always 401s, the layout treats the merchant as logged out,
  // and every navigation bounces to /login.
  const token = getEmbeddedToken();
  const res = await fetchFn(`${getBaseUrl()}/bratrax/auth/me`, {
    credentials: "include",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`Auth check failed (${res.status})`);
  const data: { user: BratraxUser } = await res.json();
  return data.user;
}

// ---------------------------------------------------------------------------
// Cross-client super_admin helpers (client switcher dropdown).
// listClients + switchClient back the dropdown in ApplicationHeader; the
// switch endpoint sets the bratrax_active_client cookie and persists
// last_client_id so the next login lands on the same client.
// ---------------------------------------------------------------------------

export interface BratraxClientSummary {
  client_id: string;
  company_name: string;
  admin_email?: string;
}

export interface BratraxClientList {
  clients: BratraxClientSummary[];
  active_client_id: string;
}

export async function bratraxListClients(): Promise<BratraxClientList> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/clients`, {
    credentials: "include",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `List clients failed (${res.status})`);
  }
  return res.json();
}

export async function bratraxSwitchClient(
  clientId: string,
): Promise<{ client_id: string; company_name: string; clickhouse_db: string }> {
  const res = await fetch(`${getBaseUrl()}/bratrax/auth/switch-client`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ client_id: clientId }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Switch client failed (${res.status})`);
  }
  return res.json();
}
