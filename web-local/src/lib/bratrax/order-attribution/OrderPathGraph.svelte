<script lang="ts">
  import {
    Background,
    Controls,
    SvelteFlow,
    type Edge,
    type Node,
    type NodeTypes,
  } from "@xyflow/svelte";
  import "@xyflow/svelte/dist/base.css";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import { writable } from "svelte/store";
  import OrderPathLane from "./OrderPathLane.svelte";
  import OrderPathNode from "./OrderPathNode.svelte";
  import { buildOrderPathLayout, type PathNodeData } from "./order-path-layout";
  import type { OrderTimelineRow } from "./types";

  export let rows: OrderTimelineRow[] = [];
  // The `is_*_winner` column for the active attribution model.
  export let winnerColumn = "is_last_touch_winner";
  // Collapse low-signal runs + chip-render behavior events. Flip to false to
  // get the original "every row is a full card" linear layout.
  export let compact = true;
  // Called when a node card is clicked — the parent opens the detail modal.
  export let onSelectRow: (row: OrderTimelineRow) => void = () => {};

  // Above this many real nodes, the initial view frames just the winner +
  // conversion (readable) instead of fitting the whole journey (tiny).
  const FOCUS_THRESHOLD = 9;

  // Collapsed runs the user has expanded back into individual nodes. Run ids
  // encode their contents, so they naturally don't match across a different
  // order — no need to reset when `rows` changes.
  let expandedRuns = new Set<string>();

  function handleNodeClick(
    e: CustomEvent<{ node: Node<PathNodeData>; event: MouseEvent | TouchEvent }>,
  ) {
    const data = e.detail?.node?.data;
    if (!data) return;
    if (data.kind === "collapsed" && data.runId) {
      // Expand this run in place (triggers a re-layout via reactivity).
      const next = new Set(expandedRuns);
      next.add(data.runId);
      expandedRuns = next;
      return;
    }
    if (data.row) onSelectRow(data.row);
  }

  // Match SvelteFlow's chrome (controls, edges, pane) to the app theme.
  $: colorMode = ($themeControl === "dark" ? "dark" : "light") as
    | "dark"
    | "light";

  const nodesStore = writable<Node<PathNodeData>[]>([]);
  const edgesStore = writable<Edge[]>([]);

  const nodeTypes: NodeTypes = {
    "order-path-node": OrderPathNode as NodeTypes[string],
    "order-path-lane": OrderPathLane as NodeTypes[string],
  };

  let height = 420;
  // Signature that remounts SvelteFlow when the graph changes (order switch or
  // a run expand), so fitView re-frames the new paths.
  let graphKey = "";
  let fitViewOptions: Record<string, unknown> = {
    padding: 0.18,
    minZoom: 0.5,
    maxZoom: 1,
    duration: 0,
  };

  $: {
    const layout = buildOrderPathLayout(rows, winnerColumn, {
      compact,
      expandedRuns,
    });
    nodesStore.set(layout.nodes);
    edgesStore.set(layout.edges);
    height = Math.max(420, Math.min(800, layout.height + 64));
    graphKey = layout.nodes.map((n) => n.id).join(",");
    // Long journey: frame the winner + conversion at a readable zoom and let
    // the user pan left for history. Short journey: fit the whole thing.
    fitViewOptions =
      layout.nodeCount > FOCUS_THRESHOLD && layout.focusNodeIds.length
        ? {
            padding: 0.3,
            minZoom: 0.3,
            maxZoom: 1,
            duration: 0,
            nodes: layout.focusNodeIds.map((id) => ({ id })),
          }
        : { padding: 0.18, minZoom: 0.5, maxZoom: 1, duration: 0 };
  }
</script>

{#if $nodesStore.length === 0}
  <div class="empty">No touchpoints to chart for this order.</div>
{:else}
  <div class="graph" style:height={`${height}px`}>
    {#key graphKey}
      <SvelteFlow
        nodes={nodesStore}
        edges={edgesStore}
        {nodeTypes}
        {colorMode}
        proOptions={{ hideAttribution: true }}
        fitView
        {fitViewOptions}
        minZoom={0.2}
        maxZoom={2}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        zoomOnScroll
        zoomOnPinch
        zoomOnDoubleClick
        panOnDrag
        preventScrolling
        onlyRenderVisibleElements={false}
        on:nodeclick={handleNodeClick}
      >
        <Background gap={20} />
        <Controls showLock={false} position="bottom-right" />
      </SvelteFlow>
    {/key}
  </div>
{/if}

<style>
  .graph {
    width: 100%;
    margin: 8px 0;
    border: 1px solid var(--color-bratrax-border, #e5e7eb);
    border-radius: 6px;
    overflow: hidden;
    background: var(--color-surface-subtle, #fafafa);
  }
  :global(.dark) .graph {
    border-color: #2e2e2e;
    background: #0f0f0f;
  }
  .empty {
    padding: 32px 8px;
    text-align: center;
    color: var(--color-text-muted, #6b7280);
    font-size: 14px;
  }
</style>
