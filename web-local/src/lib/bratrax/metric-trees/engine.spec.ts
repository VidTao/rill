import { describe, expect, it } from "vitest";
import type { AuthoredMetricNode, AuthoredMetricTree } from "./types";
import {
  buildReviewWalk,
  rankExperimentsByRootImpact,
  rankLeversByRootImpact,
  rootImpactForLever,
} from "./engine";

function node(overrides: Partial<AuthoredMetricNode> & Pick<AuthoredMetricNode, "id" | "label" | "type">): AuthoredMetricNode {
  return {
    parentId: null,
    unit: "count",
    value: 0,
    goodDirection: "up",
    ...overrides,
  };
}

function tree(nodes: AuthoredMetricNode[]): AuthoredMetricTree {
  return {
    tree_id: "tree",
    name: "Tree",
    root_node_id: "root",
    nodes,
  };
}

const impactTree = tree([
  node({
    id: "root",
    label: "Output",
    type: "output",
    value: 100,
    formula: { op: "add", operands: ["driver", "lever-a", "lever-b"] },
  }),
  node({ id: "driver", label: "Driver", type: "driver", parentId: "root", value: 30 }),
  node({
    id: "lever-a",
    label: "Expansion",
    type: "lever",
    parentId: "root",
    value: 40,
    baseline: 40,
    target: 55,
    targetDate: "2026-07-01",
  }),
  node({
    id: "lever-b",
    label: "Churn",
    type: "lever",
    parentId: "root",
    value: 30,
    baseline: 30,
    target: 10,
    targetDate: "2026-07-01",
    goodDirection: "down",
  }),
  node({ id: "surface-a", label: "Landing page", type: "surface", parentId: "lever-a", value: null }),
  node({
    id: "experiment-a",
    label: "Hero test",
    type: "experiment",
    parentId: "surface-a",
    value: null,
    hypothesis: "Better message increases conversion.",
    status: "backlog",
    ice: { impact: 8, confidence: 5, ease: 3 },
  }),
  node({
    id: "experiment-b",
    label: "Unsized test",
    type: "experiment",
    parentId: "driver",
    value: null,
    hypothesis: "Needs a lever.",
    status: "backlog",
    ice: { impact: 9, confidence: 9, ease: 9 },
  }),
]);

describe("review and impact engine", () => {
  it("propagates a lever target to the root exactly", () => {
    expect(rootImpactForLever(impactTree, "lever-a")).toBe(15);
    expect(rootImpactForLever(impactTree, "lever-b")).toBe(-20);
  });

  it("ranks levers by absolute propagated root impact", () => {
    expect(rankLeversByRootImpact(impactTree).map((rank) => [rank.nodeId, rank.rootImpact])).toEqual([
      ["lever-b", -20],
      ["lever-a", 15],
    ]);
  });

  it("estimates experiment impact from the nearest ancestor lever opportunity and ICE sizing", () => {
    const ranks = rankExperimentsByRootImpact(impactTree);

    expect(ranks[0]).toMatchObject({
      nodeId: "experiment-a",
      rootImpact: 6,
      unsized: false,
      unsizedReason: null,
      ancestorLeverId: "lever-a",
    });
    expect(ranks[1]).toMatchObject({
      nodeId: "experiment-b",
      rootImpact: null,
      unsized: true,
      unsizedReason: "no_ancestor_lever",
      ancestorLeverId: null,
    });
  });

  it("walks to the largest absolute child delta and stops at the first lever", () => {
    const reviewTree = tree([
      node({ id: "root", label: "Output", type: "output", delta: 20 }),
      node({ id: "driver", label: "Driver", type: "driver", parentId: "root", delta: 8 }),
      node({ id: "lever", label: "Lever", type: "lever", parentId: "root", delta: -12 }),
      node({ id: "experiment", label: "Experiment", type: "experiment", parentId: "lever", delta: -12 }),
    ]);

    const walk = buildReviewWalk(reviewTree);

    expect(walk.stopReason).toBe("lever");
    expect(walk.steps.map((step) => step.nodeId)).toEqual(["root", "lever"]);
  });

  it("stops when child deltas are missing or have no movement", () => {
    const missing = buildReviewWalk(
      tree([
        node({ id: "root", label: "Output", type: "output", delta: 20 }),
        node({ id: "driver", label: "Driver", type: "driver", parentId: "root" }),
      ]),
    );
    const noMovement = buildReviewWalk(
      tree([
        node({ id: "root", label: "Output", type: "output", delta: 0 }),
        node({ id: "driver", label: "Driver", type: "driver", parentId: "root", delta: 0 }),
      ]),
    );

    expect(missing.stopReason).toBe("missing_deltas");
    expect(noMovement.stopReason).toBe("no_child_movement");
  });
});
