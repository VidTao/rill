<script lang="ts">
  import type { Position } from "@xyflow/svelte";
  import { sourceColor, type PathNodeData } from "./order-path-layout";

  export let data: PathNodeData;

  // XYFlow-injected props, declared to silence unknown-prop warnings.
  export let id: string | undefined = undefined;
  export let type: string | undefined = undefined;
  export let selected = false;
  export let width: number | undefined = undefined;
  export let height: number | undefined = undefined;
  export let sourcePosition: Position | undefined = undefined;
  export let targetPosition: Position | undefined = undefined;
  export let dragging = false;
  export let zIndex = 0;
  export let isConnectable = true;
  export let positionAbsoluteX = 0;
  export let positionAbsoluteY = 0;

  const ensureFlowProps = (..._args: unknown[]) => {};
  $: ensureFlowProps(
    id,
    type,
    selected,
    width,
    height,
    sourcePosition,
    targetPosition,
    dragging,
    zIndex,
    isConnectable,
    positionAbsoluteX,
    positionAbsoluteY,
  );

  $: accent = sourceColor(data.source);
</script>

<div class="lane" style:--accent={accent}>
  <span class="dot" />
  <div class="text">
    <div class="name" title={data.sourceLabel}>{data.sourceLabel}</div>
    {#if data.laneCount != null}
      <div class="count">
        {data.laneCount} step{data.laneCount === 1 ? "" : "s"}
      </div>
    {/if}
  </div>
</div>

<style>
  .lane {
    box-sizing: border-box;
    width: 160px;
    height: 104px;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 4px;
  }
  .dot {
    flex-shrink: 0;
    width: 10px;
    height: 10px;
    border-radius: 9999px;
    background: var(--accent);
  }
  .text {
    min-width: 0;
  }
  .name {
    font-size: 12px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--color-text, #1a1a18);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .count {
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    color: var(--color-text-muted, #9ca3af);
  }
  /* Dashboard themes can pin --color-* to light values inside the canvas, so
     set readable lane-label colors explicitly in dark mode. */
  :global(.dark) .name {
    color: #ece7dd;
  }
  :global(.dark) .count {
    color: #a39d90;
  }
</style>
