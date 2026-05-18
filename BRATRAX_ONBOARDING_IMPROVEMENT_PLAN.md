# Bratrax onboarding UX improvement plan

## Context

Bratrax Lite is an analytics/attribution product for Shopify DTC brands, launched May 12, 2026. Founding 100 cohort is filling (~8 of 33 onboarded mid-May), pricing locked at ~$79/month for life. The product is a Bratrax-branded fork of Rill (this repo). The Bratrax backend (separate Go service) handles OAuth into ad/commerce platforms and computes attribution; the Rill frontend renders the resulting dashboards.

The recurring theme in user feedback is "steep learning curve." Investigation showed that complaint disaggregates into several distinct problems, only some of which are onboarding problems. **This plan covers the onboarding problems only.** UI bugs, data-display inconsistencies, admin chrome cleanup, and missing-feature requests are being handled by other tracks.

### Onboarding problems in scope

1. **No written getting-started material.** Alessio (cancelled customer, later re-engaged) asked directly: *"Do you have any onboarding material? A written guide, a walkthrough, anything that explains the basics step by step?"* The product currently ships with no written guide.
2. **Discoverability gap.** Justin (active user) found one dashboard, got real value, and assumed that was the whole product: *"Beyond the campaign deep dive page, I'm not sure where else to find usable insights."* He's using a small fraction of the product and doesn't know what he's missing.
3. **Configuration steps non-obvious.** Several setup items that materially change the user's experience are buried in Settings (timezone, Anthropic API key, costs). New users don't know they need to do them.
4. **Activation drop-off.** Users complete part of setup, abandon before seeing value. No structured nudge gets them back to the next step.
5. **No in-product orientation.** When a user lands on a dashboard, nothing teaches them how to read it, drill into it, or change the attribution model.

### Also in scope: admin and super-admin chrome

Super admins (the Bratrax team and agency operators) are users too. Their first run is also part of onboarding — and right now they land in the Rill IDE chrome with a file tree, code editor, and sensitive files exposed. Cleaning this up is a Tier 2 discoverability fix for the admin population and a security fix at the same time.

### Out of scope (other teams / tracks are handling these)

- Specific UI bugs reported by users (e.g., date defaults, timezone plumbing, dashboard flash bug, hardcoded ranges, missing time presets).
- Cross-panel data-display inconsistencies and attribution-label work.
- Broader Rill → Bratrax rebrand sweep across web-admin billing dialogs, meta tags, etc.
- Feature gaps (per-variant COGS, Microsoft Ads, PMAX views, customer-journey-per-order view).
- Structural switching costs (UTM auto-detection from incumbent attribution tools).

### Intended outcome

A non-technical Shopify operator can sign up, complete a guided setup in under 20 minutes, land on a dashboard whose purpose is clear, understand what they're looking at, and discover every other dashboard within their first session — while feeling that the product respects their time.

---

## What already exists (do not rebuild)

| Existing surface | Path | Relevance to onboarding |
|---|---|---|
| Multi-step onboarding wizard with resume logic | `web-local/src/routes/onboard/` — subroutes `shopify`, `business`, `payment`, `stack`, `embed`, `loading` | The team already built a wizard for initial setup. The new checklist must integrate with this, not replace it. Resume logic at `web-local/src/routes/+layout.ts:68-120`. |
| Bratrax welcome banner | `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte` | The current "Your analytics are live!" banner. To be replaced by the checklist; its content folds into the checklist's initial state. |
| Bratrax onboarding API client | `web-local/src/lib/bratrax/onboarding/api.ts` (11 KB) | Existing backend integration. Extend, don't reimplement. |
| Tracking template guide | `web-local/src/lib/bratrax/TrackingTemplateGuide.svelte` | The Google/Meta URL Templates UI. Section 3 of the checklist surfaces this. |
| Tracking verification checklist | `web-local/src/lib/bratrax/tracking-templates.ts:11-16` | A parameterized 4-step "Verify after setup" list. Reuse directly. |
| Settings — AI tab (Anthropic key BYOK) | `web-local/src/routes/settings/+page.svelte:30-36` | The Anthropic key entry point. Promote into the checklist (item 10). |
| Settings — Account tab (timezone) | `web-local/src/routes/settings/+page.svelte:45-76` | The timezone selector. Promote into the checklist (item 6). |
| Connectors page | `web-local/src/routes/connectors/+page.svelte` | Existing hub for Shopify, ad platforms, URL templates. Checklist links into specific sections. |
| Header breadcrumb + dashboard switcher | `web-common/src/components/navigation/breadcrumbs/BreadcrumbItem.svelte:62-142` | Already uses display names — referenced by the discoverability work. |
| KPI tile (already wraps Tooltip) | `web-common/src/features/canvas/components/kpi/KPI.svelte:170-308` | Add an info icon that surfaces `measure.description` for metric literacy. |
| Chart description rendering | `web-common/src/features/canvas/ComponentHeader.svelte:60-83` | Already supports subtitle + tooltip variants. |
| Measure/dimension `description` field | `runtime/parser/parse_metrics_view.go:37-73`, `proto/rill/runtime/v1/resources.proto` | Schema already supports per-measure descriptions; underused. |
| Existing telemetry registry | `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` | Extend with onboarding events. |
| Telemetry handler pattern | `web-common/src/metrics/BehaviourEventHandler.ts:27-99` | Reuse for `fireOnboardingEvent`. |

### What does NOT exist

- **No written getting-started guide.** Confirmed by Alessio's direct ask.
- **No tour / coachmark library.** Search for `intro.js`, `shepherd`, `driver.js`, `coachmark`, `spotlight`, `guidedTour`: zero matches.
- **No server-side per-user preferences store** for checklist state or tour dismissal. `WelcomeBanner.svelte` uses localStorage, which is insufficient (users on multiple devices).

---

## Workstream 1 — Written getting-started guide

**Why first:** Alessio asked for this directly. Looms are not enough for non-technical users who want to read-then-do. The guide is also the prerequisite for the Resources block in the checklist and for empty-state copy that links out.

**Shape:**

- A single web page (not a PDF, not a multi-page docs site). Indexable in help search.
- Sections mirror the checklist sections: Connect → Configure → Track → First look → Invite.
- Embed Looms inline where useful, but the page is readable without watching anything.
- Tone: humble, user-blameless, engineer-not-salesperson (see "Voice and tone").

**Where to publish:** Decide between (a) `docs.bratrax.com` (new) or (b) an in-app help route. Recommend (b) for v1 — `web-admin/src/routes/help/getting-started/+page.svelte` (or under `web-local/` if served from there). Link from the in-app Help tab, from every checklist item, and from every onboarding email.

**Independent of code.** Can ship in parallel with all other workstreams.

---

## Workstream 2 — Onboarding checklist (the centerpiece)

**Shape:** floating bottom-right widget (Intercom-style), hybrid auto/manual completion. Replaces the existing `WelcomeBanner.svelte`. Sits alongside the existing `/onboard/*` wizard rather than replacing it: the wizard handles the heavy-lifting initial setup; the checklist surfaces remaining items, Section 4 (orientation), and Section 5 (team) on every page thereafter.

**Structure: 5 sections, 18 items.** Sections compound — hitting Section 4 with Section 2 unfinished produces a wall (the checklist explains why). Users can mark anything complete manually to bypass.

### Section 1 — Connect your data
*(All auto-checkable from existing connector state.)*

1. Connect Shopify
2. Enable Shopify theme tracker *(currently surfaced as the "Action Required" callout on Connectors page — reuse that signal)*
3. Connect at least one ad platform (Google / Meta / TikTok / Microsoft when available)
4. Connect Klaviyo *(optional, skippable)*
5. Install pixel on external landing pages *(skippable for Shopify-only stores)*

### Section 2 — Configure your business
*(These are non-obvious to first-time users. Almost all auto-checkable from save events.)*

6. **Set your timezone** *(currently buried in Settings → Account; promote here. Auto-completes when `AccountInfo.timezone` differs from default.)*
7. Add Cost of Goods
8. Add shipping costs
9. Add custom / operating expenses
10. **Connect your Anthropic API key** *(currently buried in Settings → AI; promote here. BYOK confirmed.)*

### Section 3 — Set up paid-media tracking
*(Reuses existing `TrackingTemplateGuide.svelte` and `tracking-templates.ts`. Section is mostly surfacing those.)*

11. Install Google Ads tracking template *(copy/paste from existing Connectors UI)*
12. Install Meta Ads URL parameters *(copy/paste from existing Connectors UI)*
13. Run the "Verify after setup" checklist *(the 4 existing items in `tracking-templates.ts:11-16`)*

### Section 4 — Take your first look
*(The activation section. Each item opens a guided first-time interaction. Depends on the tour infrastructure in Workstream 3.)*

14. Open your Campaign Deep Dive *(first-visit 3-tooltip tour: KPI grid, attribution model, drill into a channel)*
15. Drill into one channel and one campaign *(auto-completes on drill event)*
16. Switch the attribution model once *(auto-completes on filter change for `attribution_model`)*
17. Ask Bratrax one question *(opens AI panel pre-loaded with 3 example prompts: "How is my ROAS this week?", "Which campaign drove the most new customers in May?", "Should I shift spend from Google to TikTok?")*

### Section 5 — Bring your team

18. Invite a teammate

### Auto-check vs. manual mark

| Auto-checkable | Manual-marked |
|---|---|
| Items 1, 2, 3, 4 (connector state from backend) | Items 5, 11, 12, 13 (visiting a guide ≠ doing the install — but offer auto-confirm when ingestion sees the UTM) |
| Item 6 (timezone changed from default) | Item 14 (visiting ≠ understanding) |
| Item 7 (≥ N non-zero COGS rows) | |
| Items 8, 9, 10 (form-save events) | |
| Items 15, 16, 17 (frontend interaction events) | |
| Item 18 (invite-sent event) | |

### Cross-system dependency

The Bratrax backend must expose `GET /v1/onboarding/status` returning `{ shopify_connected, theme_tracker_enabled, ad_platforms: string[], klaviyo_connected, cogs_rows: int, anthropic_key_set, team_invites_sent: int, ... }`. Frontend polls every 30s while checklist is open. Without this endpoint, items 1-3, 7, 10 fall back to manual marking. **Plan ships with manual fallback in v1; backend hardens in v1.1.**

### Placement

- **Floating bottom-right widget** on every authenticated page.
- **Collapsed by default** once the user dismisses the expanded view: one-line `Setup: 7/18 ▾`.
- **Auto-promotes itself** when an empty-state panel renders and the relevant checklist item is incomplete (see Workstream 4).
- **Resources block at the bottom of the expanded view:** Read the getting-started guide, Watch the setup walkthrough (Loom), Ask Bratrax, Email support.

### Files to create / modify

- New: `web-local/src/lib/bratrax/onboarding/OnboardingChecklist.svelte` — the floating widget.
- New: `web-local/src/lib/bratrax/onboarding/checklist-items.ts` — the 18-item registry plus completion-signal definitions.
- New: `web-local/src/lib/bratrax/onboarding/checklist-store.ts` — Svelte store backed by server-side preferences (Workstream 8) plus polling against the backend status endpoint.
- Extend: `web-local/src/lib/bratrax/onboarding/api.ts` — add `getOnboardingStatus()` client.
- Delete: `web-local/src/lib/bratrax/onboarding/WelcomeBanner.svelte` (replaced; banner copy folded into checklist's Section-1 header card; "ASK CLAUDE" button label folds into the checklist's "Ask Bratrax" item).
- Modify: every `import WelcomeBanner` call site (grep before deleting).

---

## Workstream 3 — First-visit tour / coachmark infrastructure

**Why:** Section 4 of the checklist needs in-app tours. No tour library exists. This is the largest engineering lift in this plan.

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

- Dismissal persists via server-side preferences (Workstream 8), NOT localStorage.
- A tour and the checklist are never shown at the same time. Tour is for first-visit orientation; checklist is for first-week activation.
- Three tours in v1: Campaign Deep Dive, Performance Overview, Customer Analytics.

**Files:**

- New: `web-local/src/lib/bratrax/onboarding/DashboardTour.svelte`
- New: `web-local/src/lib/bratrax/onboarding/tour-registry.ts` — the three v1 tours.
- New: `web-local/src/lib/bratrax/onboarding/user-preferences.ts` — shared client for the preferences API (also used by the checklist).

---

## Workstream 4 — Empty-state linkage to checklist

**Scope:** Narrow. We are NOT refactoring all empty states; we are only adding checklist-linkage CTAs to empty states a new user is likely to hit during onboarding.

**Why:** When a user hits a "No data to show" panel because they haven't finished setup, the empty state should tell them which checklist item to complete. Otherwise the user concludes the product is broken.

**Plan:**

- Identify the 3-5 empty-state surfaces a new user is most likely to encounter: no connector configured, no tracking template installed, no COGS entered, no data for selected range.
- For each, render a small CTA bar inside the empty state: *"Finish setting up tracking templates →"* that pops the checklist open at the relevant item.
- Do NOT touch empty states that are unrelated to onboarding flow.

**Files:**

- Identify the existing empty-state usages during implementation (e.g., `web-common/src/features/dashboards/pivot/PivotEmpty.svelte`); add the optional `relatedChecklistItemId` prop pattern there.
- Wire the open-checklist-at-item handler into the global checklist store from Workstream 2.

---

## Workstream 5 — Metric literacy layer

**Why:** New users see acronyms like MER, NC ROAS, Blended CPA, Contribution Margin and have no in-product way to learn what they mean. This is the most common "I don't know what I'm looking at" complaint and directly affects whether a new user feels welcomed vs. lost.

**Plan:**

The infrastructure exists. We're activating dormant features.

- `ComponentHeader.svelte:60-83` already supports subtitle + tooltip-icon variants of a `description` field.
- `KPI.svelte:170-308` already wraps tiles in a `Tooltip`.
- Measures already have a `description` field in the metrics-view YAML schema (`parse_metrics_view.go:37-73`).

Three actions:

- **Action 1:** Add a small info `(i)` icon to the KPI tile header. When `measure.description` is non-empty, hover opens a tooltip. Modify `KPI.svelte` around lines 189-196.
- **Action 2:** Audit every metrics-view YAML under `ziva-project/metrics/` and ensure every measure has a Bratrax-quality `description`. Format: *"<plain-English definition>. Formula: <formula>. <When to use it>."*
- **Action 3:** For canvas charts, ensure every chart's `description` field is populated. The 5 charts that already have subtitles (NC ROAS, NC CPA, CPM, CTR, ROAS by Channel) prove this works — extend to the others.

**Metrics that need v1 descriptions:** MER, ROAS, NC ROAS, CPA, NC CPA, Blended Spend, Blended CPA, Contribution Margin, Attributed Sales, CPM, CTR, AOV, New Purchases, NC Sales, Repeat Orders, MRR.

This is mostly content work, not code work.

---

## Workstream 6 — Dashboard discoverability (Justin's gap)

**Why:** Justin's exact problem. A new user finds one dashboard and assumes that's the whole product. The breadcrumb dashboard switcher (`BreadcrumbItem.svelte:62-142`) exists but is non-obvious to non-technical users.

**Plan:**

- Add a left rail of dashboard tiles on the org Home page (the page rendered when a user clicks "DASHBOARDS" in the top nav with no specific dashboard selected). Each tile: name, description, sparkline preview, last-viewed timestamp.
- Inline `data-tour` anchors so the Section 4 tour can teach users to switch dashboards.
- This page becomes the natural landing point after the `/onboard/*` wizard completes.

---

## Workstream 7 — Admin / super-admin chrome cleanup

**Why:** Super admins (internal team, agency operators, technical users at brands) currently land in the Rill IDE chrome — a file tree of `dim_*` / `fct_*` YAMLs, a code editor, a "Canvas configurations" right panel, and (most concerning) sensitive files like `.env` and `claude_system_prompt.txt` rendered as clickable entries in the sidebar. For this population the "steep learning curve" complaint is the same as the brand-operator one: too much chrome, too much engineering vocabulary, no orientation.

### 7a — Hide sensitive / internal files from the file tree *(security + UX)*

`.env` is a credential file. Leaking it during a screenshare is an incident, not a UX issue. Add an explicit denylist:

- `.env`, `.env.*`
- `claude_system_prompt.txt`
- `.gitignore`
- Anything under `.bratrax/` if introduced

For files admins legitimately edit (`rill.yaml` / future `bratrax.yaml`), surface them through a dedicated **Project settings** panel rather than as raw files in the tree. Identify the file-tree component during implementation; `web-common/src/layout/navigation/Navigation.svelte` is the main navigation entry.

### 7b — Hide `SUPERADMINS` tab from non-admins

Screenshot 3 shows a non-admin still seeing the `SUPERADMINS` tab in the top nav. Gate the tab on the existing `organizationPermissions` already plumbed through (`web-admin/src/routes/[organization]/+layout.svelte:8`, pattern at `web-admin/src/features/organizations/OrganizationTabs.svelte:13-28`).

### 7c — Display names instead of file names in the sidebar

The current sidebar shows `dim_subscriptions_metrics.yaml`, `fct_line_items_metrics.yaml`, etc. — dbt-style file names that mean nothing to a new admin. Read the `Display name:` field from each YAML (already supported per `parse_metrics_view.go`) and render that. Keep file names in URLs for stability. Add a hover tooltip showing the file name for power users.

### 7d — Sidebar search + type filter

With this many files (8+ metrics views, 8+ models, sources, connectors), scrolling is the dominant interaction. Add a search input at the top of the tree and quick filters: Dashboards / Metrics views / Models / Sources / Connectors.

### 7e — "View as" toggle for super admins

A persistent toggle at the top of the chrome: `View as: Admin ▾` with options `Admin`, `Brand operator`, `Viewer`. Non-admin modes hide the left file tree, the right Canvas configurations panel, the `</>` icon, the `+` add buttons, and the editor controls — matching the clean chrome the brand operator sees in screenshot 3. This lets super admins both manage the project AND see exactly what their customer experiences without logging in as a separate user.

### 7f — `rill.yaml` → `bratrax.yaml`

Currently exposes upstream identity to anyone in the admin view. Option A: rename the file (requires runtime/parser support for `bratrax.yaml` as an alias). Option B: render it in the UI as `bratrax.yaml` while keeping the underlying file name. Ship Option B first; pursue Option A as follow-up.

### Rollout note

7a and 7b are quick wins — ship in the same week as Workstream 1. 7c-7f are larger and ship behind a feature flag after the brand-operator workstreams stabilize.

---

## Workstream 8 — Server-side per-user preferences (infrastructure)

**Why:** Both the checklist (Workstream 2) and the tour (Workstream 3) need durable per-user dismissal/completion state. localStorage is insufficient — users on multiple devices need dismissals to survive.

**Plan:**

- Define a simple schema in the Bratrax backend: `user_preferences(user_id, key, value_json, updated_at)`.
- Single endpoint pair: `GET /v1/user/preferences/:key`, `PUT /v1/user/preferences/:key`.
- Frontend client at `web-local/src/lib/bratrax/onboarding/user-preferences.ts`.
- Keys used in v1: `onboarding_checklist_state`, `onboarding_tour_dismissed:<tour_id>`.

---

## Workstream 9 — Telemetry funnel

**Why:** Cannot measure whether anything above moves the needle without it. **Ship this before Workstream 2** — no checklist without instrumentation.

**Extend** `web-common/src/metrics/service/BehaviourEventTypes.ts:6-34` with:

- `onboarding_checklist_shown`
- `onboarding_item_completed` (payload: `item_id`, `section`, `auto: bool`)
- `onboarding_section_completed` (payload: `section`)
- `onboarding_checklist_dismissed`
- `onboarding_tour_started` (payload: `tour_id`)
- `onboarding_tour_step_completed` (payload: `tour_id`, `step`)
- `onboarding_tour_dismissed` (payload: `tour_id`, `step_at_dismiss`)
- `metric_tooltip_opened` (payload: `metric_name`)
- `empty_state_cta_clicked` (payload: `state_id`, `checklist_item_id`)
- `first_dashboard_view` (payload: `dashboard_name`)

Reuse the existing `BehaviourEventHandler` pattern (`web-common/src/metrics/BehaviourEventHandler.ts:27-99`). Add `fireOnboardingEvent(action, payload)`.

The primary success metric for this entire plan: percentage of newly-signed-up users who reach `onboarding_section_completed[section=4]` within 7 days.

---

## Voice and tone for all onboarding content

Brat's response to Alessio set the template:

> "We want to be honest: if it wasn't easy, we failed you. Bratrax Lite was built specifically for small teams without developers, and a steep learning curve is exactly what we were trying to eliminate. So if that was your experience, something went wrong on our end."

Humble, user-blameless, engineer-not-salesperson. Applies to:

- The written getting-started guide.
- Empty-state CTA copy.
- Tooltip text (including metric definitions).
- Checklist copy and section headers.
- The first-visit tour body text.

Anti-patterns to avoid:

- "Easy!" / "Simple!" / "In just X clicks!" — patronizing.
- Imperative shouting ("CONNECT NOW") — sales voice.
- Apologetic over-explaining — undermines trust.
- Faux-friendly mascot speak ("Oops! It looks like…").

Better:

- "We sync once every 1-3 hours, so today's data won't appear until tomorrow morning."
- "MER (Marketing Efficiency Ratio). Total revenue ÷ total ad spend across all channels. Above 3 typically means your blended marketing is profitable."
- "Finish setting up tracking templates so the dashboard can show channel-level attribution."

---

## Suggested rollout sequence

1. **Week 1 — Workstream 1 (written getting-started guide).** Independent of code. Highest impact per hour of work; Alessio asked for it directly. Ship as soon as the copy is ready.
2. **Week 1-2 — Workstream 9 (telemetry) + Workstream 8 (server-side preferences) + Workstream 7a-7b (hide sensitive files, hide SUPERADMINS tab from non-admins).** Foundational + the quick-win admin chrome fixes. Telemetry is a hard prerequisite for Workstream 2.
3. **Week 2 — Workstream 5 (metric literacy).** Mostly content, low code risk. Quick win that addresses "I don't know what I'm looking at" before the checklist ships.
4. **Week 3-4 — Workstream 2 (checklist).** Ship hybrid v1 (manual fallback for backend-dependent items), harden auto-detection in v1.1 once the backend endpoint lands.
5. **Week 4 — Workstream 4 (empty-state linkage).** Light touch; goes in after the checklist exists.
6. **Week 4-5 — Workstream 3 (tour infrastructure) + Section 4 tours.** Largest engineering lift; builds on prior work.
7. **Week 5 — Workstream 6 (dashboard discoverability).** Polish.
8. **Week 5-6 — Workstream 7c-7f (admin sidebar display names, search, view-as toggle, `bratrax.yaml`).** Larger structural changes; ship behind a feature flag.

Two rules across the rollout:

- **Telemetry first.** Don't ship the checklist without instrumentation; you can't measure success otherwise.
- **Backend dependencies fall back to manual.** Every checklist item, every empty-state link, must work in a degraded mode where the backend status endpoint is missing.

---

## Open questions to resolve before / during implementation

1. **Post-signup landing.** Trace the full flow for a brand-new user vs. a returning user with partial setup. `routes/+layout.ts:38` redirects authenticated `/` traffic to `/developer`; the `/onboard/*` routes have resume logic. Where exactly should the checklist first appear?
2. **Checklist vs. existing wizard.** Confirm recommendation: wizard handles initial setup (Sections 1-3); checklist surfaces Sections 4-5 plus any incomplete earlier items on every authenticated page thereafter.
3. **Email onboarding.** Is a welcome series, day-3 nudge, or similar already running? If yes, it should mirror the checklist structure so the in-app and email journeys reinforce each other.
4. **User-preferences storage.** Does the Bratrax backend already have a user-settings table we can extend, or is `user_preferences` a new table?
5. **Anthropic key flow.** Confirm BYOK is the only model — that every customer must supply their own key. If there's a managed-key option for some tier, item 10 may need to be conditional.

---

## Verification

**Workstream 1 (guide):** A first-time user can read it in under 15 minutes and answer "how do I connect Meta?", "how do I add COGS?", "what does NC ROAS mean?", "what does the attribution model do?".

**Workstream 2 (checklist):**
- Fresh signup → checklist appears in bottom-right.
- Section 4/5 items show as locked or "complete Section 1-3 first" until prerequisites are met.
- Connecting Shopify in the backend auto-flips item 1 within 30s (hybrid v1; manual v1 has user clicking "Mark complete").
- Click "Mark complete" on item 7. Reload. Sign in from a second browser. Verify still complete (server-side persistence).
- Open a dashboard. Switch attribution model. Verify item 16 auto-completes.
- Dismiss the checklist. Sign out and in. Verify still dismissed.

**Workstream 3 (tours):** First-ever view of Campaign Deep Dive shows the 3-step tour. Dismiss. Verify it does not re-appear, and that signing in on a second device respects the dismissal.

**Workstream 4 (empty-state linkage):** Force one of the targeted empty states. Verify the checklist-linkage CTA appears and clicking it opens the checklist at the relevant item.

**Workstream 5 (metric literacy):** Hover the info icon on every KPI tile on every default dashboard. Verify a tooltip appears with a non-empty description.

**Workstream 6 (discoverability):** A new user on the org Home page sees a left rail of dashboard tiles. Each tile is clickable and routes to the dashboard. Sparkline previews render.

**Workstream 7 (admin chrome):** Sign in as a super admin. Verify `.env` and `claude_system_prompt.txt` no longer appear in the file tree. Verify sidebar shows display names; hover shows file names. Verify search filters work. Toggle "View as: Brand operator" — verify the chrome matches what a brand operator sees. Sign in as a brand operator; verify the `SUPERADMINS` tab is not visible.

**Workstream 9 (telemetry):** Build an activation funnel in the analytics warehouse: `bratrax_first_login` → `onboarding_section_completed[section=1]` → `onboarding_section_completed[section=4]` → 7-day retention. This is the primary metric for whether this entire effort worked.

**Tests:**

- `npm run test -w web-common` after changes to shared components.
- `npm run test -w web-local` Playwright e2e for the checklist appearance, dismissal, and an auto-complete flow (mock the backend endpoint).
- `npm run quality` for lint/format before each PR.

---

## End of plan

This document is intended as a developer hand-off. Each workstream should become an issue. The "out of scope" section at the top should travel with the issues so reviewers can recognize that adjacent UX/bug/feature work is being handled separately and is not blocking onboarding.
