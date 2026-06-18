// The metric_tree config maps graph roles to metrics-view columns. Every role
// has a convention default (matching the Bratrax-generated metric_tree_nodes
// shape), so a minimal `{ metrics_view, tree_name }` config works; explicit
// per-role overrides are honored for hand-authored views.

import type {
  ComponentCommonProperties,
  ComponentFilterProperties,
} from "../types";

export type TreeOrientation = "TB" | "LR";

export interface MetricTreeSpec
  extends ComponentCommonProperties,
    ComponentFilterProperties {
  metrics_view: string;

  // --- tree selection ---
  // Column that names which tree each row belongs to (default "tree_name").
  tree_column?: string;
  // Which tree to display (a VALUE filtered on tree_column). Omit for a
  // single-tree view.
  tree_name?: string;

  // --- layout ---
  orientation?: TreeOrientation;

  // --- structure (required for render) ---
  node_id?: string;
  parent_node_id?: string;
  label?: string;

  // --- value / display (recommended) ---
  value?: string;
  unit?: string;
  delta_value?: string;
  status?: string;
  sort_order?: string;

  // --- detail panel (optional) ---
  confidence?: string;
  limitation?: string;
  observation?: string;
  suggested_test?: string;
  success_metric?: string;
}

// Roles that resolve to a metrics-view column (excludes tree_name [a value] and
// orientation [a layout enum]). Values are the convention-default column names.
export const METRIC_TREE_DEFAULT_COLUMNS = {
  tree_column: "tree_name",
  node_id: "node_id",
  parent_node_id: "parent_node_id",
  label: "label",
  value: "metric_value",
  unit: "metric_unit",
  delta_value: "delta_value",
  status: "status",
  sort_order: "sort_order",
  confidence: "confidence",
  limitation: "limitation",
  observation: "observation",
  suggested_test: "suggested_test",
  success_metric: "success_metric",
} as const;

export type MetricTreeRole = keyof typeof METRIC_TREE_DEFAULT_COLUMNS;
export type ResolvedColumns = Record<MetricTreeRole, string>;

// Roles whose column is a numeric measure (the rest are dimensions / text).
// Used for sidebar picker types + newComponentSpec auto-wiring; the live query
// re-classifies against the metrics view so either modeling still works.
export const MEASURE_ROLES: ReadonlySet<MetricTreeRole> = new Set([
  "value",
  "delta_value",
]);

// Structural roles that MUST resolve to a column for the tree to render.
export const REQUIRED_ROLES = [
  "node_id",
  "parent_node_id",
  "label",
] as const satisfies readonly MetricTreeRole[];

export function resolveTreeColumns(spec: MetricTreeSpec): ResolvedColumns {
  const out = {} as ResolvedColumns;
  for (const role of Object.keys(
    METRIC_TREE_DEFAULT_COLUMNS,
  ) as MetricTreeRole[]) {
    out[role] = spec[role] ?? METRIC_TREE_DEFAULT_COLUMNS[role];
  }
  return out;
}
