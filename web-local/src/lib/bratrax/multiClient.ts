import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

// Backs the "Add store" button in the application header. Only meaningful
// for users with rill_users.multi_client_id != NULL — the backend rejects
// callers without a parent multi-client.

function host(): string {
  return get(runtime).host;
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
