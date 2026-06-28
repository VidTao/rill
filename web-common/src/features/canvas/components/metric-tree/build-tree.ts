// Pure, framework-free flat-rows -> tree transform. Each row is one node; nodes
// link via node_id / parent_node_id. Defensive against the messy realities of a
// data-driven view: multiple roots, orphans (parent ref missing), cycles,
// duplicate / null ids, and sibling ordering via sort_order.

import type { ResolvedColumns } from "./columns";
import { isMetricTreeNodeType, type MetricTreeNodeType } from "./node-types";
import type {
  MetricTree,
  MetricTreeEdgeData,
  MetricTreeNodeData,
} from "./types";

type Row = Record<string, unknown>;

function toNum(v: unknown): number | null {
  if (v === null || v === undefined || v === "") return null;
  const n = typeof v === "number" ? v : Number(v);
  return Number.isFinite(n) ? n : null;
}

function toStr(v: unknown): string | null {
  if (v === null || v === undefined) return null;
  if (typeof v === "string") return v === "" ? null : v;
  if (
    typeof v === "number" ||
    typeof v === "boolean" ||
    typeof v === "bigint"
  ) {
    return String(v);
  }
  // Objects / arrays are not valid node fields; ignore them.
  return null;
}

function normalizeTreeName(v: string): string {
  return v.trim().toLocaleLowerCase();
}

function nodeTypeForRow(row: Row, cols: ResolvedColumns): MetricTreeNodeType {
  const explicit = toStr(row[cols.driver_type]);
  if (isMetricTreeNodeType(explicit)) return explicit;
  return toStr(row[cols.parent_node_id]) == null ? "output" : "driver";
}

export function buildMetricTree(
  rows: Row[],
  cols: ResolvedColumns,
  opts: {
    tree?: string;
    value2Unit?: string | null;
    value2Label?: string | null;
    value3Unit?: string | null;
    value3Label?: string | null;
  } = {},
): MetricTree {
  const warnings: string[] = [];

  // 1. Tree selection. The server `where` already scopes to one tree, but the
  // builder filters defensively in case it was queried without the filter.
  let scoped = rows;
  if (opts.tree != null && opts.tree !== "") {
    const exact = rows.filter((r) => toStr(r[cols.tree_column]) === opts.tree);
    if (exact.length > 0) {
      scoped = exact;
    } else {
      const wantedTree = normalizeTreeName(opts.tree);
      scoped = rows.filter((r) => {
        const rowTree = toStr(r[cols.tree_column]);
        return rowTree != null && normalizeTreeName(rowTree) === wantedTree;
      });
    }
  }

  // 2. Flat node map. Drop rows with no id; first occurrence of a dup id wins.
  const byId = new Map<string, MetricTreeNodeData>();
  for (const r of scoped) {
    const id = toStr(r[cols.node_id]);
    if (id == null) {
      warnings.push("Row dropped: missing node_id");
      continue;
    }
    if (byId.has(id)) {
      warnings.push(`Duplicate node_id "${id}" ignored`);
      continue;
    }
    byId.set(id, {
      id,
      parentId: toStr(r[cols.parent_node_id]),
      label: toStr(r[cols.label]) ?? id,
      nodeType: nodeTypeForRow(r, cols),
      depth: 0,
      value: (r[cols.value] as number | string | null | undefined) ?? null,
      unit: toStr(r[cols.unit]),
      value2: (r[cols.value2] as number | string | null | undefined) ?? null,
      value2Unit: toStr(r[cols.value2_unit]) ?? opts.value2Unit ?? null,
      value2Label: toStr(r[cols.value2_label]) ?? opts.value2Label ?? null,
      value3: (r[cols.value3] as number | string | null | undefined) ?? null,
      value3Unit: toStr(r[cols.value3_unit]) ?? opts.value3Unit ?? null,
      value3Label: toStr(r[cols.value3_label]) ?? opts.value3Label ?? null,
      delta: toNum(r[cols.delta_value]),
      deltaPercent: null,
      previousValue: null,
      status: toStr(r[cols.status]),
      sourceLabel: null,
      sourceStatus: null,
      sourceContext: null,
      formulaText: null,
      computedValue: null,
      previousComputedValue: null,
      measuredValue: null,
      previousMeasuredValue: null,
      driftValue: null,
      driftPercent: null,
      sourceError: null,
      sortOrder: toNum(r[cols.sort_order]),
      confidence: toStr(r[cols.confidence]),
      limitation: toStr(r[cols.limitation]),
      observation: toStr(r[cols.observation]),
      suggestedTest: toStr(r[cols.suggested_test]),
      successMetric: toStr(r[cols.success_metric]),
      driverType: toStr(r[cols.driver_type]),
      attributionModel: toStr(r[cols.attribution_model]),
      comparisonModel: toStr(r[cols.comparison_model]),
      segment: toStr(r[cols.segment]),
      recommendedAction: toStr(r[cols.recommended_action]),
      evidenceLabel: toStr(r[cols.evidence_label]),
      evidenceMetric: toStr(r[cols.evidence_metric]),
      logFilterKey: toStr(r[cols.log_filter_key]),
      logFilterValue: toStr(r[cols.log_filter_value]),
      personaFilterKey: toStr(r[cols.persona_filter_key]),
      personaFilterValue: toStr(r[cols.persona_filter_value]),
    });
  }

  // 3. Resolve parents -> roots / children. A parent ref to a missing node is
  // an orphan: promote it to a root so it still renders.
  const rootIds: string[] = [];
  const children = new Map<string, string[]>();
  for (const node of byId.values()) {
    if (node.parentId == null) {
      rootIds.push(node.id);
      continue;
    }
    if (!byId.has(node.parentId)) {
      warnings.push(
        `Node "${node.id}" references missing parent "${node.parentId}"; treated as root`,
      );
      node.parentId = null;
      rootIds.push(node.id);
      continue;
    }
    const kids = children.get(node.parentId);
    if (kids) kids.push(node.id);
    else children.set(node.parentId, [node.id]);
  }

  // Sibling order: sort_order asc (nulls last), then label, then id (stable).
  const sortSibs = (ids: string[]): string[] =>
    [...ids].sort((a, b) => {
      const na = byId.get(a)!;
      const nb = byId.get(b)!;
      const sa = na.sortOrder;
      const sb = nb.sortOrder;
      if (sa != null && sb != null && sa !== sb) return sa - sb;
      if ((sa == null) !== (sb == null)) return sa == null ? 1 : -1;
      return na.label.localeCompare(nb.label) || na.id.localeCompare(nb.id);
    });

  // 4. DFS from each root, assigning depth and emitting parent->child edges.
  // The visited set breaks cycles (a back-edge to an already-visited node is
  // skipped) and prevents infinite recursion.
  const visited = new Set<string>();
  const ordered: MetricTreeNodeData[] = [];
  const edges: MetricTreeEdgeData[] = [];

  const walk = (id: string, depth: number) => {
    visited.add(id);
    const node = byId.get(id)!;
    node.depth = depth;
    ordered.push(node);
    for (const childId of sortSibs(children.get(id) ?? [])) {
      if (visited.has(childId)) {
        warnings.push(`Cycle detected at node "${childId}"; edge skipped`);
        continue;
      }
      edges.push({ id: `${id}->${childId}`, source: id, target: childId });
      walk(childId, depth + 1);
    }
  };
  for (const rootId of sortSibs(rootIds)) {
    if (!visited.has(rootId)) walk(rootId, 0);
  }

  // 5. Any node never reached belongs to a pure cycle with no root entry.
  // Surface it as a root rather than silently dropping it.
  for (const node of byId.values()) {
    if (!visited.has(node.id)) {
      warnings.push(
        `Node "${node.id}" unreachable from any root (cycle); treated as root`,
      );
      node.parentId = null;
      node.depth = 0;
      visited.add(node.id);
      ordered.push(node);
    }
  }

  return {
    nodes: ordered,
    edges,
    rootIds: ordered.filter((n) => n.parentId == null).map((n) => n.id),
    warnings,
    isEmpty: ordered.length === 0,
  };
}
