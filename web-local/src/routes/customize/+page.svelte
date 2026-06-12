<script lang="ts">
  import { onMount } from "svelte";
  import { get as getStore } from "svelte/store";
  import {
    dashboardPrefs,
    loadDashboardPrefs,
    setDashboardPrefs,
    mergeDashboardPrefs,
    type DashboardPref,
  } from "$lib/bratrax/dashboardPrefs";
  import { createRuntimeServiceListResources } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  // Phase 3 of the nav restructure (docs/NAV_RESTRUCTURE_HANDOFF.MD).
  // Drag-to-reorder + checkbox-to-hide; auto-saves on every change via the
  // 500ms debounce in dashboardPrefs.setDashboardPrefs. The first 6 visible
  // rows count as primary tabs in Ribbon 2; the rest fall into MORE ▾.
  // Inactive-source greying (spec line 175) is deferred to a Phase 3.1
  // follow-up — needs a canvas → required-connector mapping that doesn't
  // exist in the ontology yet.

  const PRIMARY_TAB_COUNT = 6;

  $: ({ instanceId } = $runtime);

  $: canvasListQuery = createRuntimeServiceListResources(
    instanceId,
    { kind: "rill.runtime.v1.Canvas" },
    { query: { refetchOnMount: "always", staleTime: 0 } },
    queryClient,
  );

  $: canvases = $canvasListQuery?.data?.resources ?? [];
  $: merged = mergeDashboardPrefs(canvases, $dashboardPrefs);

  // Drag-and-drop state. Native HTML5 drag-drop keeps the row layout flexible
  // (checkbox + handle + name + tag) without fighting DraggableList's slot API.
  let dragKey: string | null = null;
  let dragOverKey: string | null = null;

  onMount(loadDashboardPrefs);

  function persist(next: ReturnType<typeof mergeDashboardPrefs>) {
    const prefs: DashboardPref[] = next.map((m, i) => ({
      key: m.key,
      position: i,
      visible: m.visible,
    }));
    setDashboardPrefs(prefs);
  }

  function handleDragStart(e: DragEvent, key: string) {
    if (!e.dataTransfer) return;
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", key);
    dragKey = key;
  }

  function handleDragOver(e: DragEvent, key: string) {
    e.preventDefault();
    if (!e.dataTransfer) return;
    e.dataTransfer.dropEffect = "move";
    dragOverKey = key;
  }

  function handleDragLeave(key: string) {
    if (dragOverKey === key) dragOverKey = null;
  }

  function handleDrop(e: DragEvent, targetKey: string) {
    e.preventDefault();
    const sourceKey = dragKey;
    dragKey = null;
    dragOverKey = null;
    if (!sourceKey || sourceKey === targetKey) return;

    const current = mergeDashboardPrefs(canvases, getStore(dashboardPrefs));
    const fromIndex = current.findIndex((m) => m.key === sourceKey);
    const toIndex = current.findIndex((m) => m.key === targetKey);
    if (fromIndex < 0 || toIndex < 0) return;

    const next = current.slice();
    const [moved] = next.splice(fromIndex, 1);
    next.splice(toIndex, 0, moved);
    persist(next);
  }

  function handleDragEnd() {
    dragKey = null;
    dragOverKey = null;
  }

  function toggleVisible(key: string) {
    const current = mergeDashboardPrefs(canvases, getStore(dashboardPrefs));
    const next = current.map((m) =>
      m.key === key ? { ...m, visible: !m.visible } : m,
    );
    persist(next);
  }

  function displayLabel(m: { resource: any; key: string }): string {
    return m.resource?.canvas?.spec?.displayName || m.key;
  }

  function statusTag(index: number, visible: boolean): string {
    if (!visible) return "hidden";
    if (index < PRIMARY_TAB_COUNT) return "primary";
    return "in more ▾";
  }
</script>

<svelte:head>
  <title>Customize — Bratrax</title>
</svelte:head>

<div
  class="flex h-full w-full items-start justify-center overflow-y-auto bg-surface-base py-12"
>
  <div
    class="relative w-full max-w-3xl border border-bratrax-border bg-surface-elevated p-8"
  >
    <div
      class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
    >
      Customize
    </div>
    <h1 class="text-2xl font-black text-bratrax-text-headline">
      Dashboard visibility &amp; order
    </h1>
    <p class="mt-3 text-sm font-light text-bratrax-text-body">
      Drag to reorder. Uncheck to hide entirely. Items above the line appear as
      primary tabs in the top nav; items below appear inside MORE&nbsp;▾. Saves
      automatically.
    </p>

    <div class="dash-list">
      {#if merged.length === 0}
        <div class="empty">
          <div class="empty-label">No dashboards yet</div>
          <p>
            Your workspace is still being set up. Dashboards appear here once
            the data pipeline finishes its first run — usually within a few
            minutes of completing onboarding.
          </p>
        </div>
      {/if}
      {#each merged as m, i (m.key)}
        {#if i === PRIMARY_TAB_COUNT && merged.length > PRIMARY_TAB_COUNT}
          <div class="threshold">
            <div class="line"></div>
            <span>Visible tabs fit up to here</span>
            <div class="line"></div>
          </div>
        {/if}
        <div
          class="dash-row"
          class:hidden-row={!m.visible}
          class:drag-over={dragOverKey === m.key}
          draggable={true}
          on:dragstart={(e) => handleDragStart(e, m.key)}
          on:dragover={(e) => handleDragOver(e, m.key)}
          on:dragleave={() => handleDragLeave(m.key)}
          on:drop={(e) => handleDrop(e, m.key)}
          on:dragend={handleDragEnd}
          aria-label={`Reorder ${displayLabel(m)}`}
        >
          <span class="handle" aria-hidden="true">⋮⋮</span>
          <button
            type="button"
            class="check"
            class:on={m.visible}
            on:click={() => toggleVisible(m.key)}
            aria-label={m.visible ? "Hide dashboard" : "Show dashboard"}
            aria-pressed={m.visible}
          >
            {#if m.visible}✓{/if}
          </button>
          <span class="name">{displayLabel(m)}</span>
          <span class="tag">{statusTag(i, m.visible)}</span>
        </div>
      {/each}
    </div>

  </div>
</div>

<style lang="postcss">
  .dash-list {
    @apply mt-6 flex flex-col;
  }

  .dash-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 6px;
    border-radius: 3px;
    cursor: grab;
    user-select: none;
  }
  .dash-row:hover {
    background: var(--color-surface);
  }
  .dash-row.drag-over {
    background: var(--color-acid-mid, rgba(212, 255, 0, 0.16));
  }
  .dash-row.hidden-row {
    opacity: 0.55;
  }
  .dash-row.hidden-row .name {
    text-decoration: line-through;
    color: var(--color-text-muted);
  }

  .handle {
    color: var(--color-text-muted);
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 14px;
    line-height: 1;
    cursor: grab;
  }

  .check {
    width: 16px;
    height: 16px;
    border: 1.5px solid var(--color-border-strong, var(--color-border));
    background: transparent;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    line-height: 1;
    color: var(--color-text);
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
  }
  .check.on {
    background: var(--color-acid);
    border-color: var(--color-acid);
    color: var(--color-bg);
    font-weight: 700;
  }

  .name {
    flex: 1;
    font-size: 13px;
    color: var(--color-text);
  }

  .tag {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 9px;
    letter-spacing: 0.12em;
    color: var(--color-text-muted);
    text-transform: uppercase;
  }

  .threshold {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 6px 0;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    letter-spacing: 0.08em;
    color: var(--color-acid);
    text-transform: uppercase;
  }
  .threshold .line {
    flex: 1;
    height: 1px;
    background: var(--color-acid);
    opacity: 0.3;
  }

  .empty {
    padding: 32px 4px;
    text-align: center;
    color: var(--color-text-muted);
    font-size: 13px;
  }
  .empty-label {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-text);
    margin-bottom: 8px;
  }
  .empty p {
    max-width: 360px;
    margin: 0 auto;
    line-height: 1.5;
  }

</style>
