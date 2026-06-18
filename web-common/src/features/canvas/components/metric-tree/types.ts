// The render contract: what `buildMetricTree` produces and every render file
// (graph, node, detail panel) consumes. Kept separate from the spec/config so
// the presentation layer never touches column-mapping concerns.

import type { MetricTreeStatus } from "./status";

export interface MetricTreeNodeData {
  // Index signature required by @xyflow/svelte's Node<T extends Record<...>>.
  [key: string]: unknown;
  id: string;
  parentId: string | null;
  label: string;
  depth: number;
  value: number | string | null;
  unit: string | null;
  delta: number | null;
  status: MetricTreeStatus | string | null;
  sortOrder: number | null;
  confidence: string | null;
  limitation: string | null;
  observation: string | null;
  suggestedTest: string | null;
  successMetric: string | null;
}

export interface MetricTreeEdgeData {
  id: string;
  source: string;
  target: string;
}

export interface MetricTree {
  nodes: MetricTreeNodeData[];
  edges: MetricTreeEdgeData[];
  rootIds: string[];
  // Non-blocking issues (orphans, cycles, dropped rows) surfaced as a notice.
  warnings: string[];
  isEmpty: boolean;
}
