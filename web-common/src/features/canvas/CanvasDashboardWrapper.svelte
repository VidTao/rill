<script lang="ts">
  import { onMount } from "svelte";
  import { dynamicHeight } from "@rilldata/web-common/layout/layout-settings.ts";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import CellInspector from "@rilldata/web-common/components/CellInspector.svelte";
  import CanvasFilters from "./filters/CanvasFilters.svelte";
  import { getCanvasStore } from "./state-managers/state-managers";
  import ThemeProvider from "../dashboards/ThemeProvider.svelte";

  export let maxWidth: number;
  export let clientWidth = 0;
  export let showGrabCursor = false;
  export let filtersEnabled: boolean | undefined;
  export let canvasName: string;
  export let embedded: boolean = false;
  export let builder = false;
  export let onClick: () => void = () => {};

  let contentRect = new DOMRectReadOnly(0, 0, 0, 0);

  // The dashboard scroll-container has overflow-y and its content sits in
  // a centered max-width wrapper. The filter-bar header has no scrollbar
  // and centers its own max-width wrapper. Their respective right edges
  // can drift apart by:
  //   - the scrollbar width (only the scroll container has one)
  //   - the scroll container's right padding (`p-2` = 8px) that the header
  //     doesn't pay
  //   - any future internal padding in the dashboard wrapper / its parent
  // Rather than reproduce that math, we bind both wrappers, measure their
  // getBoundingClientRect().right at runtime, and expose the difference
  // as the `--filter-slot-right` CSS variable on the filter wrapper. The
  // slotted right-side content (Bratrax's Edit-dashboard button) reads
  // that variable for its `right:` offset — pixel-aligned regardless of
  // browser scrollbar behaviour.
  let scrollContainerEl: HTMLDivElement;
  let dashboardWrapperEl: HTMLDivElement;
  let filterWrapperEl: HTMLDivElement;
  let scrollbarWidth = 0;
  let filterSlotRight = 0;

  function measureScrollbar() {
    if (!scrollContainerEl) return;
    scrollbarWidth = Math.max(
      0,
      scrollContainerEl.offsetWidth - scrollContainerEl.clientWidth,
    );
  }

  // ItemWrapper (the per-tile container) applies `p-2.5` = 10px padding on
  // every side, so the visible tile border ends 10px shy of the dashboard
  // wrapper's measured right edge. Subtract it to land the slot on the
  // tile's painted right edge instead of the invisible wrapper edge.
  const TILE_INNER_PADDING_PX = 10;

  function measureFilterSlotRight() {
    if (!dashboardWrapperEl || !filterWrapperEl) return;
    const dashRight =
      dashboardWrapperEl.getBoundingClientRect().right - TILE_INNER_PADDING_PX;
    const filterRight = filterWrapperEl.getBoundingClientRect().right;
    // Positive value pushes the slotted content LEFT by `right: {n}px`.
    filterSlotRight = Math.max(0, filterRight - dashRight);
  }

  function measureAll() {
    measureScrollbar();
    measureFilterSlotRight();
  }

  $: ({ instanceId } = $runtime);

  $: ({
    canvasEntity: { theme },
  } = getCanvasStore(canvasName, instanceId));

  $: ({ width: clientWidth } = contentRect);

  // Recompute when the content extent changes (a row appearing/disappearing
  // can flip the scrollbar's visibility, which shifts the dashboard
  // wrapper's right edge).
  $: clientWidth, measureAll();

  onMount(() => {
    measureAll();
    if (typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(measureAll);
    if (scrollContainerEl) observer.observe(scrollContainerEl);
    if (dashboardWrapperEl) observer.observe(dashboardWrapperEl);
    if (filterWrapperEl) observer.observe(filterWrapperEl);
    return () => observer.disconnect();
  });
</script>

<ThemeProvider theme={$theme}>
  <main
    class="flex flex-col overflow-hidden"
    class:w-full={$dynamicHeight}
    class:size-full={!$dynamicHeight}
  >
    {#if filtersEnabled}
      <header
        role="presentation"
        class="bg-surface-subtle border-b py-4 pl-2 w-full h-fit select-none z-50 flex items-center justify-center"
        style:padding-right="{8 + scrollbarWidth}px"
        on:click|self={onClick}
      >
        <!-- Max-width relative wrapper: CanvasFilters fills it; the
             `filter-right` slot is absolutely positioned by the consumer
             relative to this wrapper. The wrapper's right edge is measured
             at runtime against the dashboard wrapper's right edge below;
             the diff is exposed as `--filter-slot-right` so slotted
             content (Bratrax Edit-dashboard button) can offset itself
             precisely instead of guessing geometry. -->
        <div
          bind:this={filterWrapperEl}
          class="relative w-full flex justify-center"
          style:max-width="{maxWidth}px"
          style:--filter-slot-right="{filterSlotRight}px"
        >
          <CanvasFilters {canvasName} {maxWidth} {builder} />
          <slot name="filter-right" />
        </div>
      </header>
    {/if}

    <div
      bind:this={scrollContainerEl}
      role="presentation"
      id="canvas-scroll-container"
      class="p-2 flex flex-col items-center bg-surface-background select-none overflow-y-auto overflow-x-hidden"
      class:!cursor-grabbing={showGrabCursor}
      class:w-full={$dynamicHeight}
      class:size-full={!$dynamicHeight}
      class:pb-48={!embedded}
      on:click|self={onClick}
    >
      <div
        bind:this={dashboardWrapperEl}
        class="w-full h-fit flex flex-col items-center row-container relative"
        style:max-width="{maxWidth}px"
        style:min-width="420px"
        bind:contentRect
      >
        <slot />
      </div>
    </div>

    <CellInspector />
  </main>
</ThemeProvider>

<style>
  div {
    container-type: inline-size;
    container-name: canvas-container;
  }
</style>
