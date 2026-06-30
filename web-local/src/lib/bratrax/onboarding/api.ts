import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

function getBaseUrl(): string {
  return get(runtime).host;
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(`${getBaseUrl()}${path}`, {
    credentials: "include",
    ...init,
  });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
    } catch {
      // ignore parse errors
    }
    throw new Error(message);
  }
  return res.json();
}

export interface OnboardMeResult {
  client_id: string | null;
  company_name: string;
  clickhouse_db: string;
  shopify_embed_enabled: boolean;
  is_paid_subscriber: boolean;
  subscription_status: string | null;
  step: string;
  connected_platforms: PlatformConnection[];
  stack_selections: Record<string, unknown>;
  template_name: string | null;
  compile_status: string;
  deploy_status: string;
}

/**
 * Maps rill_onboarding_state.step to the onboarding route that should host
 * the user. Returns null for "ready" or unknown steps (no redirect needed).
 */
export function getOnboardResumeRoute(
  step: string | null | undefined,
): string | null {
  switch (step) {
    case "created":
      return "/onboard/store";
    case "payment_pending":
      return "/onboard/payment";
    case "embed_pending":
      return "/onboard/embed";
    case "platforms_connected":
      return "/onboard/stack";
    case "business_profile":
      return "/onboard/business";
    case "activating":
    case "compiling":
    case "deploying":
    case "extracting":
    case "error":
      return "/onboard/loading";
    default:
      return null;
  }
}

// Linear order of onboarding routes. The resume guard uses this to allow
// forward navigation (e.g. /onboard/stack → /onboard/business when the user
// clicks "Set up my analytics") while still bouncing backward jumps back to
// the user's current step.
//
// /onboard/payment is first because the LS paywall gates everything else —
// a user with step='payment_pending' can't reach /onboard/store until
// the subscription webhook flips them to step='created'.
//
// Adding a new onboard route requires THREE updates in lockstep:
//   1. Add a `case` to getOnboardResumeRoute() mapping the step name → route
//   2. Add the route to ONBOARD_ROUTE_ORDER below in its linear position
//   3. If it's a hard gate (user MUST be exactly here, no forward bypass),
//      add the step name to HARD_GATE_STEPS in routes/+layout.ts
// Missing #2 → infinite redirect loop. Missing #3 → URL-bar bypassable.
const ONBOARD_ROUTE_ORDER = [
  "/onboard/payment",
  "/onboard/store",
  "/onboard/embed",
  "/onboard/stack",
  "/onboard/business",
  "/onboard/loading",
];

export function getOnboardRouteIndex(pathname: string): number {
  return ONBOARD_ROUTE_ORDER.findIndex((r) => pathname.startsWith(r));
}

const CRED_KEY_TO_PLATFORM: Record<string, string> = {
  shopify_credentials: "shopify",
  woocommerce_credentials: "woocommerce",
  google_ads_credentials: "google_ads",
  facebook_ads_credentials: "facebook_ads",
  tiktok_ads_credentials: "tiktok_ads",
  klaviyo_credentials: "klaviyo",
  bing_ads_credentials: "bing_ads",
  pinterest_ads_credentials: "pinterest_ads",
  taboola_credentials: "taboola",
  outbrain_credentials: "outbrain",
  funnelish_credentials: "funnelish",
};

/**
 * Derives the list of connected platform IDs from the stack_selections
 * column. Each OAuth handler writes a `{platform}_credentials` key when
 * the user authorises that platform, so the presence of a key is the
 * authoritative "is this connected" signal.
 */
export function connectedFromStackSelections(
  stackSelections: Record<string, unknown> | null | undefined,
): string[] {
  if (!stackSelections) return [];
  return Object.keys(stackSelections)
    .map((k) => CRED_KEY_TO_PLATFORM[k])
    .filter((v): v is string => Boolean(v));
}

/**
 * Fetch the current user's client info from the backend.
 * Replaces fragile sessionStorage-based client_id passing.
 * Also caches in sessionStorage for instant access on subsequent pages.
 */
export async function onboardMe(): Promise<OnboardMeResult | null> {
  try {
    const result = await apiFetch<OnboardMeResult>("/bratrax/onboard/me");
    if (result.client_id) {
      sessionStorage.setItem("onboard_client_id", result.client_id);
    }
    return result;
  } catch {
    return null;
  }
}

export interface OnboardStartResult {
  status: string;
  client_id: string;
  client_name: string;
  clickhouse_db: string;
}

export interface OnboardConnectResult {
  status: string;
  connected_platforms: PlatformConnection[];
}

export interface PlatformConnection {
  platform: string;
  account_id: string;
  account_name: string;
  connected_at: string;
}

export interface OnboardActivateResult {
  status: string;
  client_id: string;
  template: string;
}

export interface OnboardStatus {
  client_id: string;
  step: string;
  connected_platforms: PlatformConnection[];
  stack_selections: Record<string, unknown>;
  template_name: string | null;
  compile_status: string;
  deploy_status: string;
  extraction_status: Record<string, unknown>;
  error_message: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export async function onboardStart(
  companyName: string,
  options: { requiresPayment?: boolean; isMultiStore?: boolean } = {},
): Promise<OnboardStartResult> {
  // requires_payment defaults true on the backend, so we only send the field
  // explicitly when the caller wants to opt out of the LS paywall (inceptly
  // invite-link signups). Public /signup omits it → backend defaults TRUE
  // → step machine starts at payment_pending.
  // is_multi_store defaults false on the backend; only set when the invite
  // carried the multi-store toggle so we create a rill_multi_clients parent.
  const body: Record<string, unknown> = { company_name: companyName };
  if (options.requiresPayment === false) {
    body.requires_payment = false;
  }
  if (options.isMultiStore === true) {
    body.is_multi_store = true;
  }
  return apiFetch<OnboardStartResult>("/bratrax/onboard/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// Public, unauthenticated. Called on blur of the company-name input on
// /signup and /accept-invite (kind='signup'). Tells the frontend whether
// the slug derived from the user's company name collides with an existing
// rill_clients row, before the user submits and discovers the collision
// the hard way during onboard_start.
export interface CompanyCheckResult {
  available: boolean;
  reason: "ok" | "taken" | "invalid" | "empty" | "error";
  normalized: string;
}

export async function checkCompanyAvailable(
  name: string,
): Promise<CompanyCheckResult> {
  return apiFetch<CompanyCheckResult>(
    `/bratrax/onboard/check-company?name=${encodeURIComponent(name)}`,
  );
}

export async function onboardConnect(
  clientId: string,
  platform: string,
  accountIdOrExtra?: string | Record<string, unknown>,
  accountName?: string,
): Promise<OnboardConnectResult> {
  // Backwards-compat overload: legacy callers pass (clientId, platform,
  // accountId, accountName). New callers (snippet-install platforms with no
  // account ID) pass an `extra` bag like { builder: "funnelish" } in the
  // third slot; the backend merges these into stack_selections so the
  // builder choice survives a refresh.
  let accountId = "";
  let extra: Record<string, unknown> | undefined;
  if (typeof accountIdOrExtra === "string" || accountIdOrExtra == null) {
    accountId = accountIdOrExtra ?? "";
  } else {
    extra = accountIdOrExtra;
  }
  return apiFetch<OnboardConnectResult>("/bratrax/onboard/connect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_id: clientId,
      platform,
      account_id: accountId,
      account_name: accountName ?? "",
      ...(extra ? { extra } : {}),
    }),
  });
}

export async function onboardActivate(
  clientId: string,
  stackSelections?: Record<string, unknown>,
): Promise<OnboardActivateResult> {
  return apiFetch<OnboardActivateResult>("/bratrax/onboard/activate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_id: clientId,
      stack_selections: stackSelections ?? {},
    }),
  });
}

export async function onboardStatus(clientId: string): Promise<OnboardStatus> {
  return apiFetch<OnboardStatus>(
    `/bratrax/onboard/status?client_id=${encodeURIComponent(clientId)}`,
  );
}

export async function onboardBusinessProfile(
  clientId: string,
  businessProfile: Record<string, string>,
  stackSelections?: Record<string, unknown>,
): Promise<OnboardActivateResult> {
  return apiFetch<OnboardActivateResult>("/bratrax/onboard/business-profile", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_id: clientId,
      business_profile: businessProfile,
      stack_selections: stackSelections ?? {},
    }),
  });
}

export async function onboardDisconnect(
  clientId: string,
  platform: string,
): Promise<{ status: string; platform: string }> {
  return apiFetch<{ status: string; platform: string }>(
    "/bratrax/onboard/disconnect",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ client_id: clientId, platform }),
    },
  );
}

export interface EmbedStatusResult {
  enabled: boolean;
  shop: string | null;
  reason?:
    | "enabled"
    | "not_present"
    | "no_active_theme"
    | "no_settings_data"
    | "invalid_settings_json"
    | "auth_error"
    | "api_error"
    | "no_credentials";
  debug?: Record<string, unknown>;
}

/**
 * Poll the Shopify Asset API (server-side) to check whether the merchant
 * has toggled on the Bratrax Theme App Embed. On detection, the backend
 * flips rill_clients.shopify_embed_enabled and advances the step machine
 * from 'embed_pending' to 'platforms_connected'.
 */
export async function verifyEmbedStatus(
  clientId: string,
): Promise<EmbedStatusResult> {
  return apiFetch<EmbedStatusResult>(
    `/bratrax/onboard/embed-status?client_id=${encodeURIComponent(clientId)}`,
  );
}

// -----------------------------------------------------------------------------
// WooCommerce tracking plugin: download + install verification
//
// The generic WordPress plugin (extensions/bratrax-woocommerce) is downloaded
// from the backend, installed on the merchant's store, and connected via the
// signed "Connect to Bratrax" handshake (which runs inside WP admin, not here).
// Onboarding confirms the plugin is live by polling verify-install, which counts
// recent source='browser' activity_stream events for the client.
// -----------------------------------------------------------------------------

export interface WooVerifyInstallResult {
  client_id: string;
  events_seen: number;
  by_type: Record<string, number>;
  connected: boolean;
  lookback_minutes: number;
}

/** Absolute URL for the generic plugin zip (public download). */
export function wooPluginDownloadUrl(): string {
  return `${getBaseUrl()}/bratrax/onboard/woocommerce/plugin.zip`;
}

/** Poll for recent browser events to confirm the tracking plugin is sending data. */
export async function verifyWooInstall(
  clientId: string,
): Promise<WooVerifyInstallResult> {
  return apiFetch<WooVerifyInstallResult>(
    `/bratrax/onboard/woocommerce/verify-install?client_id=${encodeURIComponent(clientId)}`,
  );
}

// -----------------------------------------------------------------------------
// OAuth public-client-IDs config (BUG-01)
//
// These values are PUBLIC by OAuth design — they're in every redirect URL
// during signup. Fetching them at runtime (instead of hardcoding in source)
// lets us rotate the Facebook app or move to a staging env without rebuilding
// the frontend.
// -----------------------------------------------------------------------------

export interface OAuthConfig {
  fb_app_id: string;
  google_client_id: string;
  bing_ads_client_id: string;
}

export function getOAuthConfig(): Promise<OAuthConfig> {
  return apiFetch<OAuthConfig>("/bratrax/onboard/oauth-config");
}

// ----- Lemon Squeezy paywall (C20+ paying-customer onboarding) ---------------
//
// Flow: onboard_start sets step='payment_pending' for unpaid clients →
// layout's getOnboardResumeRoute redirects them to /onboard/payment → page
// calls getPaymentCheckoutUrl() → window.location = url. After LS payment,
// LS redirects back to /onboard/payment?return=1, page polls
// getPaymentStatus() until is_paid=true (webhook confirms it server-side),
// then goto /onboard/store.

export interface PaymentCheckoutResult {
  checkout_url?: string;
  already_paid?: boolean;
  error?: string;
}

export async function getPaymentCheckoutUrl(): Promise<PaymentCheckoutResult> {
  return apiFetch<PaymentCheckoutResult>("/bratrax/onboard/payment/checkout", {
    method: "POST",
  });
}

export interface PaymentStatusResult {
  is_paid: boolean;
  subscription_status: string | null;
}

export async function getPaymentStatus(): Promise<PaymentStatusResult> {
  return apiFetch<PaymentStatusResult>("/bratrax/onboard/payment/status");
}
