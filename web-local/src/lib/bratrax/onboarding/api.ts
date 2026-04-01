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
  step: string;
  connected_platforms: PlatformConnection[];
  template_name: string | null;
  compile_status: string;
  deploy_status: string;
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
): Promise<OnboardStartResult> {
  return apiFetch<OnboardStartResult>("/bratrax/onboard/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ company_name: companyName }),
  });
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
