<script lang="ts">
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import { get } from "svelte/store";
  import { page } from "$app/stores";

  $: instanceId = get(runtime).instanceId;

  // Fetch Canvas resources
  $: canvasQuery = createRuntimeServiceListResources(instanceId, {
    kind: "rill.runtime.v1.Canvas",
  });

  // Fetch Explore resources
  $: exploreQuery = createRuntimeServiceListResources(instanceId, {
    kind: "rill.runtime.v1.Explore",
  });

  $: canvases = ($canvasQuery.data?.resources ?? [])
    .map((r) => ({
      name: r.meta?.name?.name ?? "",
      displayName:
        r.canvas?.state?.validSpec?.displayName ||
        r.meta?.name?.name?.replace(/_/g, " ") ||
        "",
    }))
    .filter((c) => c.name);

  $: explores = ($exploreQuery.data?.resources ?? [])
    .map((r) => ({
      name: r.meta?.name?.name ?? "",
      displayName:
        r.explore?.state?.validSpec?.displayName ||
        r.meta?.name?.name?.replace(/_/g, " ") ||
        "",
    }))
    .filter((e) => e.name);

  $: currentPath = $page.url.pathname;
</script>

<nav class="viewer-nav">
  <div class="nav-header">Dashboards</div>

  {#if canvases.length > 0}
    <ul class="nav-list">
      {#each canvases as canvas}
        <li>
          <a
            href="/files/dashboards/{canvas.name}.yaml"
            class="nav-link"
            class:active={currentPath === `/files/dashboards/${canvas.name}.yaml`}
          >
            {canvas.displayName}
          </a>
        </li>
      {/each}
    </ul>
  {/if}

  {#if explores.length > 0}
    <div class="nav-header" style="margin-top: 12px;">Explores</div>
    <ul class="nav-list">
      {#each explores as explore}
        <li>
          <a
            href="/files/dashboards/{explore.name}.yaml"
            class="nav-link"
            class:active={currentPath === `/files/dashboards/${explore.name}.yaml`}
          >
            {explore.displayName}
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</nav>

<style lang="postcss">
  .viewer-nav {
    @apply flex flex-col h-full w-56 border-r border-bratrax-border bg-bratrax-surface py-3 px-2 overflow-y-auto flex-none;
  }

  .nav-header {
    font-family: "Space Mono", monospace;
    font-size: 0.875rem; /* 14px */
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: rgba(212, 255, 0, 0.7);
    padding: 0 8px 4px;
  }

  .nav-list {
    @apply flex flex-col gap-0.5;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .nav-link {
    @apply block px-2 py-1.5 text-sm;
    color: #B0B0B0;
    text-decoration: none;
    transition: all 0.15s;
  }

  .nav-link:hover {
    background-color: #1E1E1E;
    color: #E8E4DC;
  }

  .nav-link.active {
    background-color: rgba(212, 255, 0, 0.08);
    color: #D4FF00;
    border-left: 2px solid #D4FF00;
    font-weight: 500;
  }
</style>
