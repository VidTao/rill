import type { MetricTreeSpec } from "./columns";

export function getMetricTreeContextMetricsView(spec: MetricTreeSpec): string {
  return spec.metrics_view || spec.context_metrics_view || "";
}
