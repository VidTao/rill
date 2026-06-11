import { writable, get as getStore } from "svelte/store";
import { apiFetch } from "./onboarding/api";

// Phase 3 of the nav restructure (docs/NAV_RESTRUCTURE_HANDOFF.MD):
// Per-user dashboard preferences power the CUSTOMIZE drag-and-drop panel
// AND the Ribbon 2 tab order. Backed by `rill_user_dashboard_prefs` in PG.

// ---------------------------------------------------------------------------
// Rill upstream demo-project filter
// ---------------------------------------------------------------------------
// Rill's Go binary ships bundled example projects under
// runtime/pkg/examples/embed/dist/. Their dashboards leak into the canvas
// list briefly when a freshly-provisioned client's runtime instance is
// warming up (the runtime resolves "default" to a demo instance before
// the auth cookie's mapping fully binds). These names are constants in the
// Rill binary — no real Bratrax workspace would ever name a canvas this.
// Filter them out at every canvas-list read site so the user never sees a
// stale demo tab flash on first login / post-onboarding.
//
// Sources of these names:
//   - rill-cost-monitoring → dashboards/margin_scorecard.yaml
//   - rill-openrtb-prog-ads → dashboards/auction_explore.yaml
//   - rill-github-analytics → dashboards/clickhouse_commits_explore.yaml
export const RILL_DEMO_CANVAS_NAMES: ReadonlySet<string> = new Set([
  "margin_scorecard",
  "auction_explore",
  "clickhouse_commits_explore",
]);

export function isRillDemoCanvas(name: string | undefined | null): boolean {
  return !!name && RILL_DEMO_CANVAS_NAMES.has(name);
}

export interface DashboardPref {
  key: string;
  position: number;
  visible: boolean;
}

export interface DashboardPrefsResult {
  dashboards: DashboardPref[];
}

/** Fetch the caller's saved prefs. Empty list = no overrides yet. */
export function getDashboardPrefs(): Promise<DashboardPrefsResult> {
  return apiFetch<DashboardPrefsResult>("/bratrax/user/dashboard-prefs");
}

/** Full replace of the caller's prefs. */
export function putDashboardPrefs(
  prefs: DashboardPref[],
): Promise<{ ok: boolean; count: number }> {
  return apiFetch("/bratrax/user/dashboard-prefs", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ dashboards: prefs }),
  });
}

// ---------------------------------------------------------------------------
// Live store + debounced auto-save
// ---------------------------------------------------------------------------

// The store mirrors the server's prefs and acts as the single source of
// truth for the layout (Ribbon 2 tab order) and the CUSTOMIZE page. Mutations
// go through `setDashboardPrefs` which updates the store synchronously and
// schedules a debounced PUT so the network roundtrip never lags the UI.

export const dashboardPrefs = writable<DashboardPref[] | null>(null);

// True once the initial GET has completed (either with rows or an empty list).
// Consumers can gate first render on this to avoid an order flicker when the
// strip mounts before prefs land.
export const dashboardPrefsLoaded = writable<boolean>(false);

let saveTimer: ReturnType<typeof setTimeout> | null = null;
const SAVE_DEBOUNCE_MS = 500;

/**
 * Load prefs from the server into the store. Idempotent — subsequent calls
 * after the first successful load are no-ops, so it's safe to call from any
 * component's onMount. Call `resetDashboardPrefs()` first when the auth
 * identity changes so the next load actually refetches.
 */
export async function loadDashboardPrefs(): Promise<void> {
  if (getStore(dashboardPrefsLoaded)) return;
  try {
    const res = await getDashboardPrefs();
    dashboardPrefs.set(res.dashboards ?? []);
  } catch {
    // Fail silently — the caller falls back to the runtime's natural canvas
    // order when the store is null/empty.
    dashboardPrefs.set([]);
  } finally {
    dashboardPrefsLoaded.set(true);
  }
}

/**
 * Drop the cached prefs so the next load refetches. Used when the auth user
 * changes within the same SPA session — module-scoped state otherwise leaks
 * the previous user's prefs into the new session and the strip renders with
 * the wrong order until the next hard refresh.
 */
export function resetDashboardPrefs(): void {
  // Cancel any pending debounced save so it doesn't fire under the new
  // identity with the previous user's data.
  if (saveTimer !== null) {
    clearTimeout(saveTimer);
    saveTimer = null;
  }
  dashboardPrefs.set(null);
  dashboardPrefsLoaded.set(false);
}

/** Update the store and persist after a short debounce window. */
export function setDashboardPrefs(next: DashboardPref[]): void {
  dashboardPrefs.set(next);
  if (saveTimer !== null) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    saveTimer = null;
    const snapshot = getStore(dashboardPrefs) ?? [];
    void putDashboardPrefs(snapshot).catch(() => {
      // best-effort; a failed save will be retried on the next mutation.
    });
  }, SAVE_DEBOUNCE_MS);
}

/**
 * Merge runtime canvas list with stored prefs.
 *
 * - Honors stored visibility + position for canvases that exist in prefs.
 * - Appends canvases not yet in prefs at the end of the order (visible by
 *   default — newly compiled dashboards opt-in, not opt-out).
 * - Drops pref rows whose canvas no longer exists in the runtime (the merge
 *   is one-way; the actual PG cleanup happens on the next user PUT).
 *
 * The return value is a fresh array sorted by effective position. Each entry
 * carries the runtime resource alongside the pref so callers can read the
 * `displayName` directly without a second lookup.
 */
export interface MergedDashboard<T> {
  key: string;
  resource: T;
  position: number;
  visible: boolean;
}

export function mergeDashboardPrefs<T extends { meta?: { name?: { name?: string } | null } | null }>(
  canvases: T[],
  prefs: DashboardPref[] | null,
): MergedDashboard<T>[] {
  const prefByKey = new Map<string, DashboardPref>();
  for (const p of prefs ?? []) prefByKey.set(p.key, p);

  const maxStoredPos = (prefs ?? []).reduce(
    (acc, p) => Math.max(acc, p.position),
    -1,
  );
  let appendPos = maxStoredPos + 1;

  const merged: MergedDashboard<T>[] = [];
  for (const c of canvases) {
    const key = c?.meta?.name?.name ?? "";
    if (!key) continue;
    const pref = prefByKey.get(key);
    if (pref) {
      merged.push({
        key,
        resource: c,
        position: pref.position,
        visible: pref.visible,
      });
    } else {
      merged.push({
        key,
        resource: c,
        position: appendPos++,
        visible: true,
      });
    }
  }
  merged.sort((a, b) => a.position - b.position);
  return merged;
}
