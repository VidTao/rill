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
  // Signature that remounts SvelteFlow when the order (and thus the graph)
  // changes, so fitView re-centers on the new paths.
  let graphKey = "";

  $: {
    const layout = buildOrderPathLayout(rows, winnerColumn);
    nodesStore.set(layout.nodes);
    edgesStore.set(layout.edges);
    height = Math.max(420, Math.min(800, layout.height + 64));
    graphKey = layout.nodes.map((n) => n.id).join(",");
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
        fitViewOptions={{ padding: 0.15, maxZoom: 1.1, duration: 0 }}
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
