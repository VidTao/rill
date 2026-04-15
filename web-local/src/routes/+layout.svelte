<script lang="ts">
  import { dev } from "$app/environment";
  import { page } from "$app/stores";
  import BannerCenter from "@rilldata/web-common/components/banner/BannerCenter.svelte";
  import NotificationCenter from "@rilldata/web-common/components/notifications/NotificationCenter.svelte";
  import RepresentingUserBanner from "@rilldata/web-common/features/authentication/RepresentingUserBanner.svelte";
  import FileAndResourceWatcher from "@rilldata/web-common/features/entity-management/FileAndResourceWatcher.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags";
  import { initPylonWidget } from "@rilldata/web-common/features/help/initPylonWidget";
  import RemoteProjectManager from "@rilldata/web-common/features/project/RemoteProjectManager.svelte";
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

  export let data: LayoutData;

  const { deploy } = featureFlags;

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
</script>

<QueryClientProvider client={queryClient}>
  <FileAndResourceWatcher {host} {instanceId}>
    <div class="body h-screen w-screen overflow-hidden absolute flex flex-col">
      {#if data.initialized}
        <BannerCenter />
        <RepresentingUserBanner />
        <ApplicationHeader
          {mode}
          externalUser={$bratraxUser}
          onLogout={handleBratraxLogout}
        />
        {#if $deploy}
          <RemoteProjectManager />
        {/if}
      {/if}

      {#if $bratraxUser}
        <nav class="bratrax-nav flex gap-6 border-b border-bratrax-border px-4 py-0">
          <a
            href="/"
            class="bratrax-nav-link"
            class:active={!onConnectorsPage && !onWorkshopPage}
          >
            Developer
          </a>
          <a
            href="/connectors"
            class="bratrax-nav-link"
            class:active={onConnectorsPage}
          >
            Connectors
          </a>
          <a
            href="/workshop"
            class="bratrax-nav-link"
            class:active={onWorkshopPage}
          >
            Workshop
          </a>
        </nav>
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

  /* Bratrax navigation tabs — Space Mono, uppercase, acid green active */
  .bratrax-nav-link {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 14px;
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: #858585 !important;
    text-decoration: none;
    padding: 10px 0;
    border-bottom: 2px solid transparent;
    transition: color 0.2s, border-color 0.2s;
  }

  .bratrax-nav-link:hover {
    color: #E8E4DC !important;
  }

  .bratrax-nav-link.active {
    color: #D4FF00 !important;
    border-bottom-color: #D4FF00;
  }
</style>
