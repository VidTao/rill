<script lang="ts">
  import type { PivotCanvasComponent } from "@rilldata/web-common/features/canvas/components/pivot";
  import { getFiltersForCell } from "@rilldata/web-common/features/dashboards/pivot/pivot-utils";
  import type { V1Expression } from "@rilldata/web-common/runtime-client";
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

  // Bratrax attribution/profile drilldown bridge. The web-local provider
  // sets this context; when it is absent, drilldown cells stay inert.
  const drilldown = getContext<OrderDrilldownContext | undefined>(
    ORDER_DRILLDOWN_CONTEXT,
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

  // Only wire the cell-click hook if the Bratrax provider is mounted and
  // the clicked cell is an enabled drilldown target. Campaign deep dive
  // uses metric_attributed_orders; Profile Explorer uses the profiles measure.
  //
  // Column ids are either real measure names (`profiles`) or short aliases
  // (`m0`, `m1`, ...). For nested column dimensions the alias can be prefixed
  // with `c<i>v<j>_...m<k>`, so we also resolve the trailing measure index.
  const DRILLDOWN_MEASURES = new Set(["metric_attributed_orders", "profiles"]);

  function resolveMeasureName(
    columnId: string,
    measureNames: string[],
  ): string | null {
    if (measureNames.includes(columnId)) return columnId;
    const m = columnId.match(/m(\d+)$/);
    if (!m) return null;
    return measureNames[Number(m[1])] ?? null;
  }

  function expressionIncludesDimension(
    expression: V1Expression | undefined,
    dimensionName: string,
  ): boolean {
    if (!expression?.cond?.exprs?.length) return false;
    const [left] = expression.cond.exprs;
    if (left?.ident === dimensionName) return true;
    return expression.cond.exprs.some((expr) =>
      expressionIncludesDimension(expr, dimensionName),
    );
  }

  function getCellFilters(rowId: string, columnId: string) {
    const pivotConfig = $config;
    const dataStore = $pivotDataStore;
    if (!pivotConfig || !dataStore) return undefined;
    return getFiltersForCell(pivotConfig, rowId, columnId, {}, dataStore.data);
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
    ? (_rowId: string, columnId: string, _rowData?: Record<string, unknown>) => {
        const pivotConfig = $config;
        const dataStore = $pivotDataStore;
        if (!pivotConfig || !dataStore) return;
        const measureName = resolveMeasureName(
          columnId,
          pivotConfig.measureNames,
        );
        if (!measureName || !DRILLDOWN_MEASURES.has(measureName)) {
          return;
        }
        const cell = getCellFilters(_rowId, columnId);
        if (!cell) return;
        const { filters: rawCellFilters, timeRange } = cell;
        if (
          measureName === "profiles" &&
          !expressionIncludesDimension(rawCellFilters, "profile_id")
        ) {
          return;
        }
        drilldown.open({
          measureName,
          filters: rawCellFilters,
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
