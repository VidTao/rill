import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

function getBaseUrl(): string {
  return get(runtime).host;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
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
      return "/onboard/shopify";
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

const CRED_KEY_TO_PLATFORM: Record<string, string> = {
  shopify_credentials: "shopify",
  google_ads_credentials: "google_ads",
  facebook_ads_credentials: "facebook_ads",
  tiktok_ads_credentials: "tiktok_ads",
  klaviyo_credentials: "klaviyo",
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
  options: { requiresPayment?: boolean } = {},
): Promise<OnboardStartResult> {
  // requires_payment defaults true on the backend, so we only send the field
  // explicitly when the caller wants to opt out of the LS paywall (inceptly
  // invite-link signups). Public /signup omits it → backend defaults TRUE
  // → step machine starts at payment_pending.
  const body: Record<string, unknown> = { company_name: companyName };
  if (options.requiresPayment === false) {
    body.requires_payment = false;
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
  accountId?: string,
  accountName?: string,
): Promise<OnboardConnectResult> {
  return apiFetch<OnboardConnectResult>("/bratrax/onboard/connect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_id: clientId,
      platform,
      account_id: accountId ?? "",
      account_name: accountName ?? "",
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

export async function onboardStatus(
  clientId: string,
): Promise<OnboardStatus> {
  return apiFetch<OnboardStatus>(
    `/bratrax/onboard/status?client_id=${encodeURIComponent(clientId)}`,
  );
}

export async function onboardBusinessProfile(
  clientId: string,
  businessProfile: Record<string, string>,
  stackSelections?: Record<string, unknown>,
): Promise<OnboardActivateResult> {
  return apiFetch<OnboardActivateResult>(
    "/bratrax/onboard/business-profile",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        client_id: clientId,
        business_profile: businessProfile,
        stack_selections: stackSelections ?? {},
      }),
    },
  );
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
// then goto /onboard/shopify.

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
