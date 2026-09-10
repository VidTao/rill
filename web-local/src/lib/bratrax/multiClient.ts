import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

// Backs the "Add store" button in the application header. add-store itself
// requires rill_users.multi_client_id != NULL, so a client adding their SECOND
// store has to be promoted first — hence bratraxPromoteSelf below, which the
// button always calls ahead of add-store.

function host(): string {
  return get(runtime).host;
}

export interface PromoteResult {
  multi_client_id: string;
  promoted: boolean;
}

// Turns the caller's single-store account into a multi-store parent.
// Idempotent: an account that already has a parent gets its existing id back
// with promoted:false, so calling this unconditionally is safe and keeps the
// add-store chain retryable after a partial failure. Refused with 403 when
// ALLOW_SHOPIFY_MULTI_STORE is off on the Flask side.
export async function bratraxPromoteSelf(): Promise<PromoteResult> {
  const res = await fetch(`${host()}/bratrax/multi-client/promote`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Enabling multi-store failed (${res.status})`);
  }
  return res.json();
}

export interface AddStoreResult {
  client_id: string;
  client_name: string;
  clickhouse_db: string;
  multi_client_id: string;
}

export async function bratraxAddStore(
  companyName: string,
): Promise<AddStoreResult> {
  const res = await fetch(`${host()}/bratrax/multi-client/add-store`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ company_name: companyName }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Add store failed (${res.status})`);
  }
  return res.json();
}
