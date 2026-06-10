<script lang="ts">
  import { page } from "$app/stores";
  import Breadcrumbs from "@rilldata/web-common/components/navigation/breadcrumbs/Breadcrumbs.svelte";
  import type {
    PathOption,
    PathOptions,
  } from "@rilldata/web-common/components/navigation/breadcrumbs/types";
  import { getBreadcrumbOptions } from "@rilldata/web-common/features/dashboards/dashboard-utils";
  import { useValidCanvases } from "@rilldata/web-common/features/dashboards/selectors.js";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags.ts";
  import { useProjectTitle } from "@rilldata/web-common/features/project/selectors";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  // Renders the project → dashboard breadcrumb that ApplicationHeader used to
  // carry inline. Lives on its own row in the Bratrax shell, below the tabs
  // strip. Only mounted when we're on a canvas preview route — the parent
  // gate keeps non-preview surfaces clean.

  const { stickyDashboardState } = featureFlags;

  $: ({ instanceId } = $runtime);
  $: ({
    params: { name: dashboardName },
  } = $page);

  $: canvasQuery = useValidCanvases(instanceId);
  $: projectTitleQuery = useProjectTitle(instanceId);

  $: projectTitle = $projectTitleQuery?.data ?? "Untitled Bratrax Project";
  $: canvases = $canvasQuery?.data ?? [];

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

{#if $canvasQuery?.data}
  <div class="dashboard-breadcrumb-strip">
    <Breadcrumbs {pathParts} {currentPath} />
  </div>
{/if}

<style lang="postcss">
  .dashboard-breadcrumb-strip {
    @apply flex items-center w-full px-4 py-2 border-b;
    border-color: var(--color-border, var(--border));
    background: var(--color-surface-base, var(--background));
  }
</style>
