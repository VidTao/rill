import type { MetricTreeNodeResolution } from "./live-tree";
import { isMetricTreeNodeType } from "./node-types";
import type { MetricTree, MetricTreeNodeData } from "./types";

export interface AuthoredMetricNode {
  id: string;
  parentId: string | null;
  label: string;
  type: string;
  unit: string;
  value: number | null;
  delta?: number | null;
  goodDirection?: "up" | "down";
  formula?: { op: string; operands: string[] } | null;
  owner?: string | null;
  ownerUserId?: number | null;
  ownerLabel?: string | null;
  baseline?: number | null;
  target?: number | null;
  targetDate?: string | null;
  hypothesis?: string | null;
  status?: string | null;
  ice?: { impact?: number; confidence?: number; ease?: number };
  metricBinding?: Record<string, unknown>;
  measurementBinding?: Record<string, unknown>;
  guardrailIds?: string[];
  tags?: string[];
  sortOrder?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface AuthoredMetricTree {
  tree_id: string;
  name: string;
  description?: string;
  root_node_id?: string;
  created_at?: string;
  updated_at?: string;
  nodes: AuthoredMetricNode[];
  nodeResolutions?: Record<string, MetricTreeNodeResolution>;
  liveWarnings?: string[];
}

export async function fetchAuthoredMetricTree(treeId: string): Promise<AuthoredMetricTree> {
  const res = await fetch(`/bratrax/metric-trees/${encodeURIComponent(treeId)}`, {
    credentials: "include",
  });
  if (!res.ok) {
    let message = `Metric tree ${treeId} could not be loaded (${res.status})`;
    try {
      const body = await res.json();
      message = body.error ?? body.message ?? message;
    } catch {
      // Keep default message.
    }
    throw new Error(message);
  }
  const body = await res.json();
  return body.data as AuthoredMetricTree;
}

function statusFor(node: AuthoredMetricNode, delta: number | null | undefined = node.delta): string | null {
  if (node.status) return node.status;
  if (delta == null) return null;
  if (node.goodDirection === "down") return delta <= 0 ? "good" : "bad";
  return delta >= 0 ? "good" : "bad";
}

function nodeObservation(node: AuthoredMetricNode): string | null {
  const parts: string[] = [];
  if (node.formula) parts.push(`Formula: ${node.formula.op}(${node.formula.operands.join(", ")})`);
  if (node.ownerLabel || node.owner) parts.push(`Owner: ${node.ownerLabel || node.owner}`);
  if (node.baseline != null || node.target != null) {
    parts.push(`Baseline ${node.baseline ?? "-"} -> target ${node.target ?? "-"}${node.targetDate ? ` by ${node.targetDate}` : ""}`);
  }
  if (node.hypothesis) parts.push(node.hypothesis);
  return parts.length ? parts.join(". ") : null;
}

function iceText(node: AuthoredMetricNode): string | null {
  const ice = node.ice ?? {};
  if (ice.impact == null && ice.confidence == null && ice.ease == null) return null;
  return `ICE ${ice.impact ?? "-"}/${ice.confidence ?? "-"}/${ice.ease ?? "-"}`;
}

export function authoredTreeToRenderable(tree: AuthoredMetricTree | null): MetricTree {
  if (!tree) return { nodes: [], edges: [], rootIds: [], warnings: [], isEmpty: true };
  const byId = new Map(tree.nodes.map((n) => [n.id, n]));
  const children = new Map<string, AuthoredMetricNode[]>();
  const roots: AuthoredMetricNode[] = [];
  const warnings: string[] = [];

  for (const node of tree.nodes) {
    if (!node.parentId) {
      roots.push(node);
      continue;
    }
    if (!byId.has(node.parentId)) {
      warnings.push(`Node "${node.id}" references missing parent "${node.parentId}"; treated as root`);
      roots.push(node);
      continue;
    }
    const list = children.get(node.parentId) ?? [];
    list.push(node);
    children.set(node.parentId, list);
  }

  const sortNodes = (nodes: AuthoredMetricNode[]) => [...nodes].sort((a, b) => {
    const sa = a.sortOrder ?? Number.MAX_SAFE_INTEGER;
    const sb = b.sortOrder ?? Number.MAX_SAFE_INTEGER;
    return sa - sb || a.label.localeCompare(b.label) || a.id.localeCompare(b.id);
  });

  const out: MetricTreeNodeData[] = [];
  const edges: MetricTree["edges"] = [];
  const visited = new Set<string>();

  function visit(node: AuthoredMetricNode, depth: number) {
    if (visited.has(node.id)) {
      warnings.push(`Cycle detected at node "${node.id}"; duplicate visit skipped`);
      return;
    }
    visited.add(node.id);
    const resolution = tree.nodeResolutions?.[node.id];
    out.push({
      id: node.id,
      parentId: node.parentId && byId.has(node.parentId) ? node.parentId : null,
      label: node.label,
      nodeType: isMetricTreeNodeType(node.type) ? node.type : "driver",
      depth,
      value: resolution?.value ?? node.value ?? null,
      unit: node.unit === "currency" ? "usd" : node.unit,
      value2: node.type === "experiment" ? iceText(node) : null,
      value2Unit: null,
      value2Label: node.type === "experiment" ? "Score" : null,
      value3: null,
      value3Unit: null,
      value3Label: null,
      delta: resolution?.delta ?? node.delta ?? null,
      deltaPercent: resolution?.deltaPercent ?? null,
      previousValue: resolution?.previousValue ?? null,
      status: statusFor(node, resolution?.delta ?? node.delta),
      sourceLabel: resolution?.sourceLabel ?? null,
      sourceStatus: resolution?.sourceStatus ?? null,
      sourceContext: resolution?.sourceContext ?? null,
      formulaText: resolution?.formulaText ?? (node.formula ? `${node.formula.op}(${node.formula.operands.join(", ")})` : null),
      computedValue: resolution?.computedValue ?? null,
      previousComputedValue: resolution?.previousComputedValue ?? null,
      measuredValue: resolution?.measuredValue ?? null,
      previousMeasuredValue: resolution?.previousMeasuredValue ?? null,
      driftValue: resolution?.driftValue ?? null,
      driftPercent: resolution?.driftPercent ?? null,
      sourceError: resolution?.sourceError ?? null,
      sortOrder: node.sortOrder ?? null,
      confidence: node.ice?.confidence != null ? String(node.ice.confidence) : null,
      limitation: null,
      observation: nodeObservation(node),
      suggestedTest: node.hypothesis ?? null,
      successMetric: node.target != null ? `Target: ${node.target}` : null,
      driverType: node.type,
      attributionModel: null,
      comparisonModel: null,
      segment: node.owner ?? null,
      recommendedAction: node.type === "lever" ? "Move this lever toward target." : null,
      evidenceLabel: node.formula ? "Formula" : null,
      evidenceMetric: node.formula ? node.formula.operands.join(", ") : null,
      logFilterKey: null,
      logFilterValue: null,
      personaFilterKey: null,
      personaFilterValue: null,
    });
    for (const child of sortNodes(children.get(node.id) ?? [])) {
      edges.push({ id: `${node.id}->${child.id}`, source: node.id, target: child.id });
      visit(child, depth + 1);
    }
  }

  const orderedRoots = tree.root_node_id && byId.has(tree.root_node_id)
    ? [byId.get(tree.root_node_id)!, ...sortNodes(roots.filter((r) => r.id !== tree.root_node_id))]
    : sortNodes(roots);

  for (const root of orderedRoots) visit(root, 0);
  for (const node of sortNodes(tree.nodes)) {
    if (!visited.has(node.id)) {
      warnings.push(`Node "${node.id}" was unreachable; treated as root`);
      visit({ ...node, parentId: null }, 0);
    }
  }

  return {
    nodes: out,
    edges,
    rootIds: out.filter((n) => !n.parentId).map((n) => n.id),
    warnings: [...warnings, ...(tree.liveWarnings ?? [])],
    isEmpty: out.length === 0,
  };
}
