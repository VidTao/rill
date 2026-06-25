// Dagre auto-layout for the metric tree. Top-down (TB) by default — parent
// outcome on top, drivers below — with a left-to-right (LR) option. Node sizing
// constants live here as the single source of truth shared with the custom node
// component (the CSS width/height must match NODE_W/NODE_H).

import { graphlib, layout as dagreLayout } from "@dagrejs/dagre";
import { Position, type Edge, type Node } from "@xyflow/svelte";
import type { TreeOrientation } from "./columns";
import type { MetricTreeEdgeData, MetricTreeNodeData } from "./types";

export const NODE_W = 220;
export const NODE_H = 144;
export const NODE_TYPE = "metric-tree-node";

const NODESEP = 28;
const RANKSEP = 64;
const EDGESEP = 8;

export interface MetricTreeLayout {
  nodes: Node<MetricTreeNodeData>[];
  edges: Edge[];
  width: number;
  height: number;
}

export function buildMetricTreeLayout(
  treeNodes: MetricTreeNodeData[],
  treeEdges: MetricTreeEdgeData[],
  opts: { orientation?: TreeOrientation } = {},
): MetricTreeLayout {
  const orientation = opts.orientation ?? "TB";
  const sourcePosition =
    orientation === "TB" ? Position.Bottom : Position.Right;
  const targetPosition = orientation === "TB" ? Position.Top : Position.Left;

  const g = new graphlib.Graph();
  g.setGraph({
    rankdir: orientation,
    nodesep: NODESEP,
    ranksep: RANKSEP,
    edgesep: EDGESEP,
    ranker: "tight-tree",
    marginx: 12,
    marginy: 12,
  });
  g.setDefaultEdgeLabel(() => ({}));

  // Insertion order drives Dagre's sibling ordering; buildMetricTree already
  // sorted nodes (roots first, siblings by sort_order).
  for (const n of treeNodes) g.setNode(n.id, { width: NODE_W, height: NODE_H });
  for (const e of treeEdges) g.setEdge(e.source, e.target);

  dagreLayout(g);

  const nodes: Node<MetricTreeNodeData>[] = treeNodes.map((n) => {
    const dn = g.node(n.id);
    return {
      id: n.id,
      type: NODE_TYPE,
      position: { x: (dn?.x ?? 0) - NODE_W / 2, y: (dn?.y ?? 0) - NODE_H / 2 },
      width: NODE_W,
      height: NODE_H,
      sourcePosition,
      targetPosition,
      data: n,
    };
  });

  const edges: Edge[] = treeEdges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    type: "smoothstep",
    style: "stroke:#cbd5e1;stroke-width:1.5px",
  }));

  const gg = g.graph();
  return { nodes, edges, width: gg.width ?? 0, height: gg.height ?? 0 };
}
