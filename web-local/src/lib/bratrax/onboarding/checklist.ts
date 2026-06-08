import { apiFetch } from "./api";

export type ChecklistStatus = "done" | "in_progress" | "todo" | "skipped" | "error";

export interface ChecklistItem {
  key: string;
  label: string;
  description: string | null;
  status: ChecklistStatus;
  action_url: string | null;
  action_label: string | null;
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

/** Section keys that make up "Sections 1+2" (gate the dashboards banner). */
const CORE_SECTION_KEYS = ["connect_data", "configure_business"];

/** Section keys 1–3 (gate the auto-redirect to /onboarding for admins). */
const SETUP_SECTION_KEYS = ["connect_data", "configure_business", "paid_media_tracking"];

function itemIsResolved(item: ChecklistItem): boolean {
  // Skipped and done both count as "no longer outstanding".
  return item.status === "done" || item.status === "skipped";
}

function sectionsIncomplete(result: ChecklistResult, sectionKeys: string[]): boolean {
  return result.sections
    .filter((s) => sectionKeys.includes(s.key))
    .some((s) => s.items.some((i) => !itemIsResolved(i)));
}

/** True when any Section 1–3 item is still outstanding (admin redirect rule). */
export function hasIncompleteSetup(result: ChecklistResult): boolean {
  return sectionsIncomplete(result, SETUP_SECTION_KEYS);
}

/** True when Sections 1+2 aren't both fully resolved (dashboards banner rule). */
export function coreSetupIncomplete(result: ChecklistResult): boolean {
  return sectionsIncomplete(result, CORE_SECTION_KEYS);
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
