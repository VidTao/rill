import { apiFetch } from "./api";

export type ChecklistStatus = "done" | "in_progress" | "todo" | "skipped" | "error";

export interface ChecklistItem {
  key: string;
  label: string;
  description: string | null;
  status: ChecklistStatus;
  action_url: string | null;
  action_label: string | null;
  label_url?: string | null;
  skippable_prominence: "prominent" | "quiet";
  user_override: "done" | "skipped" | null;
  required_role: string;
}

export interface ChecklistSection {
  key: string;
  title: string;
  items: ChecklistItem[];
}

export interface ChecklistContactAdmin {
  name: string;
  email: string;
}

export interface ChecklistResult {
  role: "admin" | "viewer" | "super_admin";
  workspace_name: string;
  contact_admin: ChecklistContactAdmin | null;
  workspace_step: string | null;
  checklist_dismissed: boolean;
  welcome_card_dismissed: boolean;
  sections: ChecklistSection[];
}

/** Fetch the full checklist state for the current user. */
export function getChecklist(): Promise<ChecklistResult> {
  return apiFetch<ChecklistResult>("/bratrax/onboard/checklist");
}

function postJson(path: string, body: Record<string, unknown>): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

/**
 * Record a Section-4 first-visit / first-use event. Idempotent server-side
 * (ON CONFLICT DO NOTHING) and best-effort — callers should swallow errors.
 */
export function autoMark(itemKey: string): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/checklist/auto-mark", { item_key: itemKey });
}

/**
 * Fire an auto-mark at most once per browser per item (localStorage guard),
 * best-effort. The server is also idempotent; this just avoids redundant
 * requests on every dashboard load / chat message.
 */
export function autoMarkOnce(itemKey: string): void {
  const guard = `bratrax_automark_${itemKey}`;
  try {
    if (localStorage.getItem(guard)) return;
  } catch {
    // localStorage unavailable — fall through and just fire the request.
  }
  void autoMark(itemKey)
    .then(() => {
      try {
        localStorage.setItem(guard, "1");
      } catch {
        // ignore
      }
    })
    .catch(() => {
      // Best-effort: a failed mark simply means we'll try again next time.
    });
}

/** Manually mark an item done (override, survives auto-status changes). */
export function markItem(itemKey: string): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/checklist/override", {
    item_key: itemKey,
    status: "done",
  });
}

/** Skip an item (override). */
export function skipItem(itemKey: string): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/checklist/override", {
    item_key: itemKey,
    status: "skipped",
  });
}

/** Clear an override, reverting to the auto-derived status. */
export function unskipItem(itemKey: string): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/checklist/override", {
    item_key: itemKey,
    status: null,
  });
}

/** Stop the checklist auto-loading on login (still reachable from the menu). */
export function dismissChecklist(): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/checklist/dismiss", {});
}

/** Viewer-only: stop the welcome card showing on login. */
export function dismissWelcome(): Promise<{ ok: boolean }> {
  return postJson("/bratrax/onboard/welcome/dismiss", {});
}

// ---------------------------------------------------------------------------
// Shared derivations used by the page, banner, and routing guard.
// ---------------------------------------------------------------------------

// Items whose absence means data genuinely isn't flowing yet. EVERYTHING else
// on the checklist (COGS, timezone, Anthropic key, Klaviyo, shipping, expenses,
// tracking templates, the "first look" items, invite teammate) is
// finish-at-leisure and lives on the checklist page — it must NOT nag an
// already-`ready` workspace via the dashboards banner. Without this scoping the
// banner fired for every established client, because optional items are
// perpetually `todo`.
const DATA_BLOCKING_KEYS = ["connect_shopify"];

function itemIsResolved(item: ChecklistItem): boolean {
  // Skipped and done both count as "no longer outstanding".
  return item.status === "done" || item.status === "skipped";
}

function itemsByKey(result: ChecklistResult): Map<string, ChecklistItem> {
  const map = new Map<string, ChecklistItem>();
  for (const section of result.sections) {
    for (const item of section.items) map.set(item.key, item);
  }
  return map;
}

/**
 * True only when a data-blocking item (e.g. Shopify not connected) is
 * outstanding. Gates the dashboards banner. A fully-onboarded (`ready`)
 * workspace has Shopify connected, so this is false for them — the banner
 * never nags about optional polish.
 */
export function coreSetupIncomplete(result: ChecklistResult): boolean {
  const byKey = itemsByKey(result);
  return DATA_BLOCKING_KEYS.some((key) => {
    const item = byKey.get(key);
    return !!item && !itemIsResolved(item);
  });
}

/** Count of resolved vs total items across all sections (progress bar). */
export function progressCounts(result: ChecklistResult): { done: number; total: number } {
  let done = 0;
  let total = 0;
  for (const section of result.sections) {
    for (const item of section.items) {
      total += 1;
      if (itemIsResolved(item)) done += 1;
    }
  }
  return { done, total };
}
