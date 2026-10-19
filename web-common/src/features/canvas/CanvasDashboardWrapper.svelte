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
  // slot container below (Bratrax's freshness badge + Edit-dashboard
  // button) uses it for its `right:` offset — pixel-aligned regardless of
  // browser scrollbar behaviour.
  let scrollContainerEl: HTMLDivElement;
  let dashboardWrapperEl: HTMLDivElement;
  let filterWrapperEl: HTMLDivElement;
  let filterSlotEl: HTMLDivElement;
  let scrollbarWidth = 0;
  let filterSlotRight = 0;
  // Width the slotted right-hand group (Bratrax: freshness badge + Edit
  // dashboard) takes out of the filter bar's first row, published as
  // `--filter-slot-reserve`. CanvasFilters pads that row by it, so the time
  // and comparison pills wrap to the next line before reaching the group
  // instead of sliding underneath it — which they did whenever a sidebar
  // narrowed the page, since the group is absolutely positioned.
  let filterSlotReserve = 0;
  // When even the widest first-row pill (the time range) can't fit beside the
  // slotted group, wrapping can't help: the group moves onto its own line
  // above the filters instead, still right-aligned.
  let filterSlotStacked = false;
  const FILTER_SLOT_GAP_PX = 8;

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

  function measureFilterSlotReserve() {
    const slotWidth = filterSlotEl?.offsetWidth ?? 0;
    const reserve =
      slotWidth > 0 ? slotWidth + filterSlotRight + FILTER_SLOT_GAP_PX : 0;
    // Room the first row needs for its widest pill, from the wrapper's left
    // edge. Measured independently of the reserve (pill widths and the
    // wrapper's width don't depend on it), so stacking can't oscillate.
    let needed = 0;
    const pills = filterWrapperEl?.querySelector("[data-filter-first-row]");
    if (pills && reserve > 0) {
      const wrapperLeft = filterWrapperEl.getBoundingClientRect().left;
      const pillsLeft = pills.getBoundingClientRect().left;
      const widest = Math.max(
        0,
        ...Array.from(pills.children, (el) => (el as HTMLElement).offsetWidth),
      );
      needed = pillsLeft - wrapperLeft + widest + FILTER_SLOT_GAP_PX;
    }
    filterSlotStacked =
      reserve > 0 && needed + reserve > (filterWrapperEl?.clientWidth ?? 0);
    filterSlotReserve = filterSlotStacked ? 0 : reserve;
  }

  function measureAll() {
    measureScrollbar();
    measureFilterSlotRight();
    measureFilterSlotReserve();
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
    // The slotted content changes width on its own (the freshness badge's
    // text changes between "Updated 3 hours ago" and "Next update in …").
    if (filterSlotEl) observer.observe(filterSlotEl);
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
             `filter-right` slot sits absolutely positioned at its top right.
             The wrapper's right edge is measured at runtime against the
             dashboard wrapper's right edge below; the diff
             (`--filter-slot-right`) offsets the slot so it lines up with the
             tiles, and the slot's own width (`--filter-slot-reserve`) is kept
             free in CanvasFilters' first row so nothing slides under it. -->
        <div
          bind:this={filterWrapperEl}
          class="relative w-full flex justify-center"
          class:flex-col={filterSlotStacked}
          style:max-width="{maxWidth}px"
          style:--filter-slot-right="{filterSlotRight}px"
          style:--filter-slot-reserve="{filterSlotReserve}px"
        >
          <CanvasFilters {canvasName} {maxWidth} {builder} />
          <div
            bind:this={filterSlotEl}
            class="filter-right-slot"
            class:stacked={filterSlotStacked}
            style:right={filterSlotStacked ? undefined : `${filterSlotRight}px`}
            style:margin-right={filterSlotStacked
              ? `${filterSlotRight}px`
              : undefined}
          >
            <slot name="filter-right" />
          </div>
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

  .filter-right-slot {
    position: absolute;
    top: 0;
    z-index: 60;
    /* Not a size container: inline-size containment would stop this box
       from sizing to its content, so it would measure (and render) 0 wide. */
    container-type: normal;
  }

  /* Too narrow to sit beside the pills: its own right-aligned line, above. */
  .filter-right-slot.stacked {
    position: static;
    order: -1;
    align-self: flex-end;
    margin-bottom: 8px;
  }
</style>
