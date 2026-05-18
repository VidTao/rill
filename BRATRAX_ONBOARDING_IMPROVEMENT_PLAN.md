# Bratrax onboarding UX improvement plan

## Context

Bratrax Lite is an analytics/attribution product for Shopify DTC brands, launched May 12, 2026. Founding 100 cohort is filling (~8 of 33 onboarded mid-May), pricing locked at ~$79/month for life. The product is a Bratrax-branded fork of Rill (this repo). The Bratrax backend (separate Go service) handles OAuth into ad/commerce platforms and computes attribution; the Rill frontend renders the resulting dashboards.

The recurring theme in user feedback is "steep learning curve." This document treats that complaint as five distinct problems with five distinct fixes, not one diffuse onboarding gap.

### What users actually said

**Alessio (cancelled, then re-engaged after a humble support reply):** "We're a two-person team, neither of us are developers, and the time investment to properly understand and use the tool the way it's meant to be used is just too high at the moment… we need something we can get running without a steep learning curve." On returning, his direct ask: *"Do you have any onboarding material? A written guide, a walkthrough, anything that explains the basics step by step?"*

**Martin (active user, six specific issues):**
1. UI flash/disappear bug on Campaign Deep Dive for first-day-of-campaign data.
2. Timezone set to EDT but data displayed in UTC. "Yesterday" not in quick-filter dropdown.
3. UTM migration from Triple Whale / Northbeam / Redtrack is a brick wall — rewriting hundreds of ad UTMs would reset Meta learning mode and tank performance.
4. Cost of Goods doesn't support per-variant pricing.
5. No customer-journey-per-order view ("here's what it looks like in TW").
6. Cross-panel reconciliation: top KPI tile says Blended Spend $1,517 (labeled "Correct"); bottom Channel Attribution table shows a different number ("Wrong"). Different attribution models, no in-product cue explaining the difference.

Closing line: *"The learning curve of understanding how to use it is quite steep. You have a lot of reports and queries and things that just don't make sense to a regular user."*

**Justin (active user, discoverability gap):** Found Campaign Deep Dive, got real value ("I've already made adjustments in my campaigns based on the data… those adjustments look like they are helping"). But: *"Beyond the campaign deep dive page, I'm not sure where else to find usable insights."* He's using a small fraction of the product and doesn't know what he's missing.

**In-app screenshots:**
- Campaign Deep Dive with "Today" selected: all KPIs `null`, Channel Attribution shows "No data to show for the selected filters." No explanation that "Today" data hasn't synced yet.
- User clicked an order row expecting a customer journey; got the audit log instead.
- Campaign Deep Dive with hand-labeled "Correct" / "Wrong" annotations on two panels showing inconsistent Blended Spend.

### The five-tier classification

Each tier is a different problem that wants a different fix.

| Tier | Problem | Example | Fix type |
|---|---|---|---|
| **1** | UX defects masquerading as learning curve | "Today" default returns nulls; timezone doesn't flow through; click order → audit log | Bug/copy fixes — no teaching required |
| **2** | Discoverability gaps | Justin doesn't know other dashboards exist | In-app nav cues, first-visit tooltips, checklist |
| **3** | Structural switching costs (not onboarding) | UTM migration from competitors | Product work — flag for backend roadmap |
| **4** | Genuine onboarding gap | No written getting-started guide | Write the guide |
| **5** | Feature gaps (not onboarding) | Per-variant COGS, Microsoft Ads, PMAX views | Roadmap items — flag separately |

Fixing Tier 1 alone reduces the perceived learning curve significantly. Tiers 3 and 5 are scoped out of this plan and flagged to the backend / product roadmap.

### Intended outcome

A non-technical Shopify operator can sign up, complete a guided setup in under 20 minutes, land on a dashboard whose defaults make sense, understand what they're looking at, and discover every other dashboard within their first session — all while feeling that the product respects their time.

---

## What already exists (do not rebuild)

The investigation found significant existing infrastructure. Any plan must extend rather than duplicate it.

| Existing surface | Path | Notes |
|---|---|---|
| Onboarding wizard with resume logic | `web-local/src/routes/onboard/` — subroutes `shopify`, `business`, `payment`, `stack`, `embed`, `loading` | The team already built a multi-step wizard. The new checklist must integrate with this, not replace it. Resume logic at `web-local/src/routes/+layout.ts:68-120`. |
| Bratrax welcome banner | `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte` | "Your analytics are live!" banner with "ASK CLAUDE →" CTA. localStorage-backed dismissal. To be replaced by the checklist. |
| Bratrax onboarding API client | `web-local/src/lib/bratrax/onboarding/api.ts` (11 KB) | Existing backend integration. Extend, don't reimplement. |
| Tracking template guide | `web-local/src/lib/bratrax/TrackingTemplateGuide.svelte` | Renders the Google/Meta URL Templates UI. Used in connectors flow. |
| Tracking verification checklist | `web-local/src/lib/bratrax/tracking-templates.ts:11-16` | Parameterized 4-step "Verify after setup" list. Reuse in checklist Section 3. |
| Settings — AI tab (Anthropic key BYOK) | `web-local/src/routes/settings/+page.svelte:30-36` (tab `ai`) | Key stored server-side via `getAISettings`/`updateAISettings` at lines 12-14, 173-216. Confirmed BYOK. |
| Settings — Account tab (timezone capture) | `web-local/src/routes/settings/+page.svelte:45-76` | User can pick from ~30 zones; persisted via `updateAccount()`. **Bug: not applied to dashboard queries — see P0.3.** |
| Connectors page | `web-local/src/routes/connectors/+page.svelte` | Existing hub for Shopify, ad platforms, URL templates. |
| Time-range presets registry | `web-common/src/features/dashboards/time-controls/new-time-controls.ts:53-62` | `rill-PDC` ("Yesterday") **is in the list** — preset exists, just not surfaced as a default or in the quick-filter dropdown on Bratrax dashboards. |
| Time-range resolver (timezone-aware) | `web-common/src/features/dashboards/time-controls/rill-time-ranges.ts:51` | `resolveTimeRanges()` accepts `timeZone` parameter. Bratrax dashboard code never passes the user timezone. One-line fix per call site. |
| Pivot empty state | `web-common/src/features/dashboards/pivot/PivotEmpty.svelte` | The only empty state with hardcoded copy ("No data to show…"). No shared `EmptyState` primitive. Plan introduces one. |
| Existing telemetry registry | `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` | Extend with onboarding events. |
| Telemetry handler pattern | `web-common/src/metrics/BehaviourEventHandler.ts:27-99` | Add `fireOnboardingEvent` method following same pattern. |
| Measure/dimension `description` field | `runtime/parser/parse_metrics_view.go:37-73`, `proto/rill/runtime/v1/resources.proto` | Already exists; underused. |
| KPI tile (already wraps Tooltip) | `web-common/src/features/canvas/components/kpi/KPI.svelte:170-308` | Add info icon that surfaces `measure.description`. |
| Chart description rendering | `web-common/src/features/canvas/ComponentHeader.svelte:60-83` | Supports subtitle + tooltip variants. |
| Header breadcrumb + dashboard switcher | `web-common/src/components/navigation/breadcrumbs/BreadcrumbItem.svelte:62-142` | Already uses display names. |
| Role gating pattern | `web-admin/src/routes/[organization]/+layout.svelte:8`, `web-admin/src/features/organizations/OrganizationTabs.svelte:13-28` | Reuse for top-nav role filtering. |

### What does NOT exist

- **No customer-journey / order-detail view.** Searched `customer_journey`, `customerJourney`, `OrderDetail`, `OrderView`, `JourneyView`: zero matches. The user's "click order → audit log" experience is not a routing bug, it's a missing feature. Flag to product roadmap.
- **No tour/coachmark library.** `intro.js`, `shepherd`, `driver.js`, `coachmark`, `spotlight`, `guidedTour`: zero matches.
- **No shared `EmptyState` component.** Each empty state is hand-built.
- **No written getting-started guide.** Confirmed by Alessio's direct ask.
- **No server-side per-user preferences store** for checklist state, tour dismissal, etc. `WelcomeBanner.svelte` uses localStorage.

---

## P0 — Tier 1 UX defects (ship first, ship in week 1)

These are not learning problems. They are bugs and bad defaults that make users believe the product is hard. Each one is small. Together they remove a significant fraction of the "steep learning curve" complaint at near-zero engineering cost.

### P0.1 — "Today" default returns nulls

**Problem:** Bratrax syncs at end-of-day. When a user lands on a dashboard with "Today" as the active range, every KPI shows `null` and the channel table is empty. New users believe the product is broken.

**Investigation finding:** `ziva-project/dashboards/campaign_deep_dive.yaml:9` has `time_range: 2026-03-22 to 2026-03-29` (a hardcoded historical week — separate issue, see P0.2). The "Today" default the user hit comes from the time-control preset selector defaulting to `rill-TD`, defined in `web-common/src/features/dashboards/time-controls/new-time-controls.ts`.

**Fix:**
- Change `default_time_range` to `rill-P7D` (last 7 days) on every Bratrax canvas dashboard YAML under `ziva-project/dashboards/`.
- For dashboards that show end-of-day-synced data only: explicitly remove `rill-TD` ("Today") from the preset list, or replace its handler with a soft empty state ("Today's data syncs tonight — try Yesterday or Last 7 days").

### P0.2 — Hardcoded historical date range on Campaign Deep Dive

**Problem:** `campaign_deep_dive.yaml:9` is pinned to `2026-03-22 to 2026-03-29`. New users see two-month-old data and assume their sync is broken.

**Fix:** Remove the hardcoded `time_range` line. Rely on `default_time_range: rill-P7D` from P0.1.

### P0.3 — User timezone not applied to dashboard queries

**Problem:** Martin set timezone to EDT in Settings. Dashboards still display UTC. Confirmed: `AccountInfo.timezone` (`web-local/src/lib/bratrax/settings/types.ts:6`) is captured and persisted via `updateAccount()` (`web-local/src/routes/settings/+page.svelte:42-47`), but the frontend dashboard code never passes it to `resolveTimeRanges()` (`web-common/src/features/dashboards/time-controls/rill-time-ranges.ts:51` accepts a `timeZone` parameter).

**Fix:** Plumb `AccountInfo.timezone` from the user settings store into the dashboard time-control props. Find every call site of `resolveTimeRanges()` and pass the user timezone. One-line fix per call site; the risk is missing a call site — write a regression test that loads a dashboard, changes user timezone, and asserts that the time labels shift.

### P0.4 — "Yesterday" missing from quick-filter dropdown

**Problem:** Martin needs "yesterday" data daily and has to use custom-select because it's not in the dropdown.

**Investigation finding:** `rill-PDC` ("Yesterday") IS defined in `new-time-controls.ts:53` — the preset exists, it just isn't surfaced as a quick-filter option in Bratrax dashboards.

**Fix:** Audit the time-range pickers used on Bratrax dashboards (`SuperPill.svelte` family) and ensure `rill-PDC` is in the visible preset list. Likely a configuration tweak per dashboard, not a new component.

### P0.5 — Dashboard flash / disappear bug

**Problem:** Martin reports on first-day-of-campaign data the "report flashes and disappears every few seconds."

**Fix:** Reproduce against a fresh campaign with <24h of data. Likely a query that returns transiently empty results during a loading state, re-renders, then resolves. Investigate `KPI.svelte` and the canvas data-fetch lifecycle. Track this as a P0 bug ticket separately from the onboarding plan; do not block onboarding work on it but resolve in the same week.

### P0.6 — Click order → audit log instead of customer journey

**Investigation finding:** The customer-journey view does not exist at all. The user's experience is the result of a row-click handler routing to the only available drill-down (the audit log).

**Two-part fix:**
- **Immediate (P0):** Either disable the row-click handler on the orders/channel table OR change its tooltip to set expectation ("Click to view audit details"). Do not let users click and feel deceived.
- **Roadmap (Tier 5):** Build the customer-journey view. Flag to product. Not in scope for this plan.

### P0.7 — Cross-panel attribution-model labels missing

**Problem:** A user labeled the top tile "Correct" and the bottom table "Wrong" because the two showed different Blended Spend. The most likely root cause is different attribution models applied to different panels (or different filter scopes), with no UI cue explaining which model each panel uses.

**Investigation finding:** Both panels in `campaign_deep_dive.yaml` read from the `cross_metrics` metrics view with `cross_metrics: (attribution_model IN ('last_touch'))` set at line 11 as a dashboard-level filter. If the panels truly use the same model and still produce different numbers, the discrepancy is from different measure logic (e.g., `Blended Spend` is platform-reported, `Channel Attribution > Spend` is attributed-share). Either way, the user has no way to tell.

**Fix:**
- Add a small attribution-model label under every panel title: `Model: Last touch ∙ Spend definition: Platform-reported` (text varies by panel).
- Add a single "Why these don't tie" tooltip on the dashboard header that explains platform-reported vs. attributed spend in plain English.
- If panels truly use different models, surface that explicitly per panel.

### P0.8 — `.env` and `claude_system_prompt.txt` exposed in admin file tree

**Problem:** The admin/super-admin file tree (left sidebar) renders `.env` and `claude_system_prompt.txt` as clickable files. `.env` is a credential file — leaking it during a screenshare is a security incident, not a UX issue.

**Fix:** Add an explicit denylist to the file-tree component (identify in `web-local/`; `web-common/src/layout/navigation/Navigation.svelte` is the main navigation entry). Deny: `.env`, `.env.*`, `claude_system_prompt.txt`, `.gitignore`, anything under `.bratrax/` if introduced. For files admins legitimately edit (`rill.yaml`), surface through a dedicated **Project settings** panel instead.

### P0.9 — `SUPERADMINS` tab visible to non-admins

**Problem:** Screenshot 3 (non-admin view) shows the `SUPERADMINS` tab in the top nav. Non-admins should not see it.

**Fix:** Gate the tab on the existing `organizationPermissions` already plumbed through (`web-admin/src/routes/[organization]/+layout.svelte:8`, pattern at `web-admin/src/features/organizations/OrganizationTabs.svelte:13-28`).

### P0.10 — `ASK CLAUDE` button label

**Problem:** `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte:50` reads `ASK CLAUDE →`. Claude is an implementation detail (the chat backend is agent-agnostic per `web-common/src/features/chat/core/conversation.ts`).

**Fix:** `ASK CLAUDE →` → `ASK BRATRAX →`. This file is removed when the checklist ships in P1; the label change is an interim fix.

---

## P1 — The big rocks

### P1.1 — Write the getting-started guide (highest impact per hour of work)

**Why first:** Alessio asked for this directly. Looms are not enough for non-technical users who want to read-then-do. A written guide is also the prerequisite for the Resources block in the checklist and for empty-state copy that links out.

**Shape:**
- A single web page (not a PDF, not a multi-page docs site). Indexable in help search.
- Sections mirror the checklist sections: Connect → Configure → Track → First look → Invite.
- Embed Looms inline where useful, but the page is readable without watching anything.
- Tone: humble, user-blameless, engineer-not-salesperson (see "Voice and tone").

**Where to publish:** decide between (a) `docs.bratrax.com` (new) or (b) an in-app help route. Recommend (b) for v1 — `web-admin/src/routes/help/getting-started/+page.svelte` (or under `web-local/` if served from there). Link from the in-app Help tab and from every checklist item.

### P1.2 — Onboarding checklist (replaces `WelcomeBanner`)

**Shape:** floating bottom-right widget (Intercom-style), hybrid auto/manual completion, replaces the existing "Your analytics are live!" banner.

**Structure: 5 sections, 18 items.** Sections compound: hitting Section 4 with Section 2 unfinished produces a wall (null data). Allow users to mark anything complete to bypass.

#### Section 1 — Connect your data
*(All auto-checkable from existing connector state.)*
1. Connect Shopify
2. Enable Shopify theme tracker *(currently surfaced as the "Action Required" callout on Connectors page — reuse that signal)*
3. Connect at least one ad platform (Google / Meta / TikTok / Microsoft when available)
4. Connect Klaviyo *(optional, skippable)*
5. Install pixel on external landing pages *(skippable for Shopify-only stores)*

#### Section 2 — Configure your business
*(The most "learning curve" complaints come from these being non-obvious. Almost all auto-checkable.)*
6. **Set your timezone** *(currently buried in Settings → Account; promote here. Auto-completes when `AccountInfo.timezone` differs from system default.)*
7. Add Cost of Goods *(per-variant if/when supported — see Tier 5)*
8. Add shipping costs
9. Add custom / operating expenses
10. **Connect your Anthropic API key** *(currently buried in Settings → AI; promote here. BYOK confirmed.)*

#### Section 3 — Set up paid-media tracking
*(Reuses existing `TrackingTemplateGuide.svelte` and `tracking-templates.ts`. Section is mostly surfacing those.)*
11. Install Google Ads tracking template *(copy/paste from existing Connectors UI)*
12. Install Meta Ads URL parameters *(copy/paste from existing Connectors UI)*
13. Run the "Verify after setup" checklist *(the 4 existing items in `tracking-templates.ts:11-16`)*

#### Section 4 — Take your first look
*(The activation section. Each item opens a guided first-time interaction. Section 4 is the biggest engineering lift — it depends on the tour/coachmark infrastructure in P1.3.)*
14. Open your Campaign Deep Dive *(first-visit 3-tooltip tour: KPI grid, attribution model, drill into a channel)*
15. Drill into one channel and one campaign *(auto-completes on drill event)*
16. Switch the attribution model once *(auto-completes on filter change for `attribution_model`)*
17. Ask Bratrax one question *(opens AI panel pre-loaded with 3 example prompts: "How is my ROAS this week?", "Which campaign drove the most new customers in May?", "Should I shift spend from Google to TikTok?")*

#### Section 5 — Bring your team
18. Invite a teammate

#### Auto-check vs. manual mark

| Auto-checkable | Manual-marked |
|---|---|
| Items 1, 2, 3, 4 (connector state from backend) | Items 5, 11, 12, 13 (visiting a guide ≠ doing the install — but offer auto-confirm when ingestion sees the UTM) |
| Item 6 (timezone changed from default) | Item 14 (visiting ≠ understanding) |
| Item 7 (≥ N non-zero COGS rows) | |
| Items 8, 9, 10 (form-save events) | |
| Items 15, 16, 17 (frontend interaction events) | |
| Item 18 (invite-sent event) | |

#### Cross-system dependency

The Bratrax backend must expose `GET /v1/onboarding/status` returning `{ shopify_connected, theme_tracker_enabled, ad_platforms: string[], klaviyo_connected, cogs_rows: int, anthropic_key_set, team_invites_sent: int, ... }`. Frontend polls every 30s while checklist is open. Without this endpoint, items 1-3, 7, 10 fall back to manual marking. Plan ships with manual fallback in v1; backend hardens in v1.1.

#### Placement

- **Floating bottom-right widget** on every authenticated page.
- **Collapsed by default** once the user dismisses the expanded view: one-line `Setup: 7/18 ▾`.
- **Auto-promotes itself** when an empty-state panel renders and the relevant checklist item is incomplete (see P1.4).
- **Resources block at the bottom of the expanded view:** Read the getting-started guide, Watch the setup walkthrough (Loom), Ask Bratrax, Email support.

#### Files to create / modify

- New: `web-local/src/lib/bratrax/onboarding/OnboardingChecklist.svelte` — the floating widget.
- New: `web-local/src/lib/bratrax/onboarding/checklist-items.ts` — the 18-item registry plus completion-signal definitions.
- New: `web-local/src/lib/bratrax/onboarding/checklist-store.ts` — Svelte store backed by server-side preferences (see P1.7) plus polling against the backend status endpoint.
- Extend: `web-local/src/lib/bratrax/onboarding/api.ts` — add `getOnboardingStatus()` client.
- Delete: `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte` (replaced; banner copy folded into checklist's Section-1 header card).
- Modify: every `import WelcomeBanner` callsite (grep before deleting).

### P1.3 — First-visit tooltip / coachmark infrastructure

**Why:** Section 4 of the checklist needs in-app tours. No tour library exists. The biggest engineering lift in this plan.

**Approach:** Lightweight in-house. Reuse the existing tooltip primitives in `web-common/src/components/tooltip/`. No external dependency.

**API shape:**
```ts
showTour({
  id: 'campaign-deep-dive-first-visit',
  steps: [
    { target: '[data-tour="kpi-grid"]', body: '…' },
    { target: '[data-tour="attribution-selector"]', body: '…' },
    { target: '[data-tour="channel-table"]', body: '…' },
  ],
})
```

**Constraints:**
- Dismissal persists via server-side preferences (P1.7), NOT localStorage.
- A tour and the checklist are never shown at the same time. Tour is for first-visit orientation; checklist is for first-week activation.
- Three tours in v1: Campaign Deep Dive, Performance Overview, Customer Analytics.

**Files:**
- New: `web-local/src/lib/bratrax/onboarding/DashboardTour.svelte`
- New: `web-local/src/lib/bratrax/onboarding/tour-registry.ts` — the three v1 tours.
- New: `web-local/src/lib/bratrax/onboarding/user-preferences.ts` — shared client for the preferences API (also used by the checklist).

### P1.4 — Empty-state sweep + checklist linkage

**Problem:** Empty states currently have hardcoded, terse copy (`PivotEmpty.svelte`: "No data to show for the selected filters"). They give the user no idea what to do next.

**Plan:**
- New shared `<EmptyState>` component in `web-common/src/components/empty-state/EmptyState.svelte` with props: `title`, `body`, `primaryAction`, `relatedChecklistItemId`.
- When `relatedChecklistItemId` is set, the empty state renders a CTA: *"Finish setting up tracking templates →"* that pops the checklist open at the relevant item.
- Replace `PivotEmpty.svelte` and every ad-hoc empty state with `<EmptyState>`.
- Copy follows the voice/tone guidance below.

**Specific instances to audit (non-exhaustive):**
- "Today" / no-data-yet (link to: wait or change range).
- No connector configured (link to: checklist Section 1).
- No tracking template installed (link to: checklist Section 3).
- No COGS entered (link to: checklist item 7).

### P1.5 — Metric literacy layer

**Problem:** KPI tiles (MER, NC ROAS, Blended CPA, Contribution Margin, Attributed Sales, etc.) have no plain-English definition. Some charts already have helpful subtitles (NC ROAS, NC CPA, CPM, CTR) but the pattern is inconsistent.

**Plan:**
- Infrastructure exists: `ComponentHeader.svelte:60-83` supports subtitle + tooltip-icon variants; `KPI.svelte:170-308` wraps tiles in a `Tooltip`; measures already have a `description` field in the metrics-view YAML schema (`parse_metrics_view.go:37-73`).
- **Action 1:** Add a small info `(i)` icon to the KPI tile header. When `measure.description` is non-empty, hover opens a tooltip. Modify `KPI.svelte` around lines 189-196.
- **Action 2:** Audit every metrics-view YAML under `ziva-project/metrics/` and ensure every measure has a Bratrax-quality `description`. Format: *"<plain-English definition>. Formula: <formula>. <When to use it>."*
- **Action 3:** For canvas charts, ensure every chart's `description` field is populated. The 5 charts that already have subtitles prove this works — extend to the others.

**Metrics that need v1 descriptions:** MER, ROAS, NC ROAS, CPA, NC CPA, Blended Spend, Blended CPA, Contribution Margin, Attributed Sales, CPM, CTR, AOV, New Purchases, NC Sales, Repeat Orders, MRR.

### P1.6 — Promote the attribution-model selector

**Problem:** "Attribution Model: last_touch" appears as a generic dimension-filter chip (`web-common/src/features/dashboards/filters/dimension-filters/DimensionFilterReadOnlyChip.svelte`). It's the single most consequential control on the dashboard and looks identical to a date filter.

**Plan:**
- New `AttributionModelSelector.svelte` in `web-local/src/lib/bratrax/` — labeled dropdown above the KPI grid: `Attribution model: ▾ Last touch`.
- Each option carries a one-line "when to use" description.
- Reads/writes the `attribution_model` dimension filter via the existing dashboard state store. Behavior unchanged.
- Hide the filter chip for `attribution_model` specifically.

Complements P0.7 (cross-panel labels) — together they make the attribution model a first-class, visible control rather than a hidden filter chip.

### P1.7 — Server-side per-user preferences

**Why:** Both the checklist (P1.2) and the tour (P1.3) need durable per-user dismissal/completion state. localStorage is insufficient — users on multiple devices need dismissals to survive.

**Plan:**
- Define a simple schema in the Bratrax backend: `user_preferences(user_id, key, value_json, updated_at)`.
- Single endpoint pair: `GET /v1/user/preferences/:key`, `PUT /v1/user/preferences/:key`.
- Frontend client at `web-local/src/lib/bratrax/onboarding/user-preferences.ts`.
- Keys used in v1: `onboarding_checklist_state`, `onboarding_tour_dismissed:<tour_id>`.

### P1.8 — Telemetry funnel

**Why:** Cannot measure whether anything above moves the needle without it.

**Extend** `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` with:
- `onboarding_checklist_shown`
- `onboarding_item_completed` (payload: `item_id`, `section`, `auto: bool`)
- `onboarding_section_completed` (payload: `section`)
- `onboarding_checklist_dismissed`
- `onboarding_tour_started` (payload: `tour_id`)
- `onboarding_tour_step_completed` (payload: `tour_id`, `step`)
- `onboarding_tour_dismissed` (payload: `tour_id`, `step_at_dismiss`)
- `metric_tooltip_opened` (payload: `metric_name`)
- `attribution_model_changed` (payload: `from`, `to`, `panel`)
- `empty_state_cta_clicked` (payload: `state_id`, `checklist_item_id`)
- `data_freshness_clicked`
- `first_dashboard_view` (payload: `dashboard_name`)

Reuse the existing `BehaviourEventHandler` pattern (`web-common/src/metrics/BehaviourEventHandler.ts:27-99`). Add `fireOnboardingEvent(action, payload)`.

---

## P2 — Polish and structural cleanup

### P2.1 — Dashboard nav discoverability (Justin's gap)

**Problem:** Justin found one dashboard and assumed that was all. The breadcrumb dashboard switcher (`BreadcrumbItem.svelte:62-142`) exists but is non-obvious to a non-technical user.

**Plan:**
- Add a left rail of dashboard tiles on the org Home page (the page rendered when a user clicks "DASHBOARDS" in the top nav with no specific dashboard selected). Each tile: name, description, sparkline preview, last-viewed timestamp.
- Inline `data-tour` anchors so the Section 4 tour can teach users to switch dashboards.

### P2.2 — Date range UX cleanup

**Plan:**
- Collapse the two stacked controls (`Last 7 Days …` and `as of latest day end`) into a single picker with presets: Last 7 / 28 / 90 days, MTD, QTD, YTD, Custom — plus Yesterday (P0.4).
- Replace "as of latest day end" with a small `Data updated 12 min ago` stamp sourcing the timestamp from runtime metadata.
- Move the "Comparing: Previous period" toggle adjacent to the date control.

**Files:** `web-common/src/features/dashboards/time-controls/super-pill/SuperPill.svelte`, `RangeDisplay.svelte`.

### P2.3 — Role-aware top navigation

**Plan:**
- Brand operators: Dashboards, Connectors (read-only), Help, Ask Bratrax.
- Brand admins also see: Cost Settings, Settings.
- Super admins also see: Superadmins.
- Reuse the `organizationPermissions` pattern from `OrganizationTabs.svelte:13-28`.

P0.9 (hide `SUPERADMINS` from non-admins) is the urgent subset; the full role-aware nav is P2.

### P2.4 — Admin sidebar overhaul (super-admin view)

**Problem:** From screenshot 2 — the admin file tree uses dbt-style names (`dim_*`, `fct_*`), truncates everything with `...`, has no search, exposes `rill.yaml`.

**Plan:**
- **a)** Complete the `rill.yaml` → `bratrax.yaml` rename. Option A (rename file, requires parser support) preferred; Option B (UI relabel only) ships first.
- **b)** Display names in the sidebar (read `Display name:` from each YAML), with hover tooltip showing file name for power users.
- **c)** Sidebar search + type filter (Dashboards / Metrics views / Models / Sources / Connectors).
- **d)** "View as" toggle for super admins: `Admin ▾` with `Admin`, `Brand operator`, `Viewer` options. Non-admin modes hide the left file tree, the right Canvas configurations panel, the `</>` icon, the `+` add buttons, and the editor controls — matching the chrome in screenshot 3.

### P2.5 — Rebrand sweep

Outside the P0.8 / P0.10 immediate fixes, the broader Rill → Bratrax cleanup:

**a) Centralize brand strings.** Create `web-common/src/lib/branding.ts`: `PRODUCT_NAME`, `PRODUCT_DOMAIN`, `DOCS_URL`, `AI_ASSISTANT_NAME`. All user-facing references read from there.

**b) Specific replacements:**
- `web-admin/src/features/billing/plans/WelcomeToRillCloudDialog.svelte` — rename file + title.
- `web-admin/src/routes/+layout.svelte:132` — `<meta name="description" content="Rill Cloud">` → `"Bratrax"`.
- `web-admin/src/features/projects/status/overview/DeploymentSection.svelte` — "Rill-managed" label.
- All hardcoded `docs.rilldata.com` / `www.rilldata.com` / `ui.rilldata.com` links — route through `DOCS_URL` constant.

**c) Replace welcome-page example projects.** `web-common/src/features/welcome/constants.ts` lists `rill-cost-monitoring`, `rill-openrtb-prog-ads`, `rill-github-analytics`. Recommend gating to super-admin role only; brand operators should never see them.

**d) `EMPTY_PROJECT_TITLE` in `web-common/src/features/welcome/constants.ts:1`** — currently `"Untitled Bratrax Project"`, which is correct; verify it surfaces correctly.

---

## P3 — Flagged to other teams (NOT this plan)

These are NOT onboarding fixes. They are flagged here so the plan owner can route them to the right team and so this document is a complete picture of "what would make the learning curve disappear."

### P3.1 — UTM auto-detection for incumbent attribution tools (Tier 3, structural switching cost)

**Problem:** Customers migrating from Triple Whale / Northbeam / Hyros / Redtrack have hundreds-to-thousands of existing ads tagged with the incumbent's UTM format. Rewriting them would reset Meta learning mode and tank performance. They are effectively locked into their incumbent.

**Fix:** Bratrax ingestion auto-detects and maps UTM formats from the major incumbents. **Flag to Bobo / Drasko / Maksim (backend team).**

### P3.2 — Customer-journey-per-order view (Tier 5, feature gap)

**Problem:** Martin needs an equivalent to Triple Whale's per-order journey view. Confirmed: does not exist in the codebase.

**Fix:** Build. Flag to product roadmap. P0.6 ships a stopgap that prevents the misleading click-through.

### P3.3 — Per-variant Cost of Goods (Tier 5, feature gap)

Flag to product roadmap.

### P3.4 — Microsoft Ads connector (Tier 5, feature gap)

Flag to backend roadmap.

### P3.5 — Product-level Shopping / PMAX views (Tier 5, feature gap)

Flag to product roadmap.

---

## Voice and tone for all onboarding content

Brat's response to Alessio set the template:

> "We want to be honest: if it wasn't easy, we failed you. Bratrax Lite was built specifically for small teams without developers, and a steep learning curve is exactly what we were trying to eliminate. So if that was your experience, something went wrong on our end."

Humble, user-blameless, engineer-not-salesperson. Applies to:

- The written getting-started guide.
- Empty-state copy.
- Tooltip text.
- Error messages.
- The checklist's own copy.

Connects to the cross-promo framing locked in earlier: *"We're engineers trying to build a product, not run a sales operation. The tool reflects how we work — solve a real problem, document the trade-offs, name the limits."* Specific anti-patterns to avoid:

- "Easy!" / "Simple!" / "In just X clicks!" — patronizing.
- Imperative shouting ("CONNECT NOW") — sales voice.
- Apologetic over-explaining — undermines trust.
- Faux-friendly mascot speak ("Oops! It looks like…").

Better:

- "No data yet for the date range you picked. Try the last 7 days, or finish setting up tracking templates."
- "We sync once every 1-3 hours, so today's data won't appear until tomorrow morning."
- "Attribution model: last-touch. Blended Spend on this tile is platform-reported; the channel table below uses attributed share, which is why totals don't tie."

---

## Suggested rollout sequence

1. **Week 1 — P0 sweep:** P0.1 (date defaults), P0.2 (hardcoded range), P0.3 (timezone plumbing), P0.4 ("Yesterday" preset), P0.7 (cross-panel labels), P0.8 (hide `.env`), P0.9 (hide `SUPERADMINS` for non-admins), P0.10 (`ASK CLAUDE` label). P0.5 (flash bug) tracked separately; P0.6 (order click stopgap) in parallel.
2. **Week 1-2 — P1.1 written getting-started guide.** Independent of code. Ship in parallel with the P0 sweep.
3. **Week 2 — P1.5 metric literacy + P1.4 empty-state sweep.** Content-heavy, low code risk. Together these address the bulk of "I don't know what I'm looking at" complaints.
4. **Week 2-3 — P1.7 server-side preferences + P1.8 telemetry.** Foundational for the checklist and tours.
5. **Week 3-4 — P1.2 checklist (static + hybrid auto-detect) + P1.6 attribution selector.** Backend dependency for full auto-detection; ship hybrid v1, harden in v1.1.
6. **Week 4-5 — P1.3 tour infrastructure + Section 4 tours.** Largest engineering lift.
7. **Week 5+ — P2 polish and structural cleanup.** Ship behind feature flags.
8. **Ongoing — P3 routed to other teams.** Backend (UTM, Microsoft Ads), product (journey view, per-variant COGS, PMAX).

Two rules across the rollout:

- **Telemetry first.** Don't ship P1.2 without P1.8 — you can't measure success otherwise.
- **Backend dependencies fall back to manual.** Every checklist item, every empty-state link, must work in a degraded mode where the backend status endpoint is missing.

---

## Open questions to resolve before / during implementation

1. Where exactly does the post-signup user land today? `routes/+layout.ts:38` redirects authenticated `/` traffic to `/developer`; the onboard routes (`/onboard/shopify` etc.) have resume logic. Trace the full flow for a brand-new user vs. a returning user with partial setup.
2. Should the checklist replace the existing `/onboard/*` wizard, sit alongside it, or wrap it? Recommendation: wizard stays as the heavy-lifting setup; checklist surfaces remaining items + Section 4 + Section 5 on every page after the wizard exits.
3. What email-based onboarding (welcome series, day-3 nudge) is already running? If yes, what does it say, and should the checklist mirror its structure?
4. P1.7 user-preferences storage: does the Bratrax backend already have a user-settings table we can extend, or is this a new table?
5. P0.7 cross-panel labels: are the two panels actually using different attribution models, or the same model with different measure definitions? Need a runtime read to confirm.
6. P3.2 customer-journey view: roadmap timing affects how strong the P0.6 stopgap needs to be. If the view ships in 4 weeks, a benign tooltip is enough; if 6 months, consider a more elaborate placeholder.

---

## Verification

**P0 sweep:**
- Sign up a fresh test account. Land on Campaign Deep Dive. Verify default range is "Last 7 days" with real data, not "Today" with nulls.
- Set timezone to EDT in Settings. Reload a dashboard. Verify times shown are EDT and the time-range picker resolves "Last 7 days" in EDT.
- Open the quick-filter dropdown. Verify "Yesterday" appears.
- Open Campaign Deep Dive. Verify each panel shows a `Model: …` label and a "Why these don't tie" tooltip.
- Sign in as a super admin. Verify `.env` and `claude_system_prompt.txt` do not appear in the file tree.
- Sign in as a non-admin. Verify the `SUPERADMINS` tab is not visible.
- Verify the welcome banner reads `ASK BRATRAX →` (until the checklist replaces it).

**P1.1 guide:** A first-time user can read it in under 15 minutes and answer "how do I connect Meta?", "how do I add COGS?", "what does NC ROAS mean?", "what timezone is my data in?".

**P1.2 checklist:**
- Fresh signup → checklist appears in bottom-right.
- Section 4/5 items show as locked or "complete Section 1-3 first" until prerequisites are met.
- Connecting Shopify in the backend auto-flips item 1 within 30s (with hybrid v1; in manual v1, user clicks "Mark complete").
- Click "Mark complete" on item 7. Reload. Sign in from a second browser. Verify still complete (server-side persistence).
- Open a dashboard. Switch attribution model. Verify item 16 auto-completes.
- Dismiss the checklist. Sign out and in. Verify still dismissed.

**P1.3 tours:** First-ever view of Campaign Deep Dive shows the 3-step tour. Dismiss. Verify it does not re-appear, and that signing in on a second device respects the dismissal.

**P1.4 empty states:** Force an empty channel table (e.g., apply a filter that returns nothing). Verify the new `<EmptyState>` renders with helpful copy and, where applicable, a CTA linking to the relevant checklist item.

**P1.5 metric literacy:** Hover the info icon on every KPI tile on every default dashboard. Verify a tooltip appears with a non-empty description.

**P1.6 attribution selector:** Verify the labeled dropdown appears above the KPI grid; the dimension-filter chip for `attribution_model` no longer appears in the chip row; switching the model re-renders every panel.

**P1.8 telemetry:** Build an activation funnel in the analytics warehouse: `bratrax_first_login` → `onboarding_section_completed[section=1]` → `onboarding_section_completed[section=4]` → 7-day retention. This is the primary metric for whether this entire effort worked.

**Tests:**
- `npm run test -w web-common` after changes to shared components (`EmptyState`, `KPI.svelte`, time-controls).
- `npm run test -w web-local` Playwright e2e for the checklist appearance, dismissal, and a Phase-1-auto-complete flow (mock the backend endpoint).
- `go test ./...` after runtime/parser changes (only if option A of P2.4a is taken).
- `npm run quality` for lint/format before each PR.

---

## End of plan

This document is intended as a developer hand-off. Each P0 / P1 / P2 / P3 item should become an issue. The five-tier framing should travel with the issues so reviewers can recognize that a "small UX defect" is doing strategic work.
