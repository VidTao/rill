/** Cost settings API helpers — mirrors Flask /cost-settings endpoints. */

import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import type {
  ProductCogs,
  MarketplaceProductCogs,
  GatewayFee,
  ShippingProfile,
  MediaSpendScopeAccountOption,
  ExpenseRule,
  MediaSpendScopeGuidance,
  MediaSpendScopeRule,
  MediaSpendScopeRuleData,
  StoreSettings,
  ProfitReadiness,
} from "./types";

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
      if (body.message) message = body.message;
    } catch {
      /* use default message */
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

// --- Store Settings ---

export async function getStoreSettings(): Promise<StoreSettings> {
  const res = await apiFetch<{ data: StoreSettings }>(
    "/bratrax/cost-settings/settings",
  );
  return res.data;
}

export async function saveStoreSettings(
  settings: Record<string, unknown>,
): Promise<void> {
  await apiFetch("/bratrax/cost-settings/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ settings }),
  });
}

export async function getProfitReadiness(): Promise<ProfitReadiness> {
  const res = await apiFetch<{ data: ProfitReadiness }>(
    "/bratrax/cost-settings/profit-readiness",
  );
  return res.data;
}

// --- Product COGS ---

export async function getProductsCogs(): Promise<ProductCogs[]> {
  const res = await apiFetch<{ data: ProductCogs[] }>(
    "/bratrax/cost-settings/products-cogs",
  );
  return res.data;
}

export async function saveProductsCogs(
  products: Array<{
    product_id: string;
    cogs_amount: number | null;
    handling_fee: number | null;
    title?: string;
    sku?: string;
  }>,
): Promise<void> {
  await apiFetch("/bratrax/cost-settings/products-cogs", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ products }),
  });
}

// --- Marketplace Product COGS ---

export async function getMarketplaceProductsCogs(): Promise<
  MarketplaceProductCogs[]
> {
  const res = await apiFetch<{ data: MarketplaceProductCogs[] }>(
    "/bratrax/cost-settings/marketplace-products-cogs?platform=amazon&marketplace_id=ATVPDKIKX0DER",
  );
  return res.data;
}

export async function saveMarketplaceProductsCogs(
  products: Array<{
    platform: "amazon";
    marketplace_id: string;
    asin: string;
    seller_sku: string;
    title?: string;
    unit_cost: number | null;
    handling_cost: number | null;
  }>,
): Promise<void> {
  await apiFetch("/bratrax/cost-settings/marketplace-products-cogs", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ products }),
  });
}

// --- Gateway Fees ---

export async function getGatewayFees(): Promise<GatewayFee[]> {
  const res = await apiFetch<{ data: GatewayFee[] }>(
    "/bratrax/cost-settings/gateway-fees",
  );
  return res.data;
}

export async function saveGatewayFee(
  gatewayName: string,
  percentageFee: number,
  fixedFee: number,
): Promise<void> {
  await apiFetch(
    `/bratrax/cost-settings/gateway-fees/${encodeURIComponent(gatewayName)}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        percentage_fee: percentageFee,
        fixed_fee: fixedFee,
      }),
    },
  );
}

// --- Shipping Profiles ---

export async function getShippingProfiles(): Promise<ShippingProfile[]> {
  const res = await apiFetch<{ data: ShippingProfile[] }>(
    "/bratrax/cost-settings/shipping-profiles",
  );
  return res.data;
}

export async function createShippingProfile(
  profile: ShippingProfile["data"],
): Promise<string> {
  const res = await apiFetch<{ profile_id: string }>(
    "/bratrax/cost-settings/shipping-profiles",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(profile),
    },
  );
  return res.profile_id;
}

export async function updateShippingProfile(
  profileId: string,
  profile: ShippingProfile["data"],
): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/shipping-profiles/${profileId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });
}

export async function deleteShippingProfile(profileId: string): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/shipping-profiles/${profileId}`, {
    method: "DELETE",
  });
}

// --- Expense Rules ---

export async function getExpenseRules(): Promise<ExpenseRule[]> {
  const res = await apiFetch<{ data: ExpenseRule[] }>(
    "/bratrax/cost-settings/expense-rules",
  );
  return res.data;
}

export async function createExpenseRule(
  rule: ExpenseRule["data"],
): Promise<string> {
  const res = await apiFetch<{ rule_id: string }>(
    "/bratrax/cost-settings/expense-rules",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    },
  );
  return res.rule_id;
}

export async function updateExpenseRule(
  ruleId: string,
  rule: ExpenseRule["data"],
): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/expense-rules/${ruleId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function deleteExpenseRule(ruleId: string): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/expense-rules/${ruleId}`, {
    method: "DELETE",
  });
}

// --- Media Spend Scope Rules ---

export async function getMediaSpendScopeRules(): Promise<{
  rules: MediaSpendScopeRule[];
  guidance: MediaSpendScopeGuidance | null;
  availableAccounts: MediaSpendScopeAccountOption[];
}> {
  const res = await apiFetch<{
    data: MediaSpendScopeRule[];
    meta?: {
      ui_guidance?: MediaSpendScopeGuidance;
      available_accounts?: MediaSpendScopeAccountOption[];
    };
  }>("/bratrax/cost-settings/media-spend-scope-rules");
  return {
    rules: res.data,
    guidance: res.meta?.ui_guidance ?? null,
    availableAccounts: res.meta?.available_accounts ?? [],
  };
}

export async function createMediaSpendScopeRule(
  rule: MediaSpendScopeRuleData,
): Promise<string> {
  const res = await apiFetch<{ rule_id: string }>(
    "/bratrax/cost-settings/media-spend-scope-rules",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    },
  );
  return res.rule_id;
}

export async function updateMediaSpendScopeRule(
  ruleId: string,
  rule: MediaSpendScopeRuleData,
): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/media-spend-scope-rules/${ruleId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
}

export async function deleteMediaSpendScopeRule(ruleId: string): Promise<void> {
  await apiFetch(`/bratrax/cost-settings/media-spend-scope-rules/${ruleId}`, {
    method: "DELETE",
  });
}

// --- One-time Expense ---

export async function createOneTimeExpense(expense: {
  date: string;
  amount: number;
  category: string;
  description: string;
  is_ad_spend: boolean;
}): Promise<string> {
  const res = await apiFetch<{ spend_id: string }>(
    "/bratrax/cost-settings/expenses/one-time",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(expense),
    },
  );
  return res.spend_id;
}
