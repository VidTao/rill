<script lang="ts">
  import type { PivotCanvasComponent } from "@rilldata/web-common/features/canvas/components/pivot";
  import { getFiltersForCell } from "@rilldata/web-common/features/dashboards/pivot/pivot-utils";
  import { getContext } from "svelte";
  import ComponentHeader from "../../ComponentHeader.svelte";
  import CanvasPivotRenderer from "./CanvasPivotRenderer.svelte";
  import {
    ORDER_DRILLDOWN_CONTEXT,
    type OrderDrilldownContext,
  } from "./drilldown-context";
  import { validateTableSchema } from "./selector";
  import { tableFieldMapper } from "./util";

  export let component: PivotCanvasComponent;

  // Bratrax order-attribution drilldown bridge. The web-local provider sets
  // this context when an earths_mushrooms user is logged in; otherwise it is
  // undefined and cells stay inert.
  const drilldown = getContext<OrderDrilldownContext | undefined>(
    ORDER_DRILLDOWN_CONTEXT,
  );
  // eslint-disable-next-line no-console
  console.log(
    "[order-drilldown] CanvasPivotDisplay init, context present:",
    !!drilldown,
  );

  $: ({
    parent: {
      metricsView: { getMetricsViewFromName },
    },
    specStore,
    config,
    pivotState,
    pivotDataStore,
  } = component);

  $: tableSpec = $specStore;

  $: ({
    title,
    description,
    show_description_as_tooltip,
    dimension_filters,
    time_filters,
  } = tableSpec);

  $: hasHeader = !!title || !!description;

  $: filters = {
    time_filters,
    dimension_filters,
  };

  // Only wire the cell-click hook if the Bratrax provider is mounted AND the
  // clicked measure is one we drill into. Currently scoped to
  // metric_attributed_orders on campaign_deep_dive; widening means adding more
  // names to DRILLDOWN_MEASURES.
  //
  // Column ids on the pivot are short aliases — `m0`, `m1`, ... indexing into
  // `config.measureNames`. For nested column dimensions the alias is prefixed
  // with `c<i>v<j>_…m<k>`. We pull the trailing `m<k>` off the alias and
  // translate it back to the measure's real name via config.measureNames.
  const DRILLDOWN_MEASURES = new Set(["metric_attributed_orders"]);

  function resolveMeasureName(
    columnId: string,
    measureNames: string[],
  ): string | null {
    const m = columnId.match(/m(\d+)$/);
    if (!m) return null;
    return measureNames[Number(m[1])] ?? null;
  }

  // Predicate used by the renderer to visually mark clickable cells (cursor
  // pointer + hover tint). Without this, cells stay visually inert.
  $: isClickableColumn =
    drilldown && $config
      ? (columnId: string) => {
          const name = resolveMeasureName(columnId, $config!.measureNames);
          return !!name && DRILLDOWN_MEASURES.has(name);
        }
      : undefined;

  $: onMeasureCellClick = drilldown
    ? (rowId: string, columnId: string) => {
        const pivotConfig = $config;
        const dataStore = $pivotDataStore;
        if (!pivotConfig || !dataStore) return;
        const measureName = resolveMeasureName(
          columnId,
          pivotConfig.measureNames,
        );
        if (!measureName || !DRILLDOWN_MEASURES.has(measureName)) return;
        const { filters: cellFilters, timeRange } = getFiltersForCell(
          pivotConfig,
          rowId,
          columnId,
          {},
          dataStore.data,
        );
        // eslint-disable-next-line no-console
        console.log("[order-drilldown] open", {
          measureName,
          rowId,
          columnId,
          filters: cellFilters,
          timeRange,
        });
        drilldown.open({
          measureName,
          filters: cellFilters,
          timeRange: { start: timeRange.start, end: timeRange.end },
        });
      }
    : undefined;

  $: _metricViewSpec = getMetricsViewFromName(tableSpec.metrics_view);
  $: metricsViewSpec = $_metricViewSpec.metricsView;

  $: schema = validateTableSchema(metricsViewSpec, tableSpec);

  $: if ("columns" in tableSpec && schema.isValid) {
    const columns = tableSpec?.columns || [];
    pivotState.update((state) => ({
      ...state,
      sorting: [],
      expanded: {},
      columns: tableFieldMapper(columns, metricsViewSpec),
    }));
  } else if ("col_dimensions" in tableSpec && schema.isValid) {
    const measures = tableSpec.measures || [];
    const colDimensions = tableSpec.col_dimensions || [];
    const rowDimensions = tableSpec.row_dimensions || [];
    pivotState.update((state) => ({
      ...state,
      sorting: [],
      expanded: {},
      columns: [
        ...tableFieldMapper(colDimensions, metricsViewSpec),
        ...tableFieldMapper(measures, metricsViewSpec),
      ],
      rows: tableFieldMapper(rowDimensions, metricsViewSpec),
    }));
  }
</script>

<ComponentHeader
  {component}
  {title}
  {description}
  showDescriptionAsTooltip={show_description_as_tooltip}
  {filters}
/>

<CanvasPivotRenderer
  {hasHeader}
  {schema}
  {pivotDataStore}
  pivotConfig={config}
  {pivotState}
  {onMeasureCellClick}
  {isClickableColumn}
/>
