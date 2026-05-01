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
  import { bratraxUser } from "$lib/bratrax/auth-store";
  import { bratraxLogout } from "$lib/bratrax/auth";
  import ClientSwitcher from "$lib/bratrax/ClientSwitcher.svelte";

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

  async function handleBratraxLogout() {
    await bratraxLogout();
    bratraxUser.set(null);
    window.location.href = "/login";
  }

  $: ({ host, instanceId } = $runtime);

  $: ({ route } = $page);

  $: mode = route.id?.includes("(viz)") ? "Preview" : "Developer";

  $: onConnectorsPage = $page.url.pathname.startsWith("/connectors");
  $: onWorkshopPage = $page.url.pathname.startsWith("/workshop");
  $: onCostSettingsPage = $page.url.pathname.startsWith("/cost-settings");
  $: onSettingsPage = $page.url.pathname.startsWith("/settings");
  $: onSuperadminsPage = $page.url.pathname.startsWith("/superadmins");

  // Role-based nav visibility. The DB-side enum is super_admin / admin / viewer.
  $: role = $bratraxUser?.role ?? null;
  $: isViewer = role === "viewer";
  $: isSuper = role === "super_admin";
  $: isAdminOrSuper = role === "admin" || isSuper;
</script>

<QueryClientProvider client={queryClient}>
  <FileAndResourceWatcher {host} {instanceId}>
    <div class="body h-screen w-screen overflow-hidden absolute flex flex-col">
      {#if $bratraxUser}
        <BannerCenter />
        <RepresentingUserBanner />
        <ApplicationHeader
          {mode}
          externalUser={$bratraxUser}
          onLogout={handleBratraxLogout}
        >
          <svelte:fragment slot="header-extras">
            {#if isSuper}
              <ClientSwitcher />
            {/if}
          </svelte:fragment>
        </ApplicationHeader>
        {#if !isViewer}
          <nav class="bratrax-nav flex gap-6 border-b border-bratrax-border px-4 py-0">
            <a
              href="/developer"
              class="bratrax-nav-link"
              class:active={!onConnectorsPage && !onWorkshopPage && !onCostSettingsPage && !onSettingsPage && !onSuperadminsPage}
            >
              Developer
            </a>
            {#if isAdminOrSuper}
              <a
                href="/connectors"
                class="bratrax-nav-link"
                class:active={onConnectorsPage}
              >
                Connectors
              </a>
            {/if}
            {#if isSuper}
              <a
                href="/workshop"
                class="bratrax-nav-link"
                class:active={onWorkshopPage}
              >
                Workshop
              </a>
            {/if}
            {#if isAdminOrSuper}
              <a
                href="/cost-settings"
                class="bratrax-nav-link"
                class:active={onCostSettingsPage}
              >
                Cost Settings
              </a>
              <a
                href="/settings"
                class="bratrax-nav-link"
                class:active={onSettingsPage}
              >
                Settings
              </a>
            {/if}
            {#if isSuper}
              <a
                href="/superadmins"
                class="bratrax-nav-link"
                class:active={onSuperadminsPage}
              >
                Superadmins
              </a>
            {/if}
          </nav>
        {/if}
      {/if}

      <slot />
    </div>
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

<style>
  /* Prevent trackpad navigation (like other code editors, like vscode.dev). */
  :global(body) {
    overscroll-behavior: none;
  }

  /* Bratrax navigation tabs — Space Mono, uppercase, theme-aware via tokens.
     The 3-state ladder (idle / hover / active) uses --color-* tokens that
     auto-flip with the .dark class on <html>. Active state shows the acid border. */
  .bratrax-nav-link {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 0.875rem; /* 14px */
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-text-muted) !important;
    text-decoration: none;
    padding: 10px 12px;
    border-bottom: 2px solid transparent;
    transition: color 0.2s, border-color 0.2s, background-color 0.2s;
  }

  .bratrax-nav-link:hover {
    color: var(--color-text) !important;
    background: var(--color-acid-dim);
  }

  .bratrax-nav-link.active {
    color: var(--color-text) !important;
    background: var(--color-acid-mid);
    border-bottom-color: var(--color-acid);
  }

  :global(.dark) .bratrax-nav-link.active {
    color: var(--color-acid) !important;
  }
</style>
