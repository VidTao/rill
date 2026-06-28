import type {
  AuthoredMetricNode,
  AuthoredMetricTree,
  ExperimentStatus,
  FormulaOp,
  MetricNodeType,
  MetricTreeValidation,
  PropagationResult,
  RollupResult,
} from "./types";

export const NODE_TYPES: MetricNodeType[] = [
  "output",
  "revenue_stream",
  "driver",
  "lever",
  "surface",
  "experiment",
];

export const EXPERIMENT_STATUSES: ExperimentStatus[] = [
  "backlog",
  "do_now",
  "running",
  "won",
  "lost",
  "shipped",
];

export const FORMULA_OPS: FormulaOp[] = ["add", "multiply", "subtract", "ratio"];

export const LEGAL_CHILDREN: Record<MetricNodeType, MetricNodeType[]> = {
  output: ["revenue_stream", "driver", "lever"],
  revenue_stream: ["driver", "lever"],
  driver: ["driver", "lever"],
  lever: ["surface", "driver", "experiment"],
  surface: ["experiment"],
  experiment: [],
};

export function nodeTypeLabel(type: MetricNodeType): string {
  return type.replace("_", " ");
}

export function childrenOf(nodes: AuthoredMetricNode[], id: string): AuthoredMetricNode[] {
  return nodes
    .filter((n) => n.parentId === id)
    .sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0) || a.label.localeCompare(b.label));
}

export function validateTree(nodes: AuthoredMetricNode[]): MetricTreeValidation {
  const errors: MetricTreeValidation["errors"] = [];
  const warnings: MetricTreeValidation["warnings"] = [];
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const roots = nodes.filter((n) => !n.parentId);

  if (roots.length !== 1 || roots[0]?.type !== "output") {
    errors.push({ code: "root", message: "A tree must have exactly one output root." });
  }

  for (const node of nodes) {
    if (!NODE_TYPES.includes(node.type)) {
      errors.push({ code: "node_type", node_id: node.id, message: `Invalid node type ${node.type}.` });
    }
    const parent = node.parentId ? byId.get(node.parentId) : null;
    if (node.parentId && !parent) {
      errors.push({ code: "missing_parent", node_id: node.id, message: "Parent node does not exist." });
    }
    if (parent && !LEGAL_CHILDREN[parent.type].includes(node.type)) {
      errors.push({ code: "illegal_child", node_id: node.id, message: `${node.type} cannot be child of ${parent.type}.` });
    }
    const kids = childrenOf(nodes, node.id);
    if (node.type === "experiment" && kids.length) {
      errors.push({ code: "experiment_leaf", node_id: node.id, message: "Experiments must be leaves." });
    }
    if ((node.type === "surface" || node.type === "experiment") && node.formula) {
      errors.push({ code: "formula_forbidden", node_id: node.id, message: "Surface and experiment nodes cannot have formulas." });
    }
    if (node.formula) {
      if (!FORMULA_OPS.includes(node.formula.op as FormulaOp)) {
        errors.push({ code: "formula_op", node_id: node.id, message: "Unsupported formula op for v1." });
      }
      const kidIds = new Set(kids.map((k) => k.id));
      for (const operand of node.formula.operands) {
        if (!kidIds.has(operand)) {
          errors.push({ code: "formula_operand", node_id: node.id, message: `${operand} is not a child operand.` });
        }
      }
    } else if (kids.length && ["output", "revenue_stream", "driver", "lever"].includes(node.type)) {
      warnings.push({ code: "missing_formula", node_id: node.id, message: "Measurement node has children but no formula." });
    }
    if (node.type === "lever" && (!node.owner || node.baseline == null || node.target == null || !node.targetDate)) {
      errors.push({ code: "lever_complete", node_id: node.id, message: "Lever requires owner, baseline, target, and target date." });
    }
    if (node.type === "experiment") {
      const ice = node.ice ?? {};
      if (!node.hypothesis || !node.status) {
        errors.push({ code: "experiment_complete", node_id: node.id, message: "Experiment requires hypothesis and status." });
      }
      for (const key of ["impact", "confidence", "ease"] as const) {
        const v = ice[key];
        if (!Number.isInteger(v) || (v ?? 0) < 1 || (v ?? 0) > 10) {
          errors.push({ code: "ice", node_id: node.id, message: "ICE values must be integers from 1 to 10." });
        }
      }
    }

    const seen = new Set<string>();
    let cursor: AuthoredMetricNode | undefined = node;
    while (cursor?.parentId) {
      if (seen.has(cursor.parentId)) {
        errors.push({ code: "cycle", node_id: node.id, message: "Cycle detected." });
        break;
      }
      seen.add(cursor.parentId);
      cursor = byId.get(cursor.parentId);
    }
  }

  return { errors, warnings };
}

function evaluate(op: FormulaOp, values: Array<number | null>): number | null {
  if (values.some((v) => v == null || !Number.isFinite(v))) return null;
  const nums = values as number[];
  if (op === "add") return nums.reduce((a, b) => a + b, 0);
  if (op === "multiply") return nums.reduce((a, b) => a * b, 1);
  if (op === "subtract") return nums.slice(1).reduce((a, b) => a - b, nums[0] ?? 0);
  if (op === "ratio") return nums[1] === 0 ? null : nums[0] / nums[1];
  return null;
}

export function rollupTree(tree: AuthoredMetricTree, overrides: Record<string, number> = {}): RollupResult {
  const nodes = tree.nodes;
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const values: Record<string, number | null> = {};
  const drift: Record<string, number> = {};

  function valueFor(id: string): number | null {
    if (id in values) return values[id];
    if (id in overrides) {
      values[id] = overrides[id];
      return values[id];
    }
    const node = byId.get(id);
    if (!node) return null;
    if (!node.formula || node.type === "surface" || node.type === "experiment") {
      values[id] = node.value ?? null;
      return values[id];
    }
    const computed = evaluate(node.formula.op as FormulaOp, node.formula.operands.map(valueFor));
    values[id] = computed;
    if (computed != null && node.value != null) drift[id] = computed - node.value;
    return values[id];
  }

  for (const node of nodes) valueFor(node.id);
  return { values, drift };
}

export function propagate(tree: AuthoredMetricTree, nodeId: string, newValue: number): PropagationResult {
  const byId = new Map(tree.nodes.map((n) => [n.id, n]));
  const current = rollupTree(tree).values;
  const projected = rollupTree(tree, { [nodeId]: newValue }).values;
  const delta: Record<string, number> = {};
  const path: string[] = [];
  let cursor = byId.get(nodeId);
  while (cursor) {
    path.push(cursor.id);
    const before = current[cursor.id];
    const after = projected[cursor.id];
    if (before != null && after != null) delta[cursor.id] = after - before;
    cursor = cursor.parentId ? byId.get(cursor.parentId) : undefined;
  }
  return { projected, delta, path };
}

export interface LeverRootImpactRank {
  node: AuthoredMetricNode;
  nodeId: string;
  label: string;
  rootId: string | null;
  rootImpact: number | null;
  absoluteRootImpact: number | null;
}

export interface ExperimentRootImpactRank {
  node: AuthoredMetricNode;
  nodeId: string;
  label: string;
  rootId: string | null;
  rootImpact: number | null;
  absoluteRootImpact: number | null;
  unsized: boolean;
  unsizedReason: "no_ancestor_lever" | "no_opportunity" | "missing_ice" | null;
  ancestorLeverId: string | null;
}

export interface ReviewWalkStep {
  node: AuthoredMetricNode;
  nodeId: string;
  label: string;
  delta: number | null;
  depth: number;
}

export interface ReviewWalk {
  rootId: string | null;
  steps: ReviewWalkStep[];
  stopReason: "missing_root" | "lever" | "leaf" | "missing_deltas" | "no_child_movement";
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function rootNode(tree: AuthoredMetricTree, rootId?: string): AuthoredMetricNode | null {
  if (rootId) return tree.nodes.find((node) => node.id === rootId) ?? null;
  if (tree.root_node_id) return tree.nodes.find((node) => node.id === tree.root_node_id) ?? null;
  return tree.nodes.find((node) => !node.parentId && node.type === "output") ?? null;
}

function compareImpactRanks(
  a: { absoluteRootImpact: number | null; label: string; nodeId: string },
  b: { absoluteRootImpact: number | null; label: string; nodeId: string },
): number {
  const aImpact = a.absoluteRootImpact;
  const bImpact = b.absoluteRootImpact;
  if (aImpact != null && bImpact == null) return -1;
  if (aImpact == null && bImpact != null) return 1;
  if (aImpact != null && bImpact != null && bImpact !== aImpact) return bImpact - aImpact;
  return a.label.localeCompare(b.label) || a.nodeId.localeCompare(b.nodeId);
}

export function rootImpactForLever(tree: AuthoredMetricTree, leverId: string): number | null {
  const node = tree.nodes.find((n) => n.id === leverId);
  const root = rootNode(tree);
  if (!node || node.type !== "lever" || !root || !isFiniteNumber(node.target)) return null;
  const rootDelta = propagate(tree, leverId, node.target).delta[root.id];
  return isFiniteNumber(rootDelta) ? rootDelta : null;
}

export function rankLeversByRootImpact(tree: AuthoredMetricTree): LeverRootImpactRank[] {
  const root = rootNode(tree);
  return tree.nodes
    .filter((node) => node.type === "lever")
    .map((node) => {
      const rootImpact = rootImpactForLever(tree, node.id);
      return {
        node,
        nodeId: node.id,
        label: node.label,
        rootId: root?.id ?? null,
        rootImpact,
        absoluteRootImpact: rootImpact == null ? null : Math.abs(rootImpact),
      };
    })
    .sort(compareImpactRanks);
}

function nearestAncestorLever(tree: AuthoredMetricTree, node: AuthoredMetricNode): AuthoredMetricNode | null {
  const byId = new Map(tree.nodes.map((n) => [n.id, n]));
  let cursor = node.parentId ? byId.get(node.parentId) : undefined;
  while (cursor) {
    if (cursor.type === "lever") return cursor;
    cursor = cursor.parentId ? byId.get(cursor.parentId) : undefined;
  }
  return null;
}

export function rankExperimentsByRootImpact(tree: AuthoredMetricTree): ExperimentRootImpactRank[] {
  const root = rootNode(tree);
  return tree.nodes
    .filter((node) => node.type === "experiment")
    .map((node) => {
      const ancestorLever = nearestAncestorLever(tree, node);
      const leverOpportunity = ancestorLever ? rootImpactForLever(tree, ancestorLever.id) : null;
      const impact = node.ice?.impact;
      const confidence = node.ice?.confidence;
      let rootImpact: number | null = null;
      let unsizedReason: ExperimentRootImpactRank["unsizedReason"] = null;

      if (!ancestorLever) {
        unsizedReason = "no_ancestor_lever";
      } else if (!isFiniteNumber(leverOpportunity)) {
        unsizedReason = "no_opportunity";
      } else if (!isFiniteNumber(impact) || !isFiniteNumber(confidence)) {
        unsizedReason = "missing_ice";
      } else {
        rootImpact = leverOpportunity * (impact / 10) * (confidence / 10);
      }

      return {
        node,
        nodeId: node.id,
        label: node.label,
        rootId: root?.id ?? null,
        rootImpact,
        absoluteRootImpact: rootImpact == null ? null : Math.abs(rootImpact),
        unsized: rootImpact == null,
        unsizedReason,
        ancestorLeverId: ancestorLever?.id ?? null,
      };
    })
    .sort(compareImpactRanks);
}

export function buildReviewWalk(tree: AuthoredMetricTree, rootId?: string): ReviewWalk {
  const root = rootNode(tree, rootId);
  if (!root) return { rootId: rootId ?? tree.root_node_id ?? null, steps: [], stopReason: "missing_root" };

  const steps: ReviewWalkStep[] = [];
  let cursor: AuthoredMetricNode = root;

  while (true) {
    steps.push({
      node: cursor,
      nodeId: cursor.id,
      label: cursor.label,
      delta: isFiniteNumber(cursor.delta) ? cursor.delta : null,
      depth: steps.length,
    });

    if (cursor.type === "lever") return { rootId: root.id, steps, stopReason: "lever" };

    const children = childrenOf(tree.nodes, cursor.id);
    if (!children.length) return { rootId: root.id, steps, stopReason: "leaf" };
    if (!children.every((child) => isFiniteNumber(child.delta))) {
      return { rootId: root.id, steps, stopReason: "missing_deltas" };
    }

    const [next] = [...children].sort((a, b) => {
      const impactDiff = Math.abs(b.delta!) - Math.abs(a.delta!);
      return impactDiff || (a.sortOrder ?? 0) - (b.sortOrder ?? 0) || a.label.localeCompare(b.label);
    });
    if (!next || Math.abs(next.delta!) === 0) {
      return { rootId: root.id, steps, stopReason: "no_child_movement" };
    }
    cursor = next;
  }
}

export function iceScore(node: AuthoredMetricNode): number | null {
  const ice = node.ice ?? {};
  const vals = [ice.impact, ice.confidence, ice.ease];
  if (!vals.every((v) => typeof v === "number")) return null;
  return Math.round(((vals[0]! + vals[1]! + vals[2]!) / 3) * 10) / 10;
}

export function isHealthy(node: AuthoredMetricNode): boolean | null {
  if (node.delta == null) return null;
  return node.goodDirection === "down" ? node.delta <= 0 : node.delta >= 0;
}

export function formatValue(value: number | null | undefined, unit: string | null | undefined): string {
  if (value == null || Number.isNaN(value)) return "-";
  if (unit === "currency") {
    return value.toLocaleString(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 0 });
  }
  if (unit === "percent") return `${(value * 100).toLocaleString(undefined, { maximumFractionDigits: 2 })}%`;
  if (unit === "count") return value.toLocaleString(undefined, { maximumFractionDigits: 0 });
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}
