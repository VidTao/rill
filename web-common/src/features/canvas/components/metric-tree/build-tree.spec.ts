import { describe, expect, it } from "vitest";
import { buildMetricTree } from "./build-tree";
import {
  METRIC_TREE_DEFAULT_COLUMNS,
  resolveTreeColumns,
  type MetricTreeSpec,
} from "./columns";

const COLS = resolveTreeColumns({ metrics_view: "mv" });

// Helper: build a row using the convention column names.
function row(fields: Record<string, unknown>): Record<string, unknown> {
  return fields;
}

describe("buildMetricTree", () => {
  it("builds a linear 3-level tree with correct depth and edges", () => {
    const rows = [
      row({ node_id: "profit", parent_node_id: "", label: "Profit" }),
      row({ node_id: "revenue", parent_node_id: "profit", label: "Revenue" }),
      row({ node_id: "orders", parent_node_id: "revenue", label: "Orders" }),
    ];
    const tree = buildMetricTree(rows, COLS);

    expect(tree.isEmpty).toBe(false);
    expect(tree.rootIds).toEqual(["profit"]);
    expect(tree.nodes.map((n) => [n.id, n.depth])).toEqual([
      ["profit", 0],
      ["revenue", 1],
      ["orders", 2],
    ]);
    expect(tree.edges).toEqual([
      { id: "profit->revenue", source: "profit", target: "revenue" },
      { id: "revenue->orders", source: "revenue", target: "orders" },
    ]);
    expect(tree.warnings).toEqual([]);
  });

  it("supports multiple roots ordered by sort_order", () => {
    const rows = [
      row({ node_id: "b", parent_node_id: "", label: "B", sort_order: 2 }),
      row({ node_id: "a", parent_node_id: "", label: "A", sort_order: 1 }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.rootIds).toEqual(["a", "b"]);
    expect(tree.nodes.every((n) => n.depth === 0)).toBe(true);
  });

  it("promotes an orphan (missing parent) to a root with a warning", () => {
    const rows = [
      row({ node_id: "child", parent_node_id: "ghost", label: "Child" }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.rootIds).toEqual(["child"]);
    expect(tree.nodes[0].parentId).toBeNull();
    expect(tree.warnings.some((w) => w.includes("ghost"))).toBe(true);
  });

  it("breaks a simple cycle without infinite recursion", () => {
    const rows = [
      row({ node_id: "a", parent_node_id: "b", label: "A" }),
      row({ node_id: "b", parent_node_id: "a", label: "B" }),
    ];
    const tree = buildMetricTree(rows, COLS);
    // Every node appears exactly once.
    expect(tree.nodes.map((n) => n.id).sort()).toEqual(["a", "b"]);
    expect(tree.warnings.some((w) => w.toLowerCase().includes("cycle"))).toBe(
      true,
    );
  });

  it("surfaces a pure cycle (no root) instead of dropping it", () => {
    const rows = [
      row({ node_id: "a", parent_node_id: "c", label: "A" }),
      row({ node_id: "b", parent_node_id: "a", label: "B" }),
      row({ node_id: "c", parent_node_id: "b", label: "C" }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes).toHaveLength(3);
    expect(tree.rootIds.length).toBeGreaterThanOrEqual(1);
  });

  it("orders siblings by sort_order, nulls last, ties by label then id", () => {
    const rows = [
      row({ node_id: "root", parent_node_id: "", label: "Root" }),
      row({ node_id: "c3", parent_node_id: "root", label: "C", sort_order: 3 }),
      row({ node_id: "c1", parent_node_id: "root", label: "A", sort_order: 1 }),
      row({ node_id: "c2", parent_node_id: "root", label: "B", sort_order: 2 }),
      row({ node_id: "cz", parent_node_id: "root", label: "Z" }), // null sort_order
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes.map((n) => n.id)).toEqual([
      "root",
      "c1",
      "c2",
      "c3",
      "cz",
    ]);
  });

  it("falls back to the id when label is empty", () => {
    const rows = [row({ node_id: "n1", parent_node_id: "", label: "" })];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes[0].label).toBe("n1");
  });

  it("drops a row with a missing node_id and warns", () => {
    const rows = [
      row({ node_id: "", parent_node_id: "", label: "ghost" }),
      row({ node_id: "real", parent_node_id: "", label: "Real" }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes.map((n) => n.id)).toEqual(["real"]);
    expect(tree.warnings.some((w) => w.includes("missing node_id"))).toBe(true);
  });

  it("ignores a duplicate node_id (first wins) and warns", () => {
    const rows = [
      row({ node_id: "dup", parent_node_id: "", label: "First" }),
      row({ node_id: "dup", parent_node_id: "", label: "Second" }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes).toHaveLength(1);
    expect(tree.nodes[0].label).toBe("First");
    expect(tree.warnings.some((w) => w.includes("Duplicate"))).toBe(true);
  });

  it("filters to one tree when opts.tree is given", () => {
    const rows = [
      row({
        tree_name: "Profit",
        node_id: "p",
        parent_node_id: "",
        label: "P",
      }),
      row({
        tree_name: "Retention",
        node_id: "r",
        parent_node_id: "",
        label: "R",
      }),
    ];
    const profit = buildMetricTree(rows, COLS, { tree: "Profit" });
    expect(profit.nodes.map((n) => n.id)).toEqual(["p"]);

    const none = buildMetricTree(rows, COLS, { tree: "Nope" });
    expect(none.isEmpty).toBe(true);
  });

  it("returns an empty tree for empty input without throwing", () => {
    const tree = buildMetricTree([], COLS);
    expect(tree).toMatchObject({
      nodes: [],
      edges: [],
      rootIds: [],
      isEmpty: true,
    });
  });

  it("produces the same shape via convention defaults and explicit overrides", () => {
    const conventionRows = [
      row({ node_id: "a", parent_node_id: "", label: "A" }),
      row({ node_id: "b", parent_node_id: "a", label: "B" }),
    ];
    const conventionTree = buildMetricTree(conventionRows, COLS);

    const explicitSpec: MetricTreeSpec = {
      metrics_view: "mv",
      node_id: "id",
      parent_node_id: "pid",
      label: "name",
    };
    const explicitCols = resolveTreeColumns(explicitSpec);
    const explicitRows = [
      row({ id: "a", pid: "", name: "A" }),
      row({ id: "b", pid: "a", name: "B" }),
    ];
    const explicitTree = buildMetricTree(explicitRows, explicitCols);

    expect(explicitTree.nodes.map((n) => [n.id, n.parentId, n.label])).toEqual(
      conventionTree.nodes.map((n) => [n.id, n.parentId, n.label]),
    );
  });

  it("passes optional narrative + value fields through to node data", () => {
    const rows = [
      row({
        node_id: "disc",
        parent_node_id: "",
        label: "Discount depth",
        metric_value: 36.8,
        metric_unit: "percent",
        delta_value: 4.2,
        status: "warning",
        confidence: "high",
        limitation: "Excludes full contribution margin.",
        observation: "Discounts are taking a large share of gross revenue.",
        suggested_test: "Test lower discount plus bundle or gift offer.",
        success_metric: "Net revenue per order and conversion rate.",
        sort_order: 2,
      }),
    ];
    const tree = buildMetricTree(rows, COLS);
    expect(tree.nodes[0]).toMatchObject({
      value: 36.8,
      unit: "percent",
      delta: 4.2,
      status: "warning",
      confidence: "high",
      observation: "Discounts are taking a large share of gross revenue.",
      suggestedTest: "Test lower discount plus bundle or gift offer.",
      successMetric: "Net revenue per order and conversion rate.",
      sortOrder: 2,
    });
  });
});

describe("resolveTreeColumns", () => {
  it("uses convention defaults when nothing is overridden", () => {
    const cols = resolveTreeColumns({ metrics_view: "mv" });
    expect(cols).toEqual(METRIC_TREE_DEFAULT_COLUMNS);
  });

  it("honors explicit overrides while defaulting the rest", () => {
    const cols = resolveTreeColumns({ metrics_view: "mv", node_id: "id" });
    expect(cols.node_id).toBe("id");
    expect(cols.parent_node_id).toBe("parent_node_id");
  });
});
