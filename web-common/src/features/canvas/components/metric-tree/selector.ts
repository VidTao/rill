import type { CanvasStore } from "@rilldata/web-common/features/canvas/state-managers/state-managers";
import { derived, type Readable } from "svelte/store";
import {
  REQUIRED_ROLES,
  resolveTreeColumns,
  type MetricTreeSpec,
} from "./columns";

// Validates that the metric_tree's required structural columns resolve to
// dimensions on the selected metrics view, and that the value column (if set)
// is a measure. Empty data is NOT a validation failure — the display renders an
// empty state for that.
export function validateMetricTreeSchema(
  ctx: CanvasStore,
  spec: MetricTreeSpec,
): Readable<{ isValid: boolean; error?: string; isLoading?: boolean }> {
  return derived(
    ctx.canvasEntity.metricsView.getMetricsViewFromName(spec.metrics_view),
    ({ metricsView, isLoading }) => {
      if (isLoading) {
        return { isValid: true, isLoading: true };
      }
      if (!metricsView) {
        return {
          isValid: false,
          error: `Metrics view ${spec.metrics_view} not found`,
        };
      }

      const cols = resolveTreeColumns(spec);
      const isDimension = (c: string) =>
        metricsView.dimensions?.some((d) => d.name === c || d.column === c) ??
        false;
      const isMeasure = (c: string) =>
        metricsView.measures?.some((m) => m.name === c) ?? false;

      const missing = REQUIRED_ROLES.filter((role) => !isDimension(cols[role]));
      if (missing.length) {
        const human = missing.map((r) => `${r} ("${cols[r]}")`).join(", ");
        return {
          isValid: false,
          error:
            `Metric tree is missing required column(s): ${human}. ` +
            `Map them in the sidebar or expose them as dimensions in "${spec.metrics_view}".`,
        };
      }

      if (spec.value && !isMeasure(spec.value)) {
        return {
          isValid: false,
          error: `Value column "${spec.value}" is not a measure in ${spec.metrics_view}`,
        };
      }

      return { isValid: true };
    },
  );
}
