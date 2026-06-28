<script lang="ts">
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import {
    Background,
    Controls,
    SvelteFlow,
    type Edge,
    type Node,
    type NodeTypes,
  } from "@xyflow/svelte";
  import "@xyflow/svelte/dist/base.css";
  import { onDestroy, onMount } from "svelte";
  import { writable } from "svelte/store";
  import type { TreeOrientation } from "./columns";
  import { NODE_TYPE_ORDER, nodeTypeTheme } from "./node-types";
  import { buildMetricTreeLayout, NODE_TYPE } from "./layout";
  import MetricTreeNode from "./MetricTreeNode.svelte";
  import type { MetricTreeEdgeData, MetricTreeNodeData } from "./types";

  export let nodes: MetricTreeNodeData[] = [];
  export let edges: MetricTreeEdgeData[] = [];
  export let orientation: TreeOrientation = "TB";
  export let selectedId: string | null = null;
  export let onSelect: (node: MetricTreeNodeData) => void = () => {};

  const nodesStore = writable<Node<MetricTreeNodeData>[]>([]);
  const edgesStore = writable<Edge[]>([]);
  const nodeTypes: NodeTypes = {
    [NODE_TYPE]: MetricTreeNode as NodeTypes[string],
  };
  const fitViewOptions = {
    padding: 0.2,
    minZoom: 0.4,
    maxZoom: 1,
    duration: 0,
  };

  $: layout = buildMetricTreeLayout(nodes, edges, { orientation });
  // Selection is folded into node data WITHOUT remounting (flowKey excludes
  // selectedId), so clicking a node never resets the user's pan/zoom.
  $: nodesStore.set(
    layout.nodes.map((n) => ({ ...n, selected: n.id === selectedId })),
  );
  $: edgesStore.set(layout.edges);
  $: nodeSig = layout.nodes.map((n) => n.id).join(",");

  // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion -- svelte-check widens the ternary to `string`; SvelteFlow needs the literal union
  $: colorMode = ($themeControl === "dark" ? "dark" : "light") as
    | "dark"
    | "light";

  // Remount SvelteFlow on tile resize / data / orientation change so fitView
  // reframes the tree to the new bounds.
  let containerEl: HTMLDivElement;
  let containerKey = "";
  let ro: ResizeObserver | null = null;

  onMount(() => {
    if (typeof ResizeObserver === "undefined") return;
    ro = new ResizeObserver(([entry]) => {
      if (!entry) return;
      const next = `${Math.round(entry.contentRect.width)}x${Math.round(
        entry.contentRect.height,
      )}`;
      if (next !== containerKey) containerKey = next;
    });
    if (containerEl) ro.observe(containerEl);
  });
  onDestroy(() => {
    ro?.disconnect();
    ro = null;
  });

  $: flowKey = `mt|${orientation}|${nodeSig}|${containerKey}`;

  function handleNodeClick(
    e: CustomEvent<{
      node: Node<MetricTreeNodeData>;
      event: MouseEvent | TouchEvent;
    }>,
  ) {
    const d = e.detail?.node?.data;
    if (d) onSelect(d);
  }
</script>

<div class="graph" bind:this={containerEl}>
  <div class="legend" aria-label="Metric tree node types">
    {#each NODE_TYPE_ORDER as nodeType (nodeType)}
      {@const theme = nodeTypeTheme(nodeType)}
      <span class="legend-item" style:--legend-color={theme.color}>
        <span class="swatch" />
        {theme.label}
      </span>
    {/each}
  </div>
  {#key flowKey}
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

<style lang="postcss">
  .graph {
    @apply relative size-full overflow-hidden;
  }
  .legend {
    @apply absolute left-2 top-2 z-10 flex flex-wrap items-center gap-1.5 rounded border border-gray-200 bg-surface/95 px-2 py-1 shadow-sm;
    max-width: calc(100% - 16px);
  }
  .legend-item {
    @apply inline-flex items-center gap-1 whitespace-nowrap text-[10px] font-semibold;
    color: var(--color-text-muted, #6b7280);
  }
  .swatch {
    width: 8px;
    height: 8px;
    border-radius: 9999px;
    background: var(--legend-color);
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--legend-color) 45%, transparent);
  }
  :global(.dark) .legend {
    background: rgba(22, 22, 22, 0.94);
    border-color: #2a2a2a;
  }
  :global(.dark) .legend-item {
    color: #a39d90;
  }
</style>
