<script lang="ts">
  import { Handle, Position } from "@xyflow/svelte";
  import { statusTheme } from "./status";
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

  $: theme = statusTheme(data.status);

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
  style:--accent={theme.light}
  style:--accent-dark={theme.dark}
>
  <Handle type="target" position={targetPosition} isConnectable={false} />
  <Handle type="source" position={sourcePosition} isConnectable={false} />

  {#if data.status}
    <div class="top">
      <span class="dot" />
      <span class="status">{theme.label}</span>
    </div>
  {/if}
  <div class="label" title={data.label}>{data.label}</div>
  <div class="value">{fmtValue(data.value, data.unit)}</div>
  {#if data.delta != null}
    <div class="delta">
      {data.delta > 0 ? "▲" : data.delta < 0 ? "▼" : "•"}
      {Math.abs(data.delta).toLocaleString()}
    </div>
  {/if}
</div>

<style>
  .mtnode {
    box-sizing: border-box;
    width: 220px;
    height: 96px;
    padding: 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, var(--accent) 35%, transparent);
    border-left: 4px solid var(--accent);
    background: var(--color-surface, #ffffff);
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
    box-shadow: 0 0 0 2px var(--accent);
  }
  .top {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 9999px;
    background: var(--accent);
    flex-shrink: 0;
  }
  .status {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--accent);
  }
  .label {
    font-size: 13px;
    font-weight: 600;
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
    font-weight: 700;
    line-height: 1.2;
    color: var(--color-text, #1a1a18);
  }
  .delta {
    font-size: 11px;
    color: var(--color-text-muted, #6b7280);
  }

  /* Canvas dashboards pin --color-* to the theme's light values, so dark mode
     needs explicit readable colors (same gotcha the order-attribution graph
     documents). */
  :global(.dark) .mtnode {
    background: #161616;
    border-color: color-mix(in srgb, var(--accent-dark) 55%, #2a2a2a);
    border-left-color: var(--accent-dark);
  }
  :global(.dark) .mtnode.selected {
    box-shadow: 0 0 0 2px var(--accent-dark);
  }
  :global(.dark) .status,
  :global(.dark) .dot {
    color: var(--accent-dark);
  }
  :global(.dark) .dot {
    background: var(--accent-dark);
  }
  :global(.dark) .label,
  :global(.dark) .value {
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
    background: color-mix(in srgb, var(--accent) 30%, #ffffff);
    border: 1px solid var(--accent);
    opacity: 1;
  }
</style>
