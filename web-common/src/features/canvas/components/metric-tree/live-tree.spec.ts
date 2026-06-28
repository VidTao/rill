import { describe, expect, it } from "vitest";
import type { AuthoredMetricTree } from "./authored-tree";
import {
  metricBindingKey,
  planLiveMetricRequests,
  resolveMetricTreeValues,
} from "./live-tree";

function tree(nodes: AuthoredMetricTree["nodes"]): AuthoredMetricTree {
  return {
    tree_id: "t",
    name: "Tree",
    root_node_id: "revenue",
    nodes,
  };
}

describe("metric tree live bindings", () => {
  it("groups Rill metrics-view bindings into live requests", () => {
    const t = tree([
      {
        id: "orders",
        parentId: null,
        label: "Orders",
        type: "output",
        unit: "count",
        value: null,
        metricBinding: {
          source: "rill_metrics_view",
          metricsView: "dim_orders_metrics",
          measure: "metric_total_orders",
          timeMode: "dashboard",
        },
      },
      {
        id: "aov",
        parentId: null,
        label: "AOV",
        type: "output",
        unit: "currency",
        value: null,
        metricBinding: {
          source: "rill_metrics_view",
          metricsView: "dim_orders_metrics",
          measure: "metric_avg_order_value",
          timeMode: "dashboard",
        },
      },
    ]);

    expect(planLiveMetricRequests(t)).toEqual([
      {
        key: "dim_orders_metrics::dashboard::[]",
        metricsView: "dim_orders_metrics",
        measures: ["metric_total_orders", "metric_avg_order_value"],
        filters: [],
        timeMode: "dashboard",
      },
    ]);
  });

  it("uses live leaf values and computes parents from formulas", () => {
    const t = tree([
      { id: "revenue", parentId: null, label: "Revenue", type: "output", unit: "currency", value: 0, formula: { op: "multiply", operands: ["orders", "aov"] } },
      { id: "orders", parentId: "revenue", label: "Orders", type: "driver", unit: "count", value: 1, metricBinding: { source: "rill_metrics_view", metricsView: "dim_orders_metrics", measure: "metric_total_orders" } },
      { id: "aov", parentId: "revenue", label: "AOV", type: "driver", unit: "currency", value: 1, metricBinding: { source: "rill_metrics_view", metricsView: "dim_orders_metrics", measure: "metric_avg_order_value" } },
    ]);
    const orderKey = metricBindingKey(t.nodes[1].metricBinding! as never)!;
    const aovKey = metricBindingKey(t.nodes[2].metricBinding! as never)!;

    const resolved = resolveMetricTreeValues(t, {
      [orderKey]: { value: 100 },
      [aovKey]: { value: 25 },
    });

    expect(resolved?.nodeResolutions.revenue).toMatchObject({
      value: 2500,
      computedValue: 2500,
      sourceStatus: "computed",
    });
    expect(resolved?.nodes.find((n) => n.id === "revenue")?.value).toBe(2500);
  });

  it("rolls previous-period values through formulas and computes movement", () => {
    const t = tree([
      { id: "revenue", parentId: null, label: "Revenue", type: "output", unit: "currency", value: 0, formula: { op: "multiply", operands: ["orders", "aov"] } },
      { id: "orders", parentId: "revenue", label: "Orders", type: "driver", unit: "count", value: 1, metricBinding: { source: "rill_metrics_view", metricsView: "dim_orders_metrics", measure: "metric_total_orders" } },
      { id: "aov", parentId: "revenue", label: "AOV", type: "driver", unit: "currency", value: 1, metricBinding: { source: "rill_metrics_view", metricsView: "dim_orders_metrics", measure: "metric_avg_order_value" } },
    ]);
    const orderKey = metricBindingKey(t.nodes[1].metricBinding! as never)!;
    const aovKey = metricBindingKey(t.nodes[2].metricBinding! as never)!;

    const resolved = resolveMetricTreeValues(t, {
      [orderKey]: { value: 120, previousValue: 100 },
      [aovKey]: { value: 25, previousValue: 20 },
    });

    expect(resolved?.nodeResolutions.revenue).toMatchObject({
      value: 3000,
      previousValue: 2000,
      computedValue: 3000,
      previousComputedValue: 2000,
      delta: 1000,
      deltaPercent: 0.5,
      sourceStatus: "computed",
    });
    expect(resolved?.nodes.find((n) => n.id === "revenue")?.delta).toBe(1000);
  });

  it("tracks measured-vs-computed drift when both exist", () => {
    const t = tree([
      { id: "revenue", parentId: null, label: "Revenue", type: "output", unit: "currency", value: 0, metricBinding: { source: "rill_metrics_view", metricsView: "dim_store_metrics", measure: "metric_total_sales" }, formula: { op: "add", operands: ["one_time", "subs"] } },
      { id: "one_time", parentId: "revenue", label: "One-time", type: "revenue_stream", unit: "currency", value: 60 },
      { id: "subs", parentId: "revenue", label: "Subs", type: "revenue_stream", unit: "currency", value: 40 },
    ]);
    const revenueKey = metricBindingKey(t.nodes[0].metricBinding! as never)!;

    const resolved = resolveMetricTreeValues(t, {
      [revenueKey]: { value: 110 },
    });

    expect(resolved?.nodeResolutions.revenue).toMatchObject({
      value: 110,
      measuredValue: 110,
      computedValue: 100,
      driftValue: 10,
      driftPercent: 0.1,
      sourceStatus: "live",
    });
  });

  it("marks unresolved live bindings as errors instead of silently using fallback", () => {
    const t = tree([
      { id: "orders", parentId: null, label: "Orders", type: "output", unit: "count", value: 42, metricBinding: { source: "rill_metrics_view", metricsView: "dim_orders_metrics", measure: "metric_total_orders" } },
    ]);

    const resolved = resolveMetricTreeValues(t, {});

    expect(resolved?.nodeResolutions.orders).toMatchObject({
      value: 42,
      sourceStatus: "error",
      sourceError: "Live binding was not resolved",
    });
  });
});
