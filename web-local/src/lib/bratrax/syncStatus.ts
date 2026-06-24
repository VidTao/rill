import { get } from "svelte/store";
import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

// Shared sync/backfill status helpers — consumed by the Connectors page
// (pill-as-progress-bar + per-account pop-up + reassurance copy) and the
// onboarding checklist callout.
//
// Backend contract: GET /bratrax/sync-status returns one row per
// (source, account). `source` matches the Connectors page `platform.id`
// (i.e. connected_platforms[].platform). external_pages is excluded
// server-side. `progress_pct` is already 0–100 and rounded — we use it
// directly rather than recomputing from chunk counts.

export interface SyncAccount {
  source: string;
  account_name: string;
  account_id: string;
  done_chunks: number;
  total_chunks: number;
  progress_pct: number;
  last_sync_at: string | null;
  connected_at: string | null;
}

export interface SyncStatusResponse {
  client_id: string;
  accounts: SyncAccount[];
}

export interface SourceSyncStatus {
  /** Worst-case % across the source's accounts (min). */
  progressPct: number;
  /** Oldest last_sync_at across accounts (ISO), or null if none synced yet. */
  lastSyncAt: string | null;
  /** The raw per-account rows, for the View Accounts pop-up. */
  accounts: SyncAccount[];
}

/**
 * Fetch per-source-per-account sync status for the active client. Returns an
 * empty array on any error — this UI is purely informational and must never
 * block the page it sits on. The active client (incl. super-admin switcher)
 * is resolved server-side from the auth/X-Bratrax-Client-Id headers, so no
 * params are passed.
 */
export async function fetchSyncStatus(): Promise<SyncAccount[]> {
  try {
    const res = await fetch(`${get(runtime).host}/bratrax/sync-status`, {
      credentials: "include",
    });
    if (!res.ok) return [];
    const body = (await res.json()) as SyncStatusResponse;
    return Array.isArray(body?.accounts) ? body.accounts : [];
  } catch {
    return [];
  }
}

/**
 * Group accounts by `source`, applying the spec's worst-case aggregation:
 * source-level % = min(progress_pct), source-level last sync = min(last_sync_at,
 * oldest). Surfaces stuck accounts instead of hiding them behind an average.
 */
export function aggregateBySource(
  accounts: SyncAccount[],
): Map<string, SourceSyncStatus> {
  const bySource = new Map<string, SourceSyncStatus>();
  for (const acct of accounts) {
    const existing = bySource.get(acct.source);
    if (!existing) {
      bySource.set(acct.source, {
        progressPct: acct.progress_pct,
        lastSyncAt: acct.last_sync_at,
        accounts: [acct],
      });
      continue;
    }
    existing.progressPct = Math.min(existing.progressPct, acct.progress_pct);
    existing.lastSyncAt = olderTimestamp(
      existing.lastSyncAt,
      acct.last_sync_at,
    );
    existing.accounts.push(acct);
  }
  return bySource;
}

/** True when any source still has a backfill running (< 100%). Drives the
 *  Connectors reassurance copy and the onboarding checklist callout. */
export function anyBackfilling(
  bySource: Map<string, SourceSyncStatus>,
): boolean {
  for (const status of bySource.values()) {
    if (status.progressPct < 100) return true;
  }
  return false;
}

/** Treat ≥ 100 as complete (guards against rounding above 100). */
export function isComplete(progressPct: number | undefined): boolean {
  return progressPct === undefined || progressPct >= 100;
}

/** Pop-up / card label. Percent-only per the design decision — the endpoint
 *  exposes chunks, not days, so no "(X of 365 days)" parenthetical. */
export function backfillLabel(progressPct: number): string {
  if (progressPct >= 100) return "Backfill complete";
  if (progressPct <= 0) return "Backfill starting…";
  return `${progressPct}% backfilled`;
}

/**
 * Relative "last sync" label for the main Connectors view: "just now",
 * "2 min ago", "2 hr ago", "Yesterday", "N days ago". Absolute timestamps
 * (toLocaleString) are used in the pop-up instead.
 */
export function relativeTime(iso: string | null | undefined): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diffMs = Date.now() - then;
  const mins = Math.floor(diffMs / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs} hr ago`;
  const days = Math.floor(hrs / 24);
  if (days === 1) return "Yesterday";
  return `${days} days ago`;
}

/** Return whichever ISO timestamp is older; nulls lose to a real value, and
 *  two nulls stay null (no sync on either side yet). */
function olderTimestamp(a: string | null, b: string | null): string | null {
  if (!a) return b;
  if (!b) return a;
  return new Date(a).getTime() <= new Date(b).getTime() ? a : b;
}
