<script lang="ts">
  import LoadingSpinner from "@rilldata/web-common/components/icons/LoadingSpinner.svelte";
  import ComponentError from "@rilldata/web-common/features/components/ComponentError.svelte";
  import { createInExpression } from "@rilldata/web-common/features/dashboards/stores/filter-utils";
  import { createQueryServiceMetricsViewAggregation } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import ComponentHeader from "../../ComponentHeader.svelte";
  import { getCanvasStore } from "../../state-managers/state-managers";
  import { buildMetricTree } from "./build-tree";
  import { resolveTreeColumns } from "./columns";
  import type { MetricTreeComponent } from "./index";
  import MetricTreeDetail from "./MetricTreeDetail.svelte";
  import MetricTreeGraph from "./MetricTreeGraph.svelte";
  import { validateMetricTreeSchema } from "./selector";
  import type { MetricTreeNodeData } from "./types";

  export let component: MetricTreeComponent;

  $: ({ instanceId } = $runtime);
  $: ({
    specStore,
    visible,
    parent: { name: canvasName },
  } = component);
  $: spec = $specStore;

  $: ctx = getCanvasStore(canvasName, instanceId);
  $: ({
    metricsView: { getDimensionsForMetricView, getMeasuresForMetricView },
  } = ctx.canvasEntity);

  $: schema = validateMetricTreeSchema(ctx, spec);
  $: ({ isValid, error: schemaError, isLoading: schemaLoading } = $schema);

  $: cols = resolveTreeColumns(spec);
  $: allDimensions = getDimensionsForMetricView(spec.metrics_view);
  $: allMeasures = getMeasuresForMetricView(spec.metrics_view);
  $: dimNameSet = new Set(
    $allDimensions.map((d) => (d.name || d.column) as string),
  );
  $: measNameSet = new Set($allMeasures.map((m) => m.name as string));

  // Every mapped column we want back, de-duped. tree_column is only needed when
  // filtering to a specific tree.
  $: wantedColumns = Array.from(
    new Set(
      [
        cols.node_id,
        cols.parent_node_id,
        cols.label,
        cols.value,
        cols.unit,
        cols.delta_value,
        cols.status,
        cols.sort_order,
        cols.confidence,
        cols.limitation,
        cols.observation,
        cols.suggested_test,
        cols.success_metric,
        spec.tree_name ? cols.tree_column : undefined,
      ].filter((c): c is string => !!c),
    ),
  );

  // Split into measures vs dimensions by what the metrics view actually models.
  $: dimensions = wantedColumns
    .filter((c) => dimNameSet.has(c))
    .map((name) => ({ name }));
  $: measures = wantedColumns
    .filter((c) => measNameSet.has(c))
    .map((name) => ({ name }));

  $: treeWhere =
    spec.tree_name && dimNameSet.has(cols.tree_column)
      ? createInExpression(cols.tree_column, [spec.tree_name])
      : undefined;

  $: rowsQuery = createQueryServiceMetricsViewAggregation(
    instanceId,
    spec.metrics_view,
    {
      dimensions,
      measures,
      where: treeWhere,
      limit: "1000",
      priority: 30,
    },
    {
      query: {
        enabled: isValid && $visible && dimensions.length > 0,
      },
    },
  );
  $: ({ isLoading: rowsLoading, data: rowsData } = $rowsQuery);

  $: rows = (rowsData?.data ?? []) as unknown as Record<string, unknown>[];
  $: tree = buildMetricTree(rows, cols, { tree: spec.tree_name });

  // Detail panel selection. Drop it if the selected node leaves the tree
  // (e.g. metrics_view or tree changed).
  let selectedNode: MetricTreeNodeData | null = null;
  $: if (selectedNode && !tree.nodes.some((n) => n.id === selectedNode?.id)) {
    selectedNode = null;
  }

  function handleSelect(n: MetricTreeNodeData) {
    selectedNode = n;
  }

  $: ({ title, description, show_description_as_tooltip } = spec);
  $: orientation = spec.orientation ?? "TB";
</script>

{#if schemaLoading}
  <div class="state"><LoadingSpinner size="36px" /></div>
{:else if !isValid}
  <ComponentError error={schemaError} />
{:else}
  <ComponentHeader
    {component}
    {title}
    {description}
    showDescriptionAsTooltip={show_description_as_tooltip}
  />

  {#if rowsLoading}
    <div class="state grow"><LoadingSpinner size="36px" /></div>
  {:else if tree.isEmpty}
    <div class="state grow">
      <div class="empty">
        No metric tree data{spec.tree_name ? ` for "${spec.tree_name}"` : ""}.
      </div>
    </div>
  {:else}
    <div class="relative size-full grow">
      <MetricTreeGraph
        nodes={tree.nodes}
        edges={tree.edges}
        {orientation}
        selectedId={selectedNode?.id ?? null}
        onSelect={handleSelect}
      />
      <MetricTreeDetail
        node={selectedNode}
        onClose={() => (selectedNode = null)}
      />
    </div>
  {/if}
{/if}

<style lang="postcss">
  .state {
    @apply flex size-full items-center justify-center;
  }
  .empty {
    @apply rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-fg-muted;
  }
</style>
