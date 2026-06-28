import { describe, expect, it } from "vitest";
import { getMetricTreeContextMetricsView } from "./context";

describe("metric tree context metrics view", () => {
  it("uses the authored tree context metrics view when metrics_view is empty", () => {
    expect(
      getMetricTreeContextMetricsView({
        metrics_view: "",
        tree_id: "starter-d2c-hybrid",
        context_metrics_view: "dim_orders_metrics",
      }),
    ).toBe("dim_orders_metrics");
  });

  it("keeps metrics_view as the primary context for metrics-view trees", () => {
    expect(
      getMetricTreeContextMetricsView({
        metrics_view: "metric_tree_nodes",
        context_metrics_view: "dim_orders_metrics",
      }),
    ).toBe("metric_tree_nodes");
  });
});
