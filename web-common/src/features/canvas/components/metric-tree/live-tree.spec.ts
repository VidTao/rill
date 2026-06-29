import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuthoredMetricTree } from "./authored-tree";
import {
  metricBindingKey,
  planLiveMetricRequests,
  resolveMetricTreeLiveValues,
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
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });
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



  it("sanitizes HTML 502 responses from grouped live metric requests", async () => {
    const t = tree([
      { id: "traffic", parentId: null, label: "Traffic", type: "driver", unit: "count", value: 123, metricBinding: { source: "rill_metrics_view", metricsView: "d2c_metric_tree_drivers_metrics", measure: "metric_traffic_visitors" } },
      { id: "conversion", parentId: null, label: "Conversion", type: "lever", unit: "percent", value: 0.04, metricBinding: { source: "rill_metrics_view", metricsView: "d2c_metric_tree_drivers_metrics", measure: "metric_conversion_rate" } },
    ]);
    vi.stubGlobal("fetch", vi.fn(async () => new Response(`<!doctype html><html><body><h1>502 Bad Gateway</h1><center>nginx/1.24.0</center></body></html>`, { status: 502 })));

    const resolved = await resolveMetricTreeLiveValues({
      tree: t,
      instanceId: "paw_origins_llc",
      hasTimeSeries: true,
      timeRange: { isoDuration: "P30D" },
    });

    expect(resolved?.nodeResolutions.traffic).toMatchObject({
      value: 123,
      sourceStatus: "error",
      sourceError: "d2c_metric_tree_drivers_metrics.metric_traffic_visitors, metric_conversion_rate: Metric query failed (502)",
    });
    expect(resolved?.nodeResolutions.conversion.value).toBe(0.04);
    expect(resolved?.liveWarnings).toEqual([
      "d2c_metric_tree_drivers_metrics.metric_traffic_visitors, metric_conversion_rate: Metric query failed (502)",
    ]);
    expect(JSON.stringify(resolved)).not.toMatch(/<html|nginx|Bad Gateway/i);
  });

  it("keeps current live values when only the comparison request fails", async () => {
    const t = tree([
      { id: "traffic", parentId: null, label: "Traffic", type: "driver", unit: "count", value: 123, metricBinding: { source: "rill_metrics_view", metricsView: "d2c_metric_tree_drivers_metrics", measure: "metric_traffic_visitors" } },
    ]);
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: [{ metric_traffic_visitors: 500 }] }), { status: 200, headers: { "content-type": "application/json" } }))
      .mockResolvedValueOnce(new Response("<html><body><h1>502 Bad Gateway</h1></body></html>", { status: 502 })));

    const resolved = await resolveMetricTreeLiveValues({
      tree: t,
      instanceId: "paw_origins_llc",
      hasTimeSeries: true,
      showTimeComparison: true,
      timeRange: { isoDuration: "P30D" },
      comparisonTimeRange: { isoOffset: "P30D" },
    });

    expect(resolved?.nodeResolutions.traffic).toMatchObject({
      value: 500,
      previousValue: null,
      sourceStatus: "live",
      sourceError: "d2c_metric_tree_drivers_metrics.metric_traffic_visitors: Metric query failed (502)",
    });
    expect(resolved?.liveWarnings).toEqual([
      "d2c_metric_tree_drivers_metrics.metric_traffic_visitors: Metric query failed (502)",
    ]);
    expect(JSON.stringify(resolved)).not.toMatch(/<html|Bad Gateway/i);
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
