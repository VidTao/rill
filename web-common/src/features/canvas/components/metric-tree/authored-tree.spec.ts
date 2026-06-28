import { describe, expect, it } from "vitest";
import { authoredTreeToRenderable, type AuthoredMetricTree } from "./authored-tree";

describe("authoredTreeToRenderable", () => {
  it("preserves PRD node types for authored trees", () => {
    const tree: AuthoredMetricTree = {
      tree_id: "starter-d2c-hybrid",
      name: "D2C hybrid",
      root_node_id: "revenue",
      nodes: [
        { id: "revenue", parentId: null, label: "Revenue", type: "output", unit: "currency", value: 100 },
        { id: "stream", parentId: "revenue", label: "Stream", type: "revenue_stream", unit: "currency", value: 70 },
        { id: "driver", parentId: "stream", label: "Driver", type: "driver", unit: "count", value: 10 },
        { id: "lever", parentId: "driver", label: "Lever", type: "lever", unit: "percent", value: 5, owner: "Growth", target: 7 },
        { id: "surface", parentId: "lever", label: "Surface", type: "surface", unit: "text", value: null },
        { id: "experiment", parentId: "surface", label: "Experiment", type: "experiment", unit: "text", value: null, ice: { impact: 8, confidence: 7, ease: 6 }, status: "do_now" },
      ],
    };

    const renderable = authoredTreeToRenderable(tree);

    expect(renderable.nodes.map((n) => [n.id, n.nodeType])).toEqual([
      ["revenue", "output"],
      ["stream", "revenue_stream"],
      ["driver", "driver"],
      ["lever", "lever"],
      ["surface", "surface"],
      ["experiment", "experiment"],
    ]);
    expect(renderable.nodes.find((n) => n.id === "experiment")).toMatchObject({
      value2: "ICE 8/7/6",
      value2Label: "Score",
      status: "do_now",
    });
  });
});
