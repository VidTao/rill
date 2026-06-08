<script lang="ts">
  import { page } from "$app/stores";
  import Breadcrumbs from "@rilldata/web-common/components/navigation/breadcrumbs/Breadcrumbs.svelte";
  import type {
    PathOption,
    PathOptions,
  } from "@rilldata/web-common/components/navigation/breadcrumbs/types";
  import LocalAvatarButton from "@rilldata/web-common/features/authentication/LocalAvatarButton.svelte";
  import CanvasPreviewCTAs from "@rilldata/web-common/features/canvas/CanvasPreviewCTAs.svelte";
  import ChatToggle from "@rilldata/web-common/features/chat/layouts/sidebar/ChatToggle.svelte";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import { getBreadcrumbOptions } from "@rilldata/web-common/features/dashboards/dashboard-utils";
  import { useValidCanvases } from "@rilldata/web-common/features/dashboards/selectors.js";
  import ExplorePreviewCTAs from "@rilldata/web-common/features/explores/ExplorePreviewCTAs.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags.ts";
  import { useProjectTitle } from "@rilldata/web-common/features/project/selectors";
  import { isDeployPage } from "@rilldata/web-common/layout/navigation/route-utils";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import Tag from "../components/tag/Tag.svelte";

  const { developerChat, stickyDashboardState } = featureFlags;

  export let mode: string;
  export let externalUser: { email: string; name: string; role?: string } | null = null;
  export let onLogout: (() => void) | null = null;
  export let onboardingLink: { href: string; label: string } | null = null;

  // If externalUser has a role, show a friendly role label instead of the dev/preview mode.
  $: roleLabel =
    externalUser?.role === "super_admin"
      ? "Super Admin"
      : externalUser?.role === "admin"
        ? "Administrator"
        : externalUser?.role === "viewer"
          ? "Viewer"
          : null;

  $: ({ instanceId } = $runtime);

  $: ({
    params: { name: dashboardName },
    route,
  } = $page);

  // Bratrax theme toggle — sun/moon button left of the AI/ChatToggle.
  // themeControl.preference is the persisted preference store (light/dark/system).
  $: themePreference = themeControl.preference;
  $: onDeployPage = isDeployPage($page);
  $: showDeveloperChat = $developerChat && !onDeployPage;

  $: canvasQuery = useValidCanvases(instanceId);
  $: projectTitleQuery = useProjectTitle(instanceId);

  $: projectTitle = $projectTitleQuery?.data ?? "Untitled Bratrax Project";

  $: canvases = $canvasQuery?.data ?? [];

  // Bratrax: "Dashboards" means Canvas dashboards only. Auto-generated Explore
  // dashboards are excluded from the Preview breadcrumb dropdown.
  $: dashboardOptions = {
    options: getBreadcrumbOptions([], canvases),
    carryOverSearchParams: $stickyDashboardState,
  } satisfies PathOptions;

  $: projectPath = <PathOption>{
    label: projectTitle,
    section: "project",
    depth: -1,
    href: "/",
  };

  $: pathParts = [
    { options: new Map([[projectTitle.toLowerCase(), projectPath]]) },
    dashboardOptions,
  ];

  $: currentPath = [projectTitle, dashboardName?.toLowerCase()];
</script>

<header class:border-b={!onDeployPage} class="bg-surface-base">
  {#if !onDeployPage}
    <a href="/" class="bratrax-logo-link">
      <img
        src="/img/bratrax/bratrax-logo-dark.svg"
        alt="Bratrax"
        class="bratrax-logo bratrax-logo-light-mode"
      />
      <img
        src="/img/bratrax/bratrax-logo-light.svg"
        alt="Bratrax"
        class="bratrax-logo bratrax-logo-dark-mode"
      />
    </a>

    <Tag text={roleLabel ?? mode} color="gray"></Tag>

    {#if mode === "Preview"}
      {#if $canvasQuery?.data}
        <Breadcrumbs {pathParts} {currentPath} />
      {/if}
    {/if}
  {/if}

  <div class="ml-auto flex gap-x-2 h-full w-fit items-center py-2">
    {#if mode === "Preview"}
      {#if route.id?.includes("explore")}
        <ExplorePreviewCTAs exploreName={dashboardName} />
      {:else if route.id?.includes("canvas")}
        <CanvasPreviewCTAs />
      {/if}
    {:else if showDeveloperChat}
      <slot name="header-extras" />
      <button
        type="button"
        on:click={() =>
          themeControl.set[$themePreference === "dark" ? "light" : "dark"]()}
        class="bratrax-theme-toggle"
        title={$themePreference === "dark"
          ? "Switch to light theme"
          : "Switch to dark theme"}
        aria-label="Toggle theme"
      >
        {#if $themePreference === "dark"}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="4" />
            <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
        {/if}
      </button>
      <ChatToggle />
    {/if}
    <LocalAvatarButton {externalUser} {onLogout} {onboardingLink} />
  </div>
</header>

<style lang="postcss">
  header {
    @apply w-full box-border;
    @apply flex gap-x-2 items-center px-4 flex-none;
    @apply h-[3.75rem];
  }

  .bratrax-logo-link {
    @apply flex items-center;
  }

  .bratrax-logo {
    height: 1.5rem; /* 24px */
    width: auto;
    display: block;
  }

  /* Light mode (default): show the dark-text logo, hide the light-text one */
  .bratrax-logo-dark-mode {
    display: none;
  }

  /* Dark mode: show the light-text logo, hide the dark-text one */
  :global(.dark) .bratrax-logo-light-mode {
    display: none;
  }

  :global(.dark) .bratrax-logo-dark-mode {
    display: block;
  }

  .bratrax-theme-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease, border-color 120ms ease;
  }
  .bratrax-theme-toggle:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }
  .bratrax-theme-toggle:focus-visible {
    outline: 2px solid var(--color-acid, #D4FF00);
    outline-offset: 2px;
  }
</style>
