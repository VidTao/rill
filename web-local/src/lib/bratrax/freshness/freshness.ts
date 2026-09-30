import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
import { relativeTime } from "$lib/bratrax/syncStatus";

// Dashboard data-freshness badge: "Updated 23 min ago · Next update in 17 min".
//
// Backend contract: GET /bratrax/data-freshness (bratrax server/routes/
// data_freshness.py). `data_as_of` is DATA AGE, not processing time — the
// server takes min(last extract, last processing) precisely so a client whose
// extracts stopped reads "Updated 17 days ago" even though ClickHouse keeps
// re-running the cascade over the old rows. Don't swap in the Rill model's
// `refreshedOn` here: for external models that is just the hourly re-read
// tick, and it would report "< 1 hr ago" for a client whose data is weeks old.
//
// `next_update_at` is the client's next Rill tick, and the server only sends it
// while status is "current" — a stalled client has no update coming.

export type FreshnessStatus = "current" | "delayed" | "unknown";

export interface DataFreshness {
  status: FreshnessStatus;
  data_as_of: string | null;
  last_extract_at: string | null;
  last_processed_at: string | null;
  next_update_at: string | null;
}

export interface FreshnessView {
  tone: "current" | "delayed";
  /**
   * The one line the badge shows. Current: "Next update in 17 min" — the data
   * age lives in the tooltip, to keep the badge short enough not to collide
   * with the centred filter bar. Delayed: "Updated 17 days ago", since there
   * is no next update to promise and the age is the warning.
   */
  text: string;
  /** Absolute-time detail for the tooltip, one line per entry. */
  tooltip: string[];
}

/**
 * Fetch freshness for the active client. Returns null on any error: the badge
 * is informational and must never block or break the dashboard it sits on.
 * The client (incl. the super-admin switcher) is resolved server-side from the
 * proxy's identity headers, so no params are passed. Inside the Shopify iframe,
 * installEmbeddedAuthFetch() adds the session token to /bratrax/* fetches.
 */
export async function fetchDataFreshness(): Promise<DataFreshness | null> {
  try {
    const res = await fetch(`${get(runtime).host}/bratrax/data-freshness`, {
      credentials: "include",
    });
    if (!res.ok) return null;
    return (await res.json()) as DataFreshness;
  } catch {
    return null;
  }
}

/** "in 17 min", "in 1 hr", or "any minute" once the time has passed. */
export function formatEta(iso: string, now: number): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const mins = Math.ceil((then - now) / 60_000);
  if (mins <= 0) return "any minute";
  if (mins < 60) return `in ${mins} min`;
  return `in ${Math.round(mins / 60)} hr`;
}

function absolute(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** Pure view model for the badge; null means render nothing. */
export function freshnessView(
  data: DataFreshness | null,
  now: number,
): FreshnessView | null {
  if (!data || data.status === "unknown" || !data.data_as_of) return null;

  const ago = relativeTime(data.data_as_of, now);
  if (!ago) return null;
  // relativeTime capitalises "Yesterday" for its standalone use on the
  // Connectors page; mid-sentence it should read "Updated yesterday".
  const updated = `Updated ${ago.charAt(0).toLowerCase()}${ago.slice(1)}`;

  if (data.status === "delayed") {
    return {
      tone: "delayed",
      text: updated,
      tooltip: [
        `No new data since ${absolute(data.data_as_of)}.`,
        "Numbers may be out of date — check that your platforms are still connected.",
      ],
    };
  }

  // Current but no ETA (the client's config.yaml has no mv_refresh offset):
  // with the age moved to the tooltip there is nothing left to show.
  const eta = data.next_update_at ? formatEta(data.next_update_at, now) : "";
  if (!data.next_update_at || !eta) return null;
  return {
    tone: "current",
    text: `Next update ${eta}`,
    tooltip: [
      `${updated} — includes data received up to ${absolute(data.data_as_of)}.`,
      `Dashboards update hourly — next update around ${absolute(data.next_update_at)}.`,
    ],
  };
}
