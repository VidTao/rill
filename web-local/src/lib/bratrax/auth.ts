import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

export interface BratraxUser {
  id: number;
  email: string;
  name: string;
  role: "super_admin" | "admin" | "viewer";
  project_id: string | null;
}

function getBaseUrl(): string {
  return get(runtime).host;
}

export async function bratraxLogin(
  email: string,
  password: string,
): Promise<{ user: BratraxUser }> {
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
  return res.json();
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
  return res.json();
}

export async function bratraxLogout(): Promise<void> {
  await fetch(`${getBaseUrl()}/bratrax/auth/logout`, {
    method: "POST",
    credentials: "include",
  });
}

export interface BratraxAuthConfig {
  invite_only: boolean;
}

// Public, unauthenticated. Used by /signup to decide whether to render the
// form or the invite-only message. Falls back to invite_only=false on any
// error so a transient failure doesn't lock new users out.
export async function getAuthConfig(
  fetchFn: typeof fetch = fetch,
): Promise<BratraxAuthConfig> {
  try {
    const res = await fetchFn(`${getBaseUrl()}/bratrax/auth/config`);
    if (!res.ok) return { invite_only: false };
    return await res.json();
  } catch {
    return { invite_only: false };
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

export async function requestAccess(email: string): Promise<AccessRequestResult> {
  const res = await fetch(`${getBaseUrl()}/bratrax/access-requests`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
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
  const res = await fetchFn(`${getBaseUrl()}/bratrax/auth/me`, {
    credentials: "include",
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
