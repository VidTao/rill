# Bratrax onboarding UX improvement plan

## Context

Bratrax is an attribution SaaS for DTC brands, built as a branded fork of Rill (this repo). The Bratrax backend (separate system) handles OAuth into ad/commerce platforms and computes attribution; this Rill frontend renders the resulting dashboards. Two user populations consume it:

- **Brand operators / marketers** (non-admin) — see a clean dashboard view (see screenshot 3). Their problem is metric comprehension and "what do I do next" after the backend hands them off.
- **Super admins** (your team, agency operators) — see the Rill IDE chrome (screenshots 1 + 2). Their problem is information overload, dbt-style file naming, and exposure of files that shouldn't be in the UI tree at all (`.env`, `claude_system_prompt.txt`, `rill.yaml`).

User feedback says the app has a "deep learning curve." Investigation shows three root causes, not one:

1. There is no guided path from "I just signed up" to "I am using the dashboards." Connections finish in the backend and the user is dropped into a dashboard with no orientation.
2. Metrics on the dashboard (MER, NC ROAS, Blended CPA, Contribution Margin, etc.) are unfamiliar to DTC operators and have no in-product definitions.
3. The Bratrax → Rill rebrand is partial. End users still see "Rill", "ASK CLAUDE", `rill.yaml`, and example projects named "OpenRTB Programmatic Ads" / "GitHub Analytics" that mean nothing to a DTC marketer.

Intended outcome: a marketer can land in Bratrax, understand what they're looking at within their first session, and complete a discrete activation funnel that we can measure.

## Recommended approach

Nine workstreams, ordered by leverage. Each can ship independently behind a feature flag.

---

### 1. Bratrax onboarding checklist (highest leverage)

**Shape (confirmed with user):** floating bottom-right widget (Intercom-style), hybrid auto/manual completion, replaces the existing "Your analytics are live!" banner.

**Items (9 + 3 optional, in 3 phases):**

| # | Phase | Item | Completion signal |
|---|---|---|---|
| 1 | Get data flowing | Connect your store | Backend webhook → frontend (manual fallback) |
| 2 | Get data flowing | Connect at least one ad platform | Backend webhook → frontend (manual fallback) |
| 3 | Get data flowing | Enter COGS & shipping in Cost Settings | Frontend — Cost Settings save event |
| 4 | Get data flowing | First data sync complete | Backend webhook → frontend (manual fallback) |
| 5 | Learn dashboards | View Performance Overview | Frontend — dashboard view event |
| 6 | Learn dashboards | Drill into a channel on Campaign Deep Dive | Frontend — row drill event |
| 7 | Learn dashboards | Switch the attribution model once | Frontend — filter-change event on `attribution_model` |
| 8 | Learn dashboards | Ask Bratrax your first question | Frontend — chat-message-sent event |
| 9 | Learn dashboards | Invite a teammate | Frontend — invite event |
| 10 | Optional | Set up an alert | Frontend — alert create event |
| 11 | Optional | Schedule a weekly report | Frontend — report create event |
| 12 | Optional | Connect Klaviyo / Postscript | Backend webhook |

**Critical files:**
- New: `web-local/src/lib/bratrax/onboarding/OnboardingChecklist.svelte` — the floating widget.
- New: `web-local/src/lib/bratrax/onboarding/checklist-store.ts` — Svelte store; per-user state. Server-backed (NOT localStorage — must survive device changes; existing `WelcomeBanner.svelte` uses `localStorage["bratrax_welcome_dismissed"]` but that's not durable enough for activation tracking).
- New: `web-local/src/lib/bratrax/onboarding/checklist-items.ts` — the registry above.
- Delete: `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte` (replaced by checklist; banner content folded into checklist's empty/initial state).
- Modify: wherever `WelcomeBanner` is rendered (search `import WelcomeBanner`) — replace with `OnboardingChecklist`.

**Cross-system dependency (call out as blocker for items 1, 2, 4, 12):** Bratrax backend must expose an endpoint (e.g., `GET /v1/onboarding/status`) returning `{ store_connected, ad_platforms: [...], first_sync_complete, klaviyo_connected }`. The frontend polls every 30s while the checklist is open and on first mount. Without this endpoint the checklist still ships but those items are manual-only.

**Telemetry:** new `BehaviourEventAction` values: `onboarding_checklist_shown`, `onboarding_item_completed` (with `item_id` payload), `onboarding_checklist_dismissed`, `onboarding_phase_completed`. Add to `web-common/src/metrics/service/BehaviourEventTypes.ts`.

---

### 2. Metric literacy layer

**Problem:** KPI tiles (MER, NC ROAS, Blended CPA, Contribution Margin, Attributed Sales, etc.) have no plain-English definition. Some charts already have helpful subtitles (NC ROAS, NC CPA, CPM, CTR) but the pattern is inconsistent.

**Plan:**
- The infrastructure already exists. `web-common/src/features/canvas/ComponentHeader.svelte:60-83` supports both subtitle and tooltip-icon variants of a `description` field. `web-common/src/features/canvas/components/kpi/KPI.svelte:170-308` wraps tiles in a `Tooltip`.
- Measure definitions in metrics-view YAML already have a `description` field (confirmed in `runtime/parser/parse_metrics_view.go:37-73` and `proto/rill/runtime/v1/resources.proto` — `Dimension.description`, `Measure.description`).
- **Action 1:** add a small info `(i)` icon to the KPI tile header that opens a tooltip showing the measure's `description`. Modify `KPI.svelte` around lines 189-196 (title block) to render the icon when `measure.description` is set. Reuse `BigNumberTooltipContent.svelte` pattern.
- **Action 2:** audit every metrics-view YAML under the project directory (8 metrics views per screenshot 2) and ensure every measure has a Bratrax-quality `description`. Format: *"<plain-English definition>. Formula: <formula>. <When to use it>."*
- **Action 3:** for canvas charts, ensure every chart's `description` field is populated. The 5 charts that already have subtitles (ROAS by Channel implicitly via legend, NC ROAS, NC CPA, CPM, CTR) prove this works — extend to the other 4 charts (Spend by Channel, Attributed Sales by Channel, and the table).

**Metrics that need v1 descriptions:** MER, ROAS, NC ROAS, CPA, NC CPA, Blended Spend, Blended CPA, Contribution Margin, Attributed Sales, CPM, CTR, AOV, New Purchases, NC Sales, Repeat Orders (the truncated "Rep…" column).

**No new files needed.** This is pure content + a small edit to `KPI.svelte`.

---

### 3. Promote the attribution-model selector

**Problem:** "Attribution Model: last_touch" appears as a generic dimension-filter chip (`web-common/src/features/dashboards/filters/dimension-filters/DimensionFilterReadOnlyChip.svelte`). It's the single most consequential control on the dashboard and looks identical to a date filter.

**Plan:**
- Add a dedicated `AttributionModelSelector.svelte` component in `web-local/src/lib/bratrax/` that renders as a labeled dropdown above the KPI grid, e.g. `Attribution model: ▾ Last touch`.
- Each option in the dropdown carries a one-line "when to use" description.
- Component is dashboard-aware: it reads/writes the `attribution_model` dimension filter via the existing dashboard state store, so behavior is unchanged.
- Hide the filter chip for `attribution_model` specifically (filter chip code already filters by dimension name).

**Critical files:**
- New: `web-local/src/lib/bratrax/AttributionModelSelector.svelte`.
- Modify: dashboard layout component (the parent of the filter bar; identify exact file when implementing — likely `web-common/src/features/canvas/CanvasDashboard.svelte` or similar).

---

### 4. First-dashboard coachmark tour

**Problem:** No infrastructure exists (`tour`, `coachmark`, `intro`, `shepherd`, `driver.js` all return 0 results in web-admin). Marketers get no orientation on their first dashboard.

**Plan:**
- Lightweight in-house coachmark. No external dependency. Reuse the existing tooltip primitives in `web-common/src/components/tooltip/`.
- 3 steps the first time a user opens any dashboard:
  1. KPI grid: *"These are your topline KPIs. Each tile compares the selected period to the previous one."*
  2. Attribution model selector (from workstream 3): *"Change the attribution model here to see the same data attributed differently."*
  3. Channel table: *"Click any channel row to drill into campaign → ad set → ad."*
- Dismissal: persist server-side via same per-user preferences store as the checklist (workstream 1). Do NOT use localStorage.
- Suppress the tour if the checklist is also visible — show one or the other, not both at once. The tour is for first-visit orientation; the checklist is for first-week activation.

**Critical files:**
- New: `web-local/src/lib/bratrax/onboarding/DashboardTour.svelte`.
- New: shared util `web-local/src/lib/bratrax/onboarding/user-preferences.ts` — wraps the server-side preferences API (also used by the checklist).

---

### 5. Date range UX cleanup

**Problem:** The date control has two stacked rows ("Last 7 Days May 12-18, 2026 UTC" and "as of latest day end"). "As of latest day end" is opaque to a marketer.

**Plan:**
- Collapse into one control with sensible defaults: Last 7 / 28 / 90 days, MTD, QTD, YTD, Custom.
- Replace "as of latest day end" with a small data-freshness stamp: `Data updated 12 min ago`.
- Move the "Comparing: Previous period" toggle next to the date control, not below.

**Critical files:**
- Modify: `web-common/src/features/dashboards/time-controls/super-pill/SuperPill.svelte` (main time control assembly).
- Modify: `web-common/src/features/dashboards/time-controls/super-pill/components/RangeDisplay.svelte` — replace "as of latest day end" copy with the freshness stamp, sourcing the timestamp from the runtime's last-refresh metadata.

---

### 6. Admin sidebar overhaul (super-admin view)

**Problem:** From screenshot 2 — the admin file tree exposes `.env`, `claude_system_prompt.txt`, and `rill.yaml`; uses dbt-style names (`dim_*`, `fct_*`); truncates everything with `...`; and has no search.

**Plan:**

**6a. Hide sensitive / internal files from the file tree (priority — has a security dimension).**
- Add an explicit denylist: `.env`, `claude_system_prompt.txt`, `.gitignore`, anything under `.bratrax/`, `rill.yaml` (rename or hide; see 6b).
- Modify the file-tree component (the left sidebar in `web-local/`; identify in `web-local/src/routes/` or `web-local/src/lib/`).
- For files that admins legitimately need to edit (project config), surface them through a dedicated **Project settings** panel, not as raw files.

**6b. Complete the `rill.yaml` → `bratrax.yaml` rename.** Currently exposes the upstream identity. Option A: rename the file (requires runtime/parser support for `bratrax.yaml` as an alias of `rill.yaml`). Option B: render it in the UI as `bratrax.yaml` but keep the file name (minimal). Recommend Option A long-term, ship Option B first.

**6c. Display names instead of file names in the sidebar.** Read the `Display name:` field from each YAML (already supported per `parse_metrics_view.go`) and render that. Keep file names in URLs for stability. Add a hover tooltip showing the file name for power users.

**6d. Sidebar search + type filter.** Add a search input at the top of the tree and quick filters: Dashboards / Metrics views / Models / Sources / Connectors.

**6e. View-as toggle for super admins.** Persistent toggle at the top of the chrome: `View as: Admin ▾` with options `Admin`, `Brand operator`, `Viewer`. Non-admin modes hide the left tree, the right Canvas configurations panel, the `</>` icon, the `+` add buttons, and the "Viewing default state / Preview" controls.

**Critical files:** identify when implementing — the file tree and right-panel components are in `web-local/`; the right Canvas configurations panel is rendered when a canvas dashboard is open in editor mode.

---

### 7. Rebrand sweep (Rill → Bratrax)

**Problem:** ~32 files in `web-admin/src/` still reference "Rill" in user-facing strings; documentation links hardcode `docs.rilldata.com`; the welcome page references example projects irrelevant to DTC ("OpenRTB Programmatic Ads", "GitHub Analytics", "Cost Monitoring"); the chat button says "ASK CLAUDE" (an implementation detail).

**Plan:**

**7a. Centralize brand strings.** Create `web-common/src/lib/branding.ts` exporting `PRODUCT_NAME`, `PRODUCT_DOMAIN`, `DOCS_URL`, `AI_ASSISTANT_NAME`. All user-facing references read from there. This makes future rebranding trivial.

**7b. Specific replacements:**
- `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte:46-51` — "ASK CLAUDE" → "Ask Bratrax" (chat is agent-agnostic per `web-common/src/features/chat/core/conversation.ts`; no backend change needed). N.B. this file is being removed in workstream 1, but the same label appears elsewhere if the chat entry point is reused — audit before deleting.
- `web-admin/src/features/billing/plans/WelcomeToRillCloudDialog.svelte` — title + filename rename to `WelcomeToBratraxDialog.svelte`.
- `web-admin/src/routes/+layout.svelte:132` — `<meta name="description" content="Rill Cloud">` → `"Bratrax"`.
- `web-admin/src/features/projects/status/overview/DeploymentSection.svelte` — "Rill-managed" infrastructure label.
- All hardcoded `docs.rilldata.com`, `www.rilldata.com`, `ui.rilldata.com` links — route through `DOCS_URL` constant. Some can stay (if Bratrax doesn't have its own equivalent docs) but should be labeled differently in the UI.

**7c. Replace the welcome page example projects.** `web-common/src/features/welcome/constants.ts` currently lists `rill-cost-monitoring`, `rill-openrtb-prog-ads`, `rill-github-analytics`. Replace with Bratrax-relevant examples (e.g., "Sample DTC store: 60-day demo data", "Subscription product: cohort + LTV") — or remove the example-projects section entirely from the marketer-facing path since brand users should never see it.

**7d. Fix `EMPTY_PROJECT_TITLE`.** Currently `"Untitled Bratrax Project"` — keep as-is (it's correct), but verify it surfaces correctly.

---

### 8. Role-aware top navigation

**Problem:** Screenshot 3 shows a non-admin still sees `SUPERADMINS` tab in the top nav. Top nav mixes consumer and admin destinations with no role gating.

**Plan:**
- Brand operators see: Dashboards, Connectors (read-only), Help, Ask Bratrax.
- Brand admins additionally see: Cost Settings, Settings.
- Super admins additionally see: Superadmins.
- Use the existing `organizationPermissions` props already plumbed through `web-admin/src/routes/+layout.svelte:37-42` (pattern confirmed by Explore agent; same gating exists in `web-admin/src/features/organizations/OrganizationTabs.svelte:13-28`).

**Critical files:**
- Identify the top-nav component (likely `web-admin/src/features/navigation/` or similar) and add role gating per the pattern in `OrganizationTabs.svelte`.

---

### 9. Telemetry funnel

**Problem:** No journey-level events exist. Cannot measure whether onboarding changes move retention or activation.

**Plan:**
- Extend `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` with new `BehaviourEventAction` values:
  - `onboarding_checklist_shown`
  - `onboarding_item_completed` (payload: `item_id`, `phase`)
  - `onboarding_phase_completed` (payload: `phase`)
  - `onboarding_checklist_dismissed`
  - `onboarding_tour_started`
  - `onboarding_tour_step_completed` (payload: `step`)
  - `onboarding_tour_dismissed`
  - `metric_tooltip_opened` (payload: `metric_name`)
  - `attribution_model_changed` (payload: `from`, `to`)
  - `data_freshness_clicked`
  - `first_dashboard_view` (payload: `dashboard_name`)
- Reuse the existing `BehaviourEventHandler` pattern (`web-common/src/metrics/BehaviourEventHandler.ts:27-99`). Add a `fireOnboardingEvent(action, payload)` method.

---

## Suggested rollout sequence

1. **Week 1 — Security & branding sweep:** workstreams 6a (hide `.env`, `claude_system_prompt.txt`), 7b (string replacements), 7d. Pure removal/rename; very low risk, ships immediately, fixes the upstream-brand leak.
2. **Week 1-2 — Metric literacy (#2):** content-heavy, low code risk. Highest marketer-impact change after the security sweep.
3. **Week 2-3 — Checklist (#1) + telemetry (#9):** the activation surface and the instrumentation that measures it. Blocked on backend endpoint for auto-completion of phase-1 items; ship hybrid (manual fallback) in week 2 and harden auto-detection in week 3.
4. **Week 3 — Attribution selector (#3) + date cleanup (#5):** polish of the dashboard chrome.
5. **Week 4 — Dashboard tour (#4) + role-aware nav (#8):** structural changes that build on the workstreams above.
6. **Week 4-5 — Admin sidebar (#6b-#6e):** larger overhaul; ship behind a feature flag.
7. **Week 5+ — Rebrand cleanup (#7a, #7c):** the central branding module and example-project replacement once everything else is stable.

## Critical files / existing utilities to reuse

| Purpose | Path | Notes |
|---|---|---|
| Existing Bratrax onboarding home | `web-local/src/lib/bratrax/onboarding/` | New checklist lives here; `WelcomeBanner.svelte` is removed |
| Chat entry point (rename ASK CLAUDE) | `web-common/src/features/chat/DashboardChat.svelte` | Agent-agnostic; just a label |
| Chat backend reference | `web-common/src/features/chat/core/conversation.ts` | Confirms no backend change needed |
| KPI tile (add info icon) | `web-common/src/features/canvas/components/kpi/KPI.svelte:189-196` | Already wraps in Tooltip |
| Chart description rendering | `web-common/src/features/canvas/ComponentHeader.svelte:60-83` | Supports subtitle + tooltip variants |
| Measure/dimension `description` field | `runtime/parser/parse_metrics_view.go:37-73`, `proto/rill/runtime/v1/resources.proto` | Already exists, underused |
| Time-range control | `web-common/src/features/dashboards/time-controls/super-pill/SuperPill.svelte` | Modify for #5 |
| Dimension filter chip | `web-common/src/features/dashboards/filters/dimension-filters/DimensionFilterReadOnlyChip.svelte` | Hide for `attribution_model` per #3 |
| Header breadcrumb + dashboard switcher | `web-common/src/components/navigation/breadcrumbs/BreadcrumbItem.svelte:62-142`, `web-common/src/layout/ApplicationHeader.svelte:42-80` | Already uses display names; reference for #6c |
| Role gating pattern | `web-admin/src/routes/[organization]/+layout.svelte:8`, `web-admin/src/features/organizations/OrganizationTabs.svelte:13-28` | Reuse for #8 |
| Telemetry registry | `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` | Extend for #9 |
| Telemetry handler pattern | `web-common/src/metrics/BehaviourEventHandler.ts:27-99` | Reuse for #9 |
| Existing welcome dialog (for pattern + rename) | `web-admin/src/features/billing/plans/WelcomeToRillCloudDialog.svelte` | Rename per #7b |
| Welcome-page example projects | `web-common/src/features/welcome/constants.ts` | Replace per #7c |
| Rebrand-completeness reference | `runtime/ai/instructions/data/development.md` | Internal rebrand is already complete here — use as canonical reference |

## Open items / cross-team dependencies

1. **Backend endpoint for checklist auto-completion (workstream 1).** Needs Bratrax-backend buy-in. Spec: `GET /v1/onboarding/status` returning `{ store_connected: bool, ad_platforms: string[], first_sync_complete: bool, klaviyo_connected: bool, sms_connected: bool }`. Without this, items 1, 2, 4, 12 are manual-only in v1 (still useful but lower-quality).
2. **Server-side user preferences store.** Both the checklist (workstream 1) and the dashboard tour (workstream 4) need durable per-user dismissal state. The existing `WelcomeBanner.svelte` uses localStorage which is insufficient. Confirm whether an existing user-preferences table exists in `admin/` or if a new one needs to be added.
3. **Decision: rename `rill.yaml` → `bratrax.yaml` (6b option A) or just relabel in UI (option B)?** Affects runtime parser. Recommend Option A long-term, Option B for v1.
4. **Decision: keep or remove the welcome-page example projects entirely (7c)?** Brand operators should never see them; super admins might value them for testing. Recommend removing from the brand-operator path; gating on super-admin role.

## Verification

End-to-end test plan after implementation:

**Checklist (#1):**
- Sign up a new test brand. Verify the checklist widget appears in bottom-right.
- Without connecting anything, verify items 1-4 show as incomplete and items 5-9 are locked or shown as "Phase 2 — complete data setup first".
- Connect a Shopify test store in the Bratrax backend. Within 30s, verify item 1 auto-flips to complete on the frontend.
- Click "Mark complete" on item 3 (Cost Settings). Verify it persists across page reload and across devices (sign in from a second browser).
- Complete all phase-1 items. Verify phase-2 items unlock and the widget shows phase progress.
- Open a dashboard, switch attribution model. Verify item 7 auto-completes.
- Dismiss the checklist. Sign out and back in. Verify it remains dismissed.

**Metric literacy (#2):**
- Hover the info icon on each KPI tile. Verify the tooltip shows the measure's `description`.
- Verify every chart on the default dashboards has either a subtitle or a tooltip icon with the description.

**Attribution selector (#3):**
- Verify the labeled dropdown appears above the KPI grid; the filter chip for `attribution_model` no longer appears in the chip row.
- Switch the model; verify all KPIs and charts re-render with the new attribution.

**Tour (#4):**
- First-ever view of a dashboard for a new user: tour starts. Step through all 3 steps. Verify dismissal persists per workstream-1 testing.

**Date range (#5):**
- Verify the single date control replaces the two-row layout. Verify the freshness stamp ("Data updated X min ago") sources from runtime metadata.

**Admin sidebar (#6):**
- Sign in as a super admin. Verify `.env`, `claude_system_prompt.txt` no longer appear in the tree. Verify `rill.yaml` is hidden or relabeled.
- Verify sidebar shows display names; hover shows file names.
- Search "campaign" in the sidebar; verify filtering works across all resource types.
- Toggle "View as: Brand operator". Verify the file tree, right Canvas configurations panel, `</>` icon, `+` buttons, and editor controls all disappear; chrome matches screenshot 3.

**Rebrand (#7):**
- Grep for "Rill" in user-facing strings: should be 0 hits after sweep (excluding internal documentation and code identifiers).
- Verify "ASK CLAUDE" no longer appears anywhere; replaced by "Ask Bratrax".
- Verify the welcome-page example projects either are Bratrax-themed or are hidden for non-super-admin roles.

**Top nav (#8):**
- Sign in as brand operator: verify SUPERADMINS tab is hidden.
- Sign in as brand admin: verify SUPERADMINS tab is hidden, Cost Settings and Settings are visible.
- Sign in as super admin: verify all tabs visible.

**Telemetry (#9):**
- After all workstreams ship, validate in the analytics warehouse that the new events fire end-to-end. Build a dashboard of the activation funnel: `bratrax_first_login` → `onboarding_phase_completed[phase=data]` → `onboarding_phase_completed[phase=learn]`. This is the primary metric for whether this entire effort worked.

**Tests:**
- `npm run test -w web-common` — covers any new util in `web-common` (e.g., metric registry, branding module).
- `npm run test -w web-local` — Playwright e2e for the checklist appearance, dismissal, and a Phase-1-auto-complete flow (mock the backend endpoint).
- `go test ./...` after any change in `runtime/parser/` (only if `rill.yaml` → `bratrax.yaml` is taken).
- `npm run quality` for lint/format before each PR.
