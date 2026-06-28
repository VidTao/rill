<script lang="ts">
  import { Handle, Position } from "@xyflow/svelte";
  import { nodeTypeTheme } from "./node-types";
  import { statusLabel, statusTheme } from "./status";
  import type { MetricTreeNodeData } from "./types";

  export let data: MetricTreeNodeData;
  export let selected = false;
  // Positions are injected by the layout (Bottom/Top for TB, Right/Left for LR).
  export let sourcePosition: Position = Position.Bottom;
  export let targetPosition: Position = Position.Top;

  // Other props @xyflow/svelte injects onto a custom node; declared so Svelte
  // does not warn about unknown props, then no-op'd.
  export let id: string | undefined = undefined;
  export let type: string | undefined = undefined;
  export let width: number | undefined = undefined;
  export let height: number | undefined = undefined;
  export let dragHandle: string | undefined = undefined;
  export let parentId: string | undefined = undefined;
  export let dragging = false;
  export let zIndex = 0;
  export let selectable = false;
  export let deletable = false;
  export let draggable = false;
  export let isConnectable = false;
  export let positionAbsoluteX = 0;
  export let positionAbsoluteY = 0;

  const ensureFlowProps = (..._args: unknown[]) => {};
  $: ensureFlowProps(
    id,
    type,
    width,
    height,
    dragHandle,
    parentId,
    dragging,
    zIndex,
    selectable,
    deletable,
    draggable,
    isConnectable,
    positionAbsoluteX,
    positionAbsoluteY,
  );

  $: typeTheme = nodeTypeTheme(data.nodeType);
  $: healthTheme = statusTheme(data.status);
  $: healthLabel = statusLabel(data.status);
  $: sourceText = data.sourceStatus === "live"
    ? "Live"
    : data.sourceStatus === "computed"
      ? "Computed"
      : data.sourceStatus === "fallback"
        ? "Fallback"
        : data.sourceStatus === "error"
          ? "Source error"
          : data.sourceStatus === "unbound"
            ? "Unbound"
            : null;

  function fmtValue(v: number | string | null, unit: string | null): string {
    if (v == null || v === "") return "—";
    const n = typeof v === "number" ? v : Number(v);
    if (Number.isFinite(n)) {
      if (unit === "percent") return `${n.toLocaleString()}%`;
      if (unit === "usd" || unit === "currency") {
        return n.toLocaleString(undefined, {
          style: "currency",
          currency: "USD",
          maximumFractionDigits: 0,
        });
      }
      if (unit === "count") return n.toLocaleString();
      return `${n.toLocaleString()}${unit ? ` ${unit}` : ""}`;
    }
    return String(v);
  }
</script>

<div
  class="mtnode"
  class:selected
  style:--type-color={typeTheme.color}
  style:--status-color={healthTheme.light}
  style:--status-color-dark={healthTheme.dark}
>
  <Handle type="target" position={targetPosition} isConnectable={false} />
  <Handle type="source" position={sourcePosition} isConnectable={false} />

  <div class="top">
    <span class="type-pill">{typeTheme.label}</span>
    {#if healthLabel}
      <span class="status-pill">
        <span class="dot" />
        {healthLabel}
      </span>
    {/if}
  </div>
  <div class="label" title={data.label}>{data.label}</div>
  <div class="value">{fmtValue(data.value, data.unit)}</div>
  {#if sourceText}
    <div class:error={data.sourceStatus === "error"} class="source">{sourceText}{data.sourceContext ? ` · ${data.sourceContext}` : ""}</div>
  {/if}
  {#if data.value2 != null}
    <div class="metric2">
      {#if data.value2Label}<span class="m2-label">{data.value2Label}</span>{/if}
      <span class="m2-val">{fmtValue(data.value2, data.value2Unit)}</span>
    </div>
  {/if}
  {#if data.value3 != null}
    <div class="metric2">
      {#if data.value3Label}<span class="m2-label">{data.value3Label}</span>{/if}
      <span class="m2-val">{fmtValue(data.value3, data.value3Unit)}</span>
    </div>
  {/if}
  {#if data.delta != null}
    <div class="delta">
      {data.delta > 0 ? "▲" : data.delta < 0 ? "▼" : "•"}
      {fmtValue(Math.abs(data.delta), data.unit)}{data.deltaPercent != null ? ` (${(data.deltaPercent * 100).toFixed(1)}%)` : ""}
    </div>
  {/if}
</div>

<style>
  .mtnode {
    box-sizing: border-box;
    width: 220px;
    height: 144px;
    padding: 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 3px;
    overflow: hidden;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, var(--type-color) 38%, transparent);
    border-left: 5px solid var(--type-color);
    background: color-mix(in srgb, var(--type-color) 5%, var(--color-surface, #ffffff));
    box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
    cursor: pointer;
    transition:
      box-shadow 120ms ease,
      transform 120ms ease;
  }
  .mtnode:hover {
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.18);
    transform: translateY(-1px);
  }
  .mtnode.selected {
    box-shadow: 0 0 0 2px var(--type-color);
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
    min-height: 18px;
  }
  .type-pill,
  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    border-radius: 9999px;
    font-size: 9px;
    font-weight: 800;
    letter-spacing: 0.4px;
    line-height: 1;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .type-pill {
    color: var(--type-color);
  }
  .status-pill {
    max-width: 94px;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--status-color);
  }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 9999px;
    background: var(--status-color);
    flex-shrink: 0;
  }
  .source {
    font-size: 10px;
    font-weight: 700;
    color: var(--type-color);
    opacity: 0.78;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .source.error {
    color: var(--status-color);
  }
  .label {
    font-size: 13px;
    font-weight: 650;
    line-height: 1.2;
    color: var(--color-text, #1a1a18);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .value {
    font-size: 18px;
    font-weight: 750;
    line-height: 1.2;
    color: var(--color-text, #1a1a18);
  }
  .metric2 {
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-size: 11px;
    line-height: 1.3;
    color: var(--color-text-muted, #6b7280);
  }
  .m2-label {
    text-transform: uppercase;
    letter-spacing: 0.3px;
    font-weight: 700;
    font-size: 9px;
  }
  .m2-val {
    font-weight: 600;
    color: var(--color-text, #1a1a18);
  }
  .delta {
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
  }

  :global(.dark) .mtnode {
    background: color-mix(in srgb, var(--type-color) 13%, #161616);
    border-color: color-mix(in srgb, var(--type-color) 60%, #2a2a2a);
    border-left-color: var(--type-color);
  }
  :global(.dark) .mtnode.selected {
    box-shadow: 0 0 0 2px var(--type-color);
  }
  :global(.dark) .status-pill {
    color: var(--status-color-dark);
  }
  :global(.dark) .dot {
    background: var(--status-color-dark);
  }
  :global(.dark) .label,
  :global(.dark) .value,
  :global(.dark) .m2-val {
    color: #ece7dd;
  }
  :global(.dark) .delta {
    color: #a39d90;
  }
  :global(.svelte-flow__handle) {
    width: 6px;
    height: 6px;
    min-width: 6px;
    min-height: 6px;
    border-radius: 9999px;
    background: color-mix(in srgb, var(--type-color) 30%, #ffffff);
    border: 1px solid var(--type-color);
    opacity: 1;
  }
</style>
