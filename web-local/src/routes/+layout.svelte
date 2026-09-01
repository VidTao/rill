<script lang="ts">
  import { dev } from "$app/environment";
  import { page } from "$app/stores";
  import BannerCenter from "@rilldata/web-common/components/banner/BannerCenter.svelte";
  import NotificationCenter from "@rilldata/web-common/components/notifications/NotificationCenter.svelte";
  import RepresentingUserBanner from "@rilldata/web-common/features/authentication/RepresentingUserBanner.svelte";
  import FileAndResourceWatcher from "@rilldata/web-common/features/entity-management/FileAndResourceWatcher.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags";
  import { initPylonWidget } from "@rilldata/web-common/features/help/initPylonWidget";
  import ApplicationHeader from "@rilldata/web-common/layout/ApplicationHeader.svelte";
  import DashboardTabsStrip from "@rilldata/web-common/layout/DashboardTabsStrip.svelte";
  import {
    dashboardPrefs,
    dashboardPrefsLoaded,
    isInternalSuperadminCanvas,
    isRillDemoCanvas,
    loadDashboardPrefs,
    mergeDashboardPrefs,
    resetDashboardPrefs,
  } from "$lib/bratrax/dashboardPrefs";
  import SettingsDropdown from "@rilldata/web-common/layout/SettingsDropdown.svelte";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import BlockingOverlayContainer from "@rilldata/web-common/layout/BlockingOverlayContainer.svelte";
  import { overlay } from "@rilldata/web-common/layout/overlay-store";
  import {
    initPosthog,
    posthogIdentify,
  } from "@rilldata/web-common/lib/analytics/posthog";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import {
    errorEventHandler,
    initMetrics,
  } from "@rilldata/web-common/metrics/initMetrics";
  import { localServiceGetMetadata } from "@rilldata/web-common/runtime-client/local-service";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import type { Query } from "@tanstack/query-core";
  import { QueryClientProvider } from "@tanstack/svelte-query";
  import type { AxiosError } from "axios";
  import { onMount } from "svelte";
  import type { LayoutData } from "./$types";
  import "@rilldata/web-common/app.css";
  import "../bratrax-theme.css";
  import {
    bratraxUser,
    bratraxOnboarded,
    bratraxOnboardResumeRoute,
    bratraxShowWelcomeCard,
    bratraxViewerMidOnboarding,
  } from "$lib/bratrax/auth-store";
  import { bratraxLogout } from "$lib/bratrax/auth";
  import {
    clearEmbeddedToken,
    isShopifyEmbedded,
  } from "$lib/bratrax/shopify-embed";
  import ClientSwitcher from "$lib/bratrax/ClientSwitcher.svelte";
  import AddStoreButton from "$lib/bratrax/AddStoreButton.svelte";
  import OrderDrilldownProvider from "$lib/bratrax/order-attribution/OrderDrilldownProvider.svelte";
  import ChecklistBanner from "$lib/bratrax/onboarding/ChecklistBanner.svelte";
  import ContinueSetupPill from "$lib/bratrax/onboarding/ContinueSetupPill.svelte";
  import ReconnectPill from "$lib/bratrax/onboarding/ReconnectPill.svelte";
  import WelcomeCard from "$lib/bratrax/onboarding/WelcomeCard.svelte";
  import MidOnboardingViewer from "$lib/bratrax/onboarding/MidOnboardingViewer.svelte";
  import SupportChatSidebar from "$lib/bratrax/support-bot/SupportChatSidebar.svelte";

  export let data: LayoutData;

  queryClient.getQueryCache().config.onError = (
    error: AxiosError,
    query: Query,
  ) => errorEventHandler?.requestErrorEventHandler(error, query);
  initPylonWidget();

  let removeJavascriptListeners: () => void;
  onMount(async () => {
    const config = await localServiceGetMetadata();

    const shouldSendAnalytics =
      config.analyticsEnabled && !import.meta.env.VITE_PLAYWRIGHT_TEST && !dev;

    if (shouldSendAnalytics) {
      await initMetrics(config); // Proxies events through the Rill "intake" service
      initPosthog(config.version);
      posthogIdentify(config.userId, {
        installId: config.installId,
      });

      removeJavascriptListeners =
        errorEventHandler.addJavascriptErrorListeners();
    }

    featureFlags.set(false, "adminServer");
    featureFlags.set(config.readonly, "readOnly");
  });

  /**
   * Async mount doesnt support an unsubscribe method.
   * So we need this to make sure javascript listeners for error handler is removed.
   */
  onMount(() => {
    return () => removeJavascriptListeners?.();
  });

  // Phase 3: hydration of the dashboard-prefs store happens in the
  // identity-aware reactive block below — when canvasUserKey first goes
  // from "anon" to "<userId>:<clientId>", we reset + load fresh.

  // Imported here rather than in auth.ts so the logout path owns clearing
  // both credentials in one place.
  async function handleBratraxLogout() {
    await bratraxLogout();
    // bratraxLogout only expires the bratrax_auth cookie. Inside the Shopify
    // iframe that cookie was never usable — the session is held by the bearer
    // token in sessionStorage — so without this the merchant stays signed in,
    // /login sees a valid session and bounces them straight back to the page
    // they just tried to leave.
    clearEmbeddedToken();
    bratraxUser.set(null);
    window.location.href = "/login";
  }

  $: ({ host, instanceId } = $runtime);

  $: ({ route } = $page);

  $: mode = route.id?.includes("(viz)") ? "Preview" : "Dashboards";
  $: onCanvasPreview = !!route.id?.includes("(viz)/canvas");
  $: onCustomizePage = $page.url.pathname.startsWith("/customize");
  // Ribbon 2 (dashboard tabs strip) mounts on canvas previews and on the
  // /customize page, so the tabs persist while editing visibility/order.
  $: showDashboardTabsStrip = onCanvasPreview || onCustomizePage;
  // Strip-ready gate: TanStack reports `isPending: true` until the first
  // fetch lands (and we nuke the cache on user-change, so there's no stale
  // data to short-circuit this). Once the query has settled AND the prefs
  // store has hydrated, render the strip with the right tabs in the right
  // order. The same-height placeholder fills the space until then.
  // Use explicit `=== false` (not !isPending) so when $canvasListQuery is
  // still undefined during the initial reactive cascade we read FALSE rather
  // than blowing up on `undefined.isPending`. Same logical outcome — strip
  // stays hidden until the query resolves — but null-safe.
  $: dashboardTabsReady =
    $canvasListQuery?.isPending === false && $dashboardPrefsLoaded;

  // Phase 3: merge runtime canvases × per-user prefs into the final tab list
  // that DashboardTabsStrip renders. Hidden canvases are filtered out; new
  // ones (not yet in prefs) get appended at the end via mergeDashboardPrefs.
  // Demo canvases (Rill upstream bundled examples) are dropped at the source
  // so they can never leak into the tab strip during runtime warm-up — see
  // RILL_DEMO_CANVAS_NAMES in dashboardPrefs.ts.
  $: realCanvases = ($canvasListQuery?.data?.resources ?? []).filter((r) => {
    const name = r.meta?.name?.name;
    return (
      !isRillDemoCanvas(name) && (isSuper || !isInternalSuperadminCanvas(name))
    );
  });
  $: dashboardTabs = mergeDashboardPrefs(realCanvases, $dashboardPrefs)
    .filter((m) => m.visible)
    .map((m) => ({
      key: m.key,
      label: m.resource?.canvas?.spec?.displayName || m.key,
    }));

  $: onConnectorsPage = $page.url.pathname.startsWith("/connectors");
  $: onCostSettingsPage = $page.url.pathname.startsWith("/cost-settings");
  $: onSettingsPage = $page.url.pathname.startsWith("/settings");
  $: onSuperadminsPage = $page.url.pathname.startsWith("/superadmins");
  $: onClientsPage = $page.url.pathname.startsWith("/clients");
  $: onHelpPage = $page.url.pathname.startsWith("/help");
  $: onMetricTreesPage = $page.url.pathname.startsWith("/metric-trees");
  // Used to hide the "Add store" button during onboarding. The nav strip
  // itself is no longer gated on this — $bratraxOnboarded (set by the
  // +layout.ts guard from the onboarding step) drives the minimal-vs-full
  // nav, so a mid-onboarding user still gets a Settings + Help strip.
  $: onOnboardPage = $page.url.pathname.startsWith("/onboard");

  // /embed/* renders inside the Shopify admin iframe, which supplies its own
  // navigation. Our header, dashboard tabs, client switcher, checklist banner
  // and support panel would be a second, competing chrome inside a container
  // that is already only ~1150px wide on desktop and ~400px on mobile admin.
  // The embedded pages carry their own "Open in Bratrax" affordance instead.
  $: onEmbedPage = $page.url.pathname.startsWith("/embed");

  // Running inside the Shopify admin iframe. Distinct from onEmbedPage, which
  // is only the single-dashboard route: this covers the onboarding screens too,
  // which render with full chrome but are still inside Shopify's frame. Sticky
  // for the session, so it's safe to read once rather than reactively.
  const shopifyEmbedded = isShopifyEmbedded();

  // Role-based nav visibility. The DB-side enum is super_admin / admin / viewer.
  $: role = $bratraxUser?.role ?? null;
  $: isViewer = role === "viewer";
  $: isSuper = role === "super_admin";
  $: isAdminOrSuper = role === "admin" || isSuper;
  // Multi-store users (rill_users.multi_client_id != NULL) get the same
  // client switcher dropdown super_admins use — the backend scopes the
  // returned list to their sibling sub-stores.
  $: isMultiStore = !!$bratraxUser?.multi_client_id;

  // First-canvas lookup powers the DASHBOARDS tab href so clicks go straight
  // to /canvas/<name>, skipping the /developer detour. Falls back to /developer
  // when the list is empty (it renders the "No dashboards yet" placeholder).
  // The queryClient must be passed explicitly: this call sits in the root
  // layout's script, above the QueryClientProvider in the component tree, so
  // the context-based lookup TanStack uses by default would not find one.
  //
  // In multi-tenant Rill the runtime instanceId is the literal string
  // "default" (the Go proxy maps it to the user's real instance via the
  // auth cookie). The default query key only sees "default", so a cached
  // response from a previous identity OR Rill's bundled demo projects
  // (e.g. margin_scorecard) could leak into a fresh-logged-in admin's view.
  // Fix: include the auth user identity in the query key. Different user
  // → different cache slot → no leak, no reactive cleanup needed.
  $: canvasUserKey = $bratraxUser
    ? `${$bratraxUser.id}:${$bratraxUser.client_id ?? ""}`
    : "anon";

  // Reset + refetch dashboard prefs whenever the auth identity changes.
  // The prefs store is module-scoped (persists across SPA navigations and
  // in-session logins), so without this the previous user's order leaks
  // into the new session and the strip renders the wrong tabs until F5.
  let lastSeenPrefsUserKey: string | null = null;
  $: if (canvasUserKey !== lastSeenPrefsUserKey) {
    lastSeenPrefsUserKey = canvasUserKey;
    resetDashboardPrefs();
    if ($bratraxUser) void loadDashboardPrefs();
  }
  $: canvasListQuery = createRuntimeServiceListResources(
    instanceId,
    { kind: "rill.runtime.v1.Canvas" },
    {
      query: {
        queryKey: ["bratrax-canvas-list", canvasUserKey, instanceId, "Canvas"],
        refetchOnMount: "always",
        staleTime: 0,
        // Warm-up poll: keep refetching every 2s while the runtime returns
        // ZERO real canvases (only Rill demos, or an empty list). The custom
        // queryKey above means FileAndResourceWatcher's SSE invalidation
        // doesn't reach this query — without a poll we'd see an empty strip
        // post-onboarding until the user hits F5. Stops the moment at least
        // one real canvas lands. Genuinely-empty workspaces poll forever at
        // 2s (tiny payload; the strip just stays empty until the user gets
        // canvases compiled — acceptable v1 trade-off).
        refetchInterval: (query) => {
          const resources = (query?.state?.data?.resources ?? []) as {
            meta?: { name?: { name?: string } };
          }[];
          const realCount = resources.filter((r) => {
            const name = r.meta?.name?.name;
            return (
              !isRillDemoCanvas(name) &&
              (isSuper || !isInternalSuperadminCanvas(name))
            );
          }).length;
          return realCount > 0 ? false : 2000;
        },
      },
    },
    queryClient,
  );
  $: firstCanvasName = (() => {
    const resources = $canvasListQuery?.data?.resources ?? [];
    for (const r of resources) {
      const name = r.meta?.name?.name;
      // Skip Rill upstream demo dashboards so the DASHBOARDS link never
      // points at margin_scorecard / auction_explore / etc. during warm-up.
      if (
        name &&
        !isRillDemoCanvas(name) &&
        (isSuper || !isInternalSuperadminCanvas(name))
      )
        return name;
    }
    return null;
  })();
  $: dashboardsHref = firstCanvasName
    ? `/canvas/${firstCanvasName}`
    : "/developer";
</script>

<QueryClientProvider client={queryClient}>
  <FileAndResourceWatcher {host} {instanceId}>
    <OrderDrilldownProvider>
      <div
        class="body h-screen w-screen overflow-hidden absolute flex flex-col"
      >
        {#if $bratraxUser && !onEmbedPage}
          <BannerCenter />
          <RepresentingUserBanner />
          <ApplicationHeader
            {mode}
            externalUser={$bratraxUser}
            onLogout={handleBratraxLogout}
            hideAssistants={shopifyEmbedded}
            onboardingLink={isViewer
              ? { href: "/onboarding", label: "Getting started" }
              : null}
          >
            <svelte:fragment slot="header-extras">
              {#if isSuper || isMultiStore}
                <ClientSwitcher />
              {/if}
              {#if isMultiStore && !isViewer && !onOnboardPage}
                <AddStoreButton />
              {/if}
            </svelte:fragment>

            <svelte:fragment slot="continue-setup">
              <!-- Both pills self-gate on role + state; safe to mount
                   unconditionally — they render nothing when they shouldn't. -->
              <!-- Dead connector token. Deliberately first: a broken pipeline
                   outranks unfinished setup, and in practice the two rarely
                   co-occur (this needs a connector that was once working). -->
              <ReconnectPill />
              <!-- Renders nothing for super-admin/viewer or when the checklist
                   is dismissed/empty. -->
              <ContinueSetupPill />
            </svelte:fragment>

            <svelte:fragment slot="nav-tabs">
              <!-- Nav restructure (NAV_RESTRUCTURE_HANDOFF.MD Phase 1):
                   Dashboards-as-tabs moved to Ribbon 2 below; Ribbon 1 carries
                   only the SETTINGS ▾ dropdown + inline standalone items. -->
              {#if isViewer}
                <!-- Viewers see a slim Ribbon 1: DASHBOARDS + HELP. Account +
                     theme are reachable via the avatar dropdown; "Getting
                     started" lives there too via the onboardingLink prop. No
                     SETTINGS menu. DASHBOARDS hides when the viewer has no
                     real canvases yet (no broken /developer fallback). -->
                {#if firstCanvasName}
                  <a
                    href={dashboardsHref}
                    class="bratrax-nav-link"
                    class:active={onCanvasPreview}
                  >
                    Dashboards
                  </a>
                {/if}
                <a
                  href="/help"
                  class="bratrax-nav-link"
                  class:active={onHelpPage}
                >
                  Help
                </a>
              {:else if !$bratraxOnboarded && !isSuper}
                <!-- Mid-onboarding minimum nav. Settings is the full dropdown
                     (matches the post-onboarding nav visually); its non-Settings
                     items bounce back to the onboarding funnel via the
                     +layout.ts isAlwaysAllowed gate — expected behavior, since
                     /onboarding is the resume link the user came from anyway. -->
                <a
                  href={$bratraxOnboardResumeRoute ?? "/developer"}
                  class="bratrax-nav-link"
                  class:active={onOnboardPage}
                >
                  Onboarding
                </a>
                <!-- Hidden inside the Shopify iframe — see the note on the
                     post-onboarding nav below. -->
                {#if !shopifyEmbedded}
                  <SettingsDropdown />
                {/if}
                <a
                  href="/help"
                  class="bratrax-nav-link"
                  class:active={onHelpPage}
                >
                  Help
                </a>
              {:else}
                <!-- Admin + super-admin Ribbon 1 nav (post-onboarding).
                     DASHBOARDS is the leftmost item of the right cluster,
                     sitting just before SETTINGS ▾. -->
                <a
                  href={dashboardsHref}
                  class="bratrax-nav-link"
                  class:active={onCanvasPreview}
                >
                  Dashboards
                </a>
                <a
                  href="/metric-trees"
                  class="bratrax-nav-link"
                  class:active={onMetricTreesPage}
                >
                  Metric Trees
                </a>
                <!-- Settings is hidden inside the Shopify iframe: its
                     destinations are full-app pages that don't belong in
                     a ~1150px frame beside Shopify's own navigation. -->
                {#if isAdminOrSuper && !shopifyEmbedded}
                  <SettingsDropdown />
                {/if}
                {#if isSuper}
                  <a
                    href="/superadmins"
                    class="bratrax-nav-link"
                    class:active={onSuperadminsPage}
                  >
                    Superadmins
                  </a>
                  <a
                    href="/clients"
                    class="bratrax-nav-link"
                    class:active={onClientsPage}
                  >
                    Clients
                  </a>
                {/if}
                <a
                  href="/help"
                  class="bratrax-nav-link"
                  class:active={onHelpPage}
                >
                  Help
                </a>
              {/if}
            </svelte:fragment>
          </ApplicationHeader>

          {#if showDashboardTabsStrip}
            {#if dashboardTabsReady}
              <DashboardTabsStrip tabs={dashboardTabs} {isAdminOrSuper} />
            {:else}
              <!-- Same-height placeholder while the canvas list settles —
                   prevents a partial-list flash on first login and keeps
                   the layout from shifting when the real tabs land. -->
              <div class="dashboard-tabs-placeholder"></div>
            {/if}
          {/if}

          <ChecklistBanner />
        {/if}

        <!-- Page content and the support-bot panel share a flex row so opening
             the panel narrows the page rather than covering it. The panel is
             mounted here once instead of inside every page layout; it collapses
             to `display: none` when closed, leaving this row a no-op.

             On /embed/* it renders as an overlay instead (see `overlay` below).
             It used to be suppressed there for two reasons, both now handled:
             EmbedHeader renders its own "?" toggle, so the panel can be closed
             again, and overlaying costs zero layout width, so a 420px column no
             longer starves the canvas in Shopify's ~1150px column.

             The mount condition takes `onEmbedPage` as an alternative to
             $bratraxUser rather than an extra restriction: inside the iframe the
             merchant is authenticated by a Shopify session token, and
             bratraxGetMe only ever attaches the sessionStorage JWT, so
             $bratraxUser can be null on a cold embedded load even though the
             support-bot's own fetch authenticates fine. Gating on it there would
             leave the "?" inert on first paint. -->
        <!-- `relative` is the anchor for the overlay variant of the panel; it is
             inert for the in-flow one. -->
        <div class="relative flex min-h-0 flex-1 flex-row">
          <div class="flex min-h-0 min-w-0 flex-1 flex-col">
            {#if $bratraxViewerMidOnboarding && !onHelpPage}
              <MidOnboardingViewer />
            {:else}
              <slot />
            {/if}
          </div>

          {#if $bratraxUser || onEmbedPage}
            <SupportChatSidebar overlay={onEmbedPage} />
          {/if}
        </div>
      </div>
    </OrderDrilldownProvider>
  </FileAndResourceWatcher>
</QueryClientProvider>

{#if $overlay !== null}
  <BlockingOverlayContainer
    bg="linear-gradient(to right, rgba(0,0,0,.6), rgba(0,0,0,.8))"
  >
    <div slot="title" class="font-bold">
      {$overlay?.title}
    </div>
    <svelte:fragment slot="detail">
      {#if $overlay?.detail}
        <svelte:component
          this={$overlay.detail.component}
          {...$overlay.detail.props}
        />
      {/if}
    </svelte:fragment>
  </BlockingOverlayContainer>
{/if}

<NotificationCenter />

{#if $bratraxShowWelcomeCard}
  <WelcomeCard />
{/if}

<style>
  /* Prevent trackpad navigation (like other code editors, like vscode.dev). */
  :global(body) {
    overscroll-behavior: none;
  }

  /* Matches DashboardTabsStrip's outer chrome (border + height) so the
     pre-data placeholder reserves the same vertical space and avoids a
     layout shift when the real tabs replace it. */
  .dashboard-tabs-placeholder {
    height: 50px;
    width: 100%;
    border-bottom: 1px solid var(--color-border, var(--border));
    background: var(--color-surface-base, var(--background));
  }

  /* Bratrax navigation tabs — Space Mono, uppercase, theme-aware via tokens.
     The 3-state ladder (idle / hover / active) uses --color-* tokens that
     auto-flip with the .dark class on <html>. Active state shows the acid border. */
  .bratrax-nav-link {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted) !important;
    text-decoration: none;
    padding: 10px 12px;
    border-bottom: 2px solid transparent;
    transition:
      color 0.2s,
      border-color 0.2s,
      background-color 0.2s;
  }

  .bratrax-nav-link:hover {
    color: var(--color-text-secondary) !important;
  }

  /* Active on the utility bar signals with brightness only — bright text + a
     neutral underline, NO acid. The acid "you are here" marker is reserved for
     the section tabs (Tier 2), so the screen carries exactly one acid signal. */
  .bratrax-nav-link.active {
    color: var(--bratrax-text-headline, var(--color-text)) !important;
    border-bottom-color: var(--color-border-strong);
  }
</style>
