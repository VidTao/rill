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
  import { getBreadcrumbOptions } from "@rilldata/web-common/features/dashboards/dashboard-utils";
  import {
    useValidCanvases,
    useValidExplores,
  } from "@rilldata/web-common/features/dashboards/selectors.js";
  import DeployProjectCTA from "@rilldata/web-common/features/dashboards/workspace/DeployProjectCTA.svelte";
  import ExplorePreviewCTAs from "@rilldata/web-common/features/explores/ExplorePreviewCTAs.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags.ts";
  import { useProjectTitle } from "@rilldata/web-common/features/project/selectors";
  import { isDeployPage } from "@rilldata/web-common/layout/navigation/route-utils";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { get } from "svelte/store";
  import { parseDocument } from "yaml";
  import InputWithConfirm from "../components/forms/InputWithConfirm.svelte";
  import Tag from "../components/tag/Tag.svelte";
  import { fileArtifacts } from "../features/entity-management/file-artifacts";

  const { deploy, developerChat, stickyDashboardState } = featureFlags;

  export let mode: string;
  export let externalUser: { email: string; name: string; role?: string } | null = null;
  export let onLogout: (() => void) | null = null;

  // If externalUser has a role, show a friendly role label instead of the dev/preview mode.
  $: roleLabel =
    externalUser?.role === "admin"
      ? "Administrator"
      : externalUser?.role === "viewer"
        ? "Viewer"
        : null;

  $: ({ instanceId } = $runtime);

  $: ({
    params: { name: dashboardName },
    route,
  } = $page);

  $: ({ unsavedFiles } = fileArtifacts);
  $: ({ size: unsavedFileCount } = $unsavedFiles);
  $: onDeployPage = isDeployPage($page);
  $: showDeployCTA = $deploy && !onDeployPage;
  $: showDeveloperChat = $developerChat && !onDeployPage;

  $: exploresQuery = useValidExplores(instanceId);
  $: canvasQuery = useValidCanvases(instanceId);
  $: projectTitleQuery = useProjectTitle(instanceId);

  $: projectTitle = $projectTitleQuery?.data ?? "Untitled Rill Project";

  $: explores = $exploresQuery?.data ?? [];
  $: canvases = $canvasQuery?.data ?? [];

  $: defaultDashboard = explores[0] ?? canvases[0] ?? null;

  $: hasValidDashboard = Boolean(defaultDashboard);

  $: dashboardOptions = {
    options: getBreadcrumbOptions(explores, canvases),
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

  async function submitTitleChange(editedTitle: string) {
    const artifact = fileArtifacts.getFileArtifact("/rill.yaml");

    let content = get(artifact.editorContent);

    if (!content) {
      await artifact.fetchContent();
      content = get(artifact.remoteContent);
      if (!content) {
        return;
      }
    }
    const parsed = parseDocument(content);

    parsed.set("display_name", editedTitle);

    artifact.updateEditorContent(parsed.toString(), true);
    await artifact.saveLocalContent();
  }
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
      {#if $exploresQuery?.data}
        <Breadcrumbs {pathParts} {currentPath} />
      {/if}
    {:else if mode === "Developer"}
      <InputWithConfirm
        size="md"
        bumpDown
        type="Project"
        textClass="font-medium"
        value={projectTitle}
        onConfirm={submitTitleChange}
        showIndicator={unsavedFileCount > 0}
      />
    {/if}
  {/if}

  <div class="ml-auto flex gap-x-2 h-full w-fit items-center py-2">
    {#if mode === "Preview"}
      {#if route.id?.includes("explore")}
        <ExplorePreviewCTAs exploreName={dashboardName} />
      {:else if route.id?.includes("canvas")}
        <CanvasPreviewCTAs canvasName={dashboardName} />
      {/if}
    {:else if showDeveloperChat}
      <ChatToggle />
    {/if}
    {#if showDeployCTA}
      <DeployProjectCTA {hasValidDashboard} />
    {/if}
    <LocalAvatarButton {externalUser} {onLogout} />
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
</style>
