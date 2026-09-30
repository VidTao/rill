import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

// Client side of the MCP connector OAuth flow (rill/bratrax/oauth*.go).
//
// Claude or ChatGPT sends the merchant to /bratrax/oauth/authorize, which parks
// the request and lands them on /oauth/consent?resume=<id> (via /login or
// /signup when they have no session). If their workspace hasn't finished
// onboarding, the consent page sends them through the normal funnel and
// remembers the resume id in localStorage; the root layout guard brings them
// back to the consent page once onboarding reaches `activating`.
//
// localStorage (not sessionStorage) because the funnel opens new tabs: Lemon
// Squeezy checkout returns to a fresh /payment-complete tab. The server is the
// authority either way: a stale hint just yields "this request has expired",
// and reconnecting from Claude starts a fresh one.

export const CONSENT_PATH = "/oauth/consent";
const RESUME_KEY = "bratrax_oauth_resume";

// Onboarding steps at which the merchant's own part is done; mirrors
// oauthReadySteps in rill/bratrax/oauth.go.
const USER_ONBOARDING_DONE = new Set([
  "activating",
  "compiling",
  "deploying",
  "extracting",
  "ready",
  "error",
]);

export interface OAuthWorkspace {
  client_id: string;
  company_name: string;
  step: string | null;
  ready: boolean;
}

export interface OAuthPendingView {
  status: "ready" | "needs_onboarding" | "no_workspace";
  client_name: string;
  redirect_host: string;
  source: string;
  email: string;
  role: string;
  step: string | null;
  default_client_id: string;
  workspaces: OAuthWorkspace[];
}

export interface OAuthPendingInfo {
  valid: boolean;
  client_name?: string;
  source?: string;
}

function baseUrl(): string {
  return get(runtime).host;
}

async function readError(res: Response): Promise<string> {
  try {
    const body = await res.json();
    if (body?.error) return String(body.error);
  } catch {
    // not JSON
  }
  return `Request failed (${res.status})`;
}

/** Loads the consent page's data. Also claims the request for this user. */
export async function getPendingAuthorization(
  resume: string,
): Promise<{ status: number; view?: OAuthPendingView; error?: string }> {
  const res = await fetch(
    `${baseUrl()}/bratrax/oauth/pending?resume=${encodeURIComponent(resume)}`,
    { credentials: "include" },
  );
  if (!res.ok) return { status: res.status, error: await readError(res) };
  return { status: res.status, view: (await res.json()) as OAuthPendingView };
}

/**
 * Public branding info for /login and /signup. Never throws: a failure only
 * costs the "Sign in to connect Claude" banner.
 */
export async function getPendingInfo(resume: string): Promise<OAuthPendingInfo> {
  try {
    const res = await fetch(
      `${baseUrl()}/bratrax/oauth/pending/info?resume=${encodeURIComponent(resume)}`,
      { credentials: "include" },
    );
    if (!res.ok) return { valid: false };
    return (await res.json()) as OAuthPendingInfo;
  } catch {
    return { valid: false };
  }
}

/** Approves or cancels; returns the URL to send the browser to. */
export async function decideAuthorization(
  resume: string,
  approve: boolean,
  clientId: string,
): Promise<string> {
  const res = await fetch(`${baseUrl()}/bratrax/oauth/decision`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ resume_id: resume, approve, client_id: clientId }),
  });
  if (!res.ok) throw new Error(await readError(res));
  const body = (await res.json()) as { redirect_to: string };
  return body.redirect_to;
}

/**
 * Extracts the resume id from a /login or /signup `redirect` param, but only
 * when it points at the consent page. Anything else is not ours to interpret.
 */
export function resumeIdFromRedirect(redirect: string | null): string | null {
  if (!redirect || !redirect.startsWith(CONSENT_PATH + "?")) return null;
  try {
    return new URL(redirect, "https://bratrax.invalid").searchParams.get("resume");
  } catch {
    return null;
  }
}

export function consentUrl(resume: string): string {
  return `${CONSENT_PATH}?resume=${encodeURIComponent(resume)}`;
}

export function isUserOnboardingDone(step: string | null | undefined): boolean {
  return !!step && USER_ONBOARDING_DONE.has(step);
}

export function saveResumeHint(resume: string): void {
  try {
    localStorage.setItem(RESUME_KEY, resume);
  } catch {
    // Storage blocked: reconnecting from Claude after onboarding still works.
  }
}

export function readResumeHint(): string | null {
  try {
    return localStorage.getItem(RESUME_KEY);
  } catch {
    return null;
  }
}

export function clearResumeHint(): void {
  try {
    localStorage.removeItem(RESUME_KEY);
  } catch {
    // ignore
  }
}
