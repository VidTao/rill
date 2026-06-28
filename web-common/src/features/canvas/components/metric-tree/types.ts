// The render contract: what `buildMetricTree` produces and every render file
// (graph, node, detail panel) consumes. Kept separate from the spec/config so
// the presentation layer never touches column-mapping concerns.

import type { MetricTreeNodeType } from "./node-types";
import type { MetricTreeStatus } from "./status";

export interface MetricTreeNodeData {
  // Index signature required by @xyflow/svelte's Node<T extends Record<...>>.
  [key: string]: unknown;
  id: string;
  parentId: string | null;
  label: string;
  nodeType: MetricTreeNodeType;
  depth: number;
  value: number | string | null;
  unit: string | null;
  // Optional secondary metrics shown on the node face (e.g. # people, rev/person).
  value2: number | string | null;
  value2Unit: string | null;
  value2Label: string | null;
  value3: number | string | null;
  value3Unit: string | null;
  value3Label: string | null;
  delta: number | null;
  deltaPercent: number | null;
  previousValue: number | string | null;
  status: MetricTreeStatus | string | null;
  sourceLabel: string | null;
  sourceStatus: "live" | "computed" | "fallback" | "unbound" | "error" | null;
  sourceContext: string | null;
  formulaText: string | null;
  computedValue: number | null;
  previousComputedValue: number | null;
  measuredValue: number | null;
  previousMeasuredValue: number | null;
  driftValue: number | null;
  driftPercent: number | null;
  sourceError: string | null;
  sortOrder: number | null;
  confidence: string | null;
  limitation: string | null;
  observation: string | null;
  suggestedTest: string | null;
  successMetric: string | null;
  driverType: string | null;
  attributionModel: string | null;
  comparisonModel: string | null;
  segment: string | null;
  recommendedAction: string | null;
  evidenceLabel: string | null;
  evidenceMetric: string | null;
  logFilterKey: string | null;
  logFilterValue: string | null;
  personaFilterKey: string | null;
  personaFilterValue: string | null;
}

export type MetricTreeDetailRow = Record<string, unknown>;

export interface MetricTreeDetailData {
  relationships: MetricTreeDetailRow[];
  evidence: MetricTreeDetailRow[];
  personas: MetricTreeDetailRow[];
  logs: MetricTreeDetailRow[];
  loading: boolean;
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
