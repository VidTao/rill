<script lang="ts">
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import { get } from "svelte/store";
  import { page } from "$app/stores";

  $: instanceId = get(runtime).instanceId;

  // Fetch Canvas resources (dashboards)
  $: canvasQuery = createRuntimeServiceListResources(instanceId, {
    kind: "rill.runtime.v1.Canvas",
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
</nav>

<style lang="postcss">
  .viewer-nav {
    @apply flex flex-col h-full w-56 border-r border-bratrax-border bg-bratrax-surface py-3 px-2 overflow-y-auto flex-none;
  }

  /* "DASHBOARDS" caps label — uses the design-system .section-header pattern.
     Idle = AA-passing dark olive on cream / 70% acid on dark (theme-aware). */
  .nav-header {
    font-family: "Space Mono", monospace;
    font-size: 0.875rem; /* 14px */
    font-weight: 700;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-acid-text);
    padding: 0 8px 4px;
  }

  .nav-list {
    @apply flex flex-col gap-0.5;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  /* 3-state nav-item ladder — theme-aware via design-system tokens. */
  .nav-link {
    @apply block px-2 py-1.5 text-sm;
    color: var(--color-text);
    text-decoration: none;
    border-left: 2px solid transparent;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
  }

  .nav-link:hover {
    background-color: var(--color-acid-dim);
    color: var(--color-text);
  }

  .nav-link.active {
    background-color: var(--color-acid-mid);
    border-left: 2px solid var(--color-acid);
    color: var(--color-text);
    font-weight: 500;
  }

  :global(.dark) .nav-link {
    color: var(--color-text-secondary);
  }
  :global(.dark) .nav-link:hover {
    color: var(--color-text);
  }
  :global(.dark) .nav-link.active {
    color: var(--color-acid);
  }
</style>
