import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type {
  AdvertisingConnection,
  CrmConnection,
  PlatformConnection,
} from "./types";

function getBaseUrl(): string {
  return get(runtime).host;
}

async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(`${getBaseUrl()}${path}`, {
    credentials: "include",
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error?: string }).error ?? `Request failed (${res.status})`,
    );
  }
  return res.json() as Promise<T>;
}

// ── Generic connector endpoints ──

export async function getAuthUrl(
  apiSlug: string,
  params?: Record<string, string>,
): Promise<string> {
  const qs = params ? "?" + new URLSearchParams(params).toString() : "";
  const data = await apiFetch<{ url: string }>(
    `/bratrax/connectors/${apiSlug}/auth-url${qs}`,
  );
  return data.url;
}

export async function exchangeToken(
  apiSlug: string,
  body: Record<string, string>,
): Promise<unknown> {
  return apiFetch(`/bratrax/connectors/${apiSlug}/insertTokensData`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// ── Shopify-specific ──

export async function getShopifyAuthUrl(shop: string): Promise<string> {
  const data = await apiFetch<{ url: string }>(
    `/bratrax/connectors/shopify/shop-auth-url?shop=${encodeURIComponent(shop)}`,
  );
  return data.url;
}

export async function exchangeShopifyTokens(
  code: string,
  shop: string,
): Promise<unknown> {
  return apiFetch(`/bratrax/connectors/shopify/insertTokensData`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, shop }),
  });
}

// ── Facebook-specific (client SDK flow) ──

export interface AdAccountInfo {
  accountId: string;
  accountName: string;
}

export async function exchangeFacebookTokens(
  shortLivedToken: string,
  email: string,
  selectedAccounts: AdAccountInfo[],
): Promise<unknown> {
  return apiFetch(`/bratrax/connectors/facebook/insertTokensData`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      short_lived_token: shortLivedToken,
      email,
      selectedAccounts,
    }),
  });
}

// ── Google-specific (auth code flow) ──

export interface GoogleAdAccount {
  customer_id: string;
  manager_account_id: string;
  name: string;
  currency_code: string;
  time_zone: string;
  children?: GoogleAdAccount[];
}

export async function getGoogleAdAccounts(
  code: string,
): Promise<GoogleAdAccount[]> {
  const data = await apiFetch<GoogleAdAccount[]>(
    `/bratrax/connectors/google/adAccounts?code=${encodeURIComponent(code)}`,
  );
  return data;
}

export async function exchangeGoogleTokens(
  code: string,
  selectedAccounts: AdAccountInfo[],
): Promise<unknown> {
  return apiFetch(`/bratrax/connectors/google/insertTokensData`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, selectedAccounts }),
  });
}

// ── Connection fetchers ──

interface ConnectionsResponse<T> {
  success: boolean;
  data: T;
}

export async function getAllPlatformConnections(): Promise<
  Record<string, PlatformConnection>
> {
  const res = await apiFetch<
    ConnectionsResponse<Record<string, PlatformConnection>>
  >("/bratrax/users/get-all-platform-connections");
  return res.data;
}

export async function getAllAdConnections(): Promise<
  Record<string, AdvertisingConnection[]>
> {
  const res = await apiFetch<
    ConnectionsResponse<Record<string, AdvertisingConnection[]>>
  >("/bratrax/users/get-all-ad-connections");
  return res.data;
}

export async function getAllCrmConnections(): Promise<
  Record<string, CrmConnection[]>
> {
  const res = await apiFetch<
    ConnectionsResponse<Record<string, CrmConnection[]>>
  >("/bratrax/users/get-all-crm-connections");
  return res.data;
}
