// Bridge between the upstream canvas pivot (web-common) and the Bratrax
// order-attribution drilldown (web-local). The canvas pivot reads an optional
// handler from Svelte context; web-local mounts a provider that sets it.
// When the context is absent (any non-Bratrax build, or a session for a client
// the drilldown isn't gated on) the cell stays inert — zero cost.

import type { V1Expression } from "@rilldata/web-common/runtime-client";

export const ORDER_DRILLDOWN_CONTEXT = "bratrax:order-drilldown";

export interface MeasureCellClickContext {
  // Measure that was clicked, e.g. "metric_attributed_orders". The provider
  // uses this to ignore clicks on measures it doesn't drill into.
  measureName: string;
  // Merged filter: cell-row filters + dashboard where-filter. Drop this
  // straight into a metrics-view rows query as `where`. The provider can also
  // walk this expression to extract per-dimension values for display, since
  // each cell filter is an in(<dim>, [<val>]) leaf.
  filters: V1Expression | undefined;
  // Time range that scopes the cell, already merged with the dashboard's
  // time selection. Strings are ISO timestamps.
  timeRange: { start: string | undefined; end: string | undefined };
}

export interface OrderDrilldownContext {
  open: (ctx: MeasureCellClickContext) => void;
}
