import {
  createAndExpression,
  createInExpression,
} from "@rilldata/web-common/features/dashboards/stores/filter-utils";
import type { AuthoredMetricTree } from "./authored-tree";

export type MetricTreeSourceStatus =
  | "live"
  | "computed"
  | "fallback"
  | "unbound"
  | "error";

export interface MetricTreeBindingFilter {
  dimension: string;
  op?: "in" | "not_in";
  values: string[];
}

export interface MetricTreeMetricBinding {
  source?: string;
  type?: string;
  metricsView?: string;
  metrics_view?: string;
  measure?: string;
  filters?: MetricTreeBindingFilter[];
  timeMode?: "dashboard" | "all_time" | "snapshot";
  time_mode?: "dashboard" | "all_time" | "snapshot";
  comparison?: "previous_period" | "none";
}

export interface MetricTreeLiveRequest {
  key: string;
  metricsView: string;
  measures: string[];
  filters: MetricTreeBindingFilter[];
  timeMode: "dashboard" | "all_time" | "snapshot";
}

export interface MetricTreeLiveValue {
  value: number | null;
  previousValue?: number | null;
  error?: string | null;
  previousError?: string | null;
}

export interface MetricTreeNodeResolution {
  value: number | null;
  previousValue: number | null;
  delta: number | null;
  deltaPercent: number | null;
  measuredValue: number | null;
  previousMeasuredValue: number | null;
  computedValue: number | null;
  previousComputedValue: number | null;
  driftValue: number | null;
  driftPercent: number | null;
  sourceStatus: MetricTreeSourceStatus;
  sourceLabel: string | null;
  sourceContext: string | null;
  formulaText: string | null;
  sourceError: string | null;
}

export interface MetricTreeResolveContext {
  dashboardTimeLabel?: string | null;
  dashboardFilterLabel?: string | null;
}

export interface ResolvedMetricTree extends AuthoredMetricTree {
  nodeResolutions: Record<string, MetricTreeNodeResolution>;
  liveWarnings: string[];
}

function bindingSource(binding: MetricTreeMetricBinding | undefined): string | null {
  return binding?.source ?? binding?.type ?? null;
}

function bindingMetricsView(binding: MetricTreeMetricBinding | undefined): string | null {
  return binding?.metricsView ?? binding?.metrics_view ?? null;
}

function bindingTimeMode(binding: MetricTreeMetricBinding | undefined): "dashboard" | "all_time" | "snapshot" {
  return binding?.timeMode ?? binding?.time_mode ?? "dashboard";
}

export function metricTreeBindingFilterKey(filters: MetricTreeBindingFilter[] | undefined): string {
  if (!filters || filters.length === 0) return "[]";
  return JSON.stringify(filters.map((f) => ({ dimension: f.dimension, op: f.op ?? "in", values: [...f.values].sort() })).sort((a, b) => a.dimension.localeCompare(b.dimension)));
}

export function metricBindingKey(binding: MetricTreeMetricBinding): string | null {
  const metricsView = bindingMetricsView(binding);
  const measure = binding.measure;
  if (!metricsView || !measure) return null;
  return `${metricsView}::${measure}::${bindingTimeMode(binding)}::${metricTreeBindingFilterKey(binding.filters)}`;
}

export function metricBindingLabel(binding: MetricTreeMetricBinding | undefined): string | null {
  const metricsView = bindingMetricsView(binding);
  if (!metricsView || !binding?.measure) return null;
  return `${metricsView}.${binding.measure}`;
}

export function metricBindingContextLabel(
  binding: MetricTreeMetricBinding | undefined,
  context: MetricTreeResolveContext = {},
): string | null {
  if (!binding) return null;
  const parts: string[] = [];
  const timeMode = bindingTimeMode(binding);
  if (timeMode === "snapshot") parts.push("Current snapshot");
  else if (timeMode === "all_time") parts.push("All time");
  else parts.push(context.dashboardTimeLabel ?? "Selected time range");
  if (binding.filters?.length) {
    parts.push(`${binding.filters.length} node filter${binding.filters.length === 1 ? "" : "s"}`);
  }
  if (context.dashboardFilterLabel) parts.push(context.dashboardFilterLabel);
  return parts.join(" · ");
}

export function metricRequestMeasureKey(req: MetricTreeLiveRequest, measure: string): string {
  return `${req.metricsView}::${measure}::${req.timeMode}::${metricTreeBindingFilterKey(req.filters)}`;
}

export function planLiveMetricRequests(tree: AuthoredMetricTree | null): MetricTreeLiveRequest[] {
  if (!tree) return [];
  const groups = new Map<string, MetricTreeLiveRequest>();
  for (const node of tree.nodes) {
    const binding = node.metricBinding as MetricTreeMetricBinding | undefined;
    const source = bindingSource(binding);
    if (source !== "rill_metrics_view" || !binding?.measure) continue;
    const metricsView = bindingMetricsView(binding);
    if (!metricsView) continue;
    const filters = binding.filters ?? [];
    const timeMode = bindingTimeMode(binding);
    const key = `${metricsView}::${timeMode}::${metricTreeBindingFilterKey(filters)}`;
    const group = groups.get(key) ?? { key, metricsView, measures: [], filters, timeMode };
    if (!group.measures.includes(binding.measure)) group.measures.push(binding.measure);
    groups.set(key, group);
  }
  return [...groups.values()];
}

function toNumber(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : null;
}

function evalFormula(op: string, values: number[]): number | null {
  if (values.some((v) => !Number.isFinite(v))) return null;
  if (op === "add") return values.reduce((a, b) => a + b, 0);
  if (op === "multiply") return values.reduce((a, b) => a * b, 1);
  if (op === "subtract") return values.length ? values[0] - values.slice(1).reduce((a, b) => a + b, 0) : null;
  if (op === "ratio") return values.length >= 2 && values[1] !== 0 ? values[0] / values[1] : null;
  return null;
}

function formulaText(formula: { op: string; operands: string[] } | null | undefined): string | null {
  if (!formula) return null;
  return `${formula.op}(${formula.operands.join(", ")})`;
}

function deltaFor(value: number | null, previousValue: number | null): { delta: number | null; deltaPercent: number | null } {
  if (value == null || previousValue == null) return { delta: null, deltaPercent: null };
  const delta = value - previousValue;
  return { delta, deltaPercent: previousValue !== 0 ? delta / previousValue : null };
}


export interface ResolveMetricTreeLiveOptions {
  tree: AuthoredMetricTree | null;
  instanceId: string;
  timeRange?: unknown;
  comparisonTimeRange?: unknown;
  where?: unknown;
  hasTimeSeries?: boolean;
  showTimeComparison?: boolean;
  dashboardTimeLabel?: string | null;
  dashboardFilterLabel?: string | null;
  priority?: number;
}

function whereForLiveRequest(req: MetricTreeLiveRequest, inheritedWhere?: unknown) {
  const clauses = req.filters
    .filter((filter) => filter.op == null || filter.op === "in")
    .map((filter) => createInExpression(filter.dimension, filter.values));
  if (inheritedWhere) clauses.push(inheritedWhere);
  if (clauses.length === 0) return undefined;
  return clauses.length === 1 ? clauses[0] : createAndExpression(clauses);
}

async function runAggregationRequest(
  instanceId: string,
  req: MetricTreeLiveRequest,
  requestTimeRange: unknown,
  inheritedWhere: unknown,
  hasTimeSeries: boolean,
  priority: number,
): Promise<{ row: Record<string, unknown>; error: string | null }> {
  const body: Record<string, unknown> = {
    dimensions: [],
    measures: req.measures.map((name) => ({ name })),
    where: whereForLiveRequest(req, inheritedWhere),
    limit: "1",
    priority,
  };
  if (req.timeMode === "dashboard" && hasTimeSeries && requestTimeRange) {
    body.timeRange = requestTimeRange;
  }
  const res = await fetch(
    `/v1/instances/${encodeURIComponent(instanceId)}/queries/metrics-views/${encodeURIComponent(req.metricsView)}/aggregation`,
    {
      method: "POST",
      credentials: "include",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(body),
    },
  );
  if (!res.ok) {
    const message = await res.text().catch(() => "Metric query failed");
    const detail = message || `Metric query failed (${res.status})`;
    return {
      row: {},
      error: `${req.metricsView}.${req.measures.join(", ")}: ${detail}`,
    };
  }
  const json = await res.json();
  return { row: json?.data?.[0] ?? {}, error: null };
}

async function fetchLiveRequest(
  instanceId: string,
  req: MetricTreeLiveRequest,
  options: ResolveMetricTreeLiveOptions,
): Promise<Record<string, MetricTreeLiveValue>> {
  const hasTimeSeries = options.hasTimeSeries ?? true;
  const current = await runAggregationRequest(
    instanceId,
    req,
    req.timeMode === "dashboard" ? options.timeRange : undefined,
    options.where,
    hasTimeSeries,
    options.priority ?? 35,
  );
  const shouldCompare =
    req.timeMode === "dashboard" &&
    hasTimeSeries &&
    !!options.showTimeComparison &&
    !!options.comparisonTimeRange;
  const previous = shouldCompare
    ? await runAggregationRequest(
        instanceId,
        req,
        options.comparisonTimeRange,
        options.where,
        hasTimeSeries,
        options.priority ?? 35,
      )
    : { row: {}, error: null };

  return Object.fromEntries(
    req.measures.map((measure) => [
      metricRequestMeasureKey(req, measure),
      {
        value: current.row[measure] ?? null,
        previousValue: shouldCompare ? previous.row[measure] ?? null : null,
        error: current.error,
        previousError: previous.error,
      },
    ]),
  );
}

export async function resolveMetricTreeLiveValues(
  options: ResolveMetricTreeLiveOptions,
): Promise<ResolvedMetricTree | null> {
  const { tree, instanceId } = options;
  if (!tree || !instanceId) return resolveMetricTreeValues(tree, {}, {
    dashboardTimeLabel: options.dashboardTimeLabel,
    dashboardFilterLabel: options.dashboardFilterLabel,
  });
  const requests = planLiveMetricRequests(tree);
  const liveValues: Record<string, MetricTreeLiveValue> = {};
  const liveWarnings: string[] = [];
  const bindableNodes = tree.nodes.filter((node) => {
    const binding = node.metricBinding as MetricTreeMetricBinding | undefined;
    return bindingSource(binding) === "rill_metrics_view" && !!binding?.measure && !!bindingMetricsView(binding);
  });
  if (bindableNodes.length === 0) {
    liveWarnings.push("No live metric bindings were found on this tree.");
  }
  await Promise.all(
    requests.map(async (req) => {
      const values = await fetchLiveRequest(instanceId, req, options);
      for (const node of tree.nodes) {
        const binding = node.metricBinding as MetricTreeMetricBinding | undefined;
        if (!binding?.measure) continue;
        const bindingKey = metricBindingKey(binding);
        const valueKey = metricRequestMeasureKey(req, binding.measure);
        if (!bindingKey || bindingKey !== valueKey) continue;
        const value = values[valueKey] ?? { value: null, error: `${req.metricsView}.${binding.measure}: Metric value missing` };
        if (value.error) liveWarnings.push(value.error);
        liveValues[bindingKey] = value;
      }
    }),
  );
  const resolved = resolveMetricTreeValues(tree, liveValues, {
    dashboardTimeLabel: options.dashboardTimeLabel,
    dashboardFilterLabel: options.dashboardFilterLabel,
  });
  if (resolved) resolved.liveWarnings = [...resolved.liveWarnings, ...liveWarnings];
  return resolved;
}

export function resolveMetricTreeValues(
  tree: AuthoredMetricTree | null,
  liveValues: Record<string, MetricTreeLiveValue>,
  context: MetricTreeResolveContext = {},
): ResolvedMetricTree | null {
  if (!tree) return null;
  const warnings: string[] = [];
  const byId = new Map(tree.nodes.map((node) => [node.id, node]));
  const resolutions: Record<string, MetricTreeNodeResolution> = {};

  function resolveNode(nodeId: string, visiting = new Set<string>()): MetricTreeNodeResolution {
    if (resolutions[nodeId]) return resolutions[nodeId];
    const node = byId.get(nodeId);
    if (!node) {
      return {
        value: null,
        previousValue: null,
        delta: null,
        deltaPercent: null,
        measuredValue: null,
        previousMeasuredValue: null,
        computedValue: null,
        previousComputedValue: null,
        driftValue: null,
        driftPercent: null,
        sourceStatus: "error",
        sourceLabel: null,
        sourceContext: null,
        formulaText: null,
        sourceError: `Missing node ${nodeId}`,
      };
    }
    if (visiting.has(nodeId)) {
      warnings.push(`Formula cycle at ${nodeId}`);
      return {
        value: toNumber(node.value),
        previousValue: null,
        delta: null,
        deltaPercent: null,
        measuredValue: null,
        previousMeasuredValue: null,
        computedValue: null,
        previousComputedValue: null,
        driftValue: null,
        driftPercent: null,
        sourceStatus: "error",
        sourceLabel: null,
        sourceContext: null,
        formulaText: formulaText(node.formula),
        sourceError: "Formula cycle",
      };
    }

    const binding = node.metricBinding as MetricTreeMetricBinding | undefined;
    const sourceLabel = metricBindingLabel(binding);
    const sourceContext = metricBindingContextLabel(binding, context);
    const bindingKey = binding && bindingSource(binding) === "rill_metrics_view" ? metricBindingKey(binding) : null;
    const live = bindingKey ? liveValues[bindingKey] : undefined;
    const fallback = toNumber(node.value);
    let measuredValue: number | null = null;
    let previousMeasuredValue: number | null = null;
    let sourceStatus: MetricTreeSourceStatus = "unbound";
    let sourceError: string | null = null;

    if (bindingKey && live) {
      measuredValue = toNumber(live.value);
      previousMeasuredValue = toNumber(live.previousValue);
      sourceStatus = live.error ? "error" : "live";
      sourceError = live.error ?? live.previousError ?? null;
    } else if (bindingKey) {
      sourceStatus = "error";
      sourceError = "Live binding was not resolved";
    } else if (sourceLabel) {
      sourceStatus = "error";
      sourceError = "Unsupported live binding";
    } else if (fallback != null && !node.formula) {
      sourceStatus = "fallback";
      measuredValue = fallback;
    }

    let computedValue: number | null = null;
    let previousComputedValue: number | null = null;
    const f = node.formula;
    if (f?.operands?.length) {
      visiting.add(nodeId);
      const operandResolutions = f.operands.map((operand) => resolveNode(operand, visiting));
      visiting.delete(nodeId);
      const operandValues = operandResolutions.map((resolution) => resolution.value);
      const previousOperandValues = operandResolutions.map((resolution) => resolution.previousValue);
      if (operandValues.every((v): v is number => v != null)) {
        computedValue = evalFormula(f.op, operandValues);
      }
      if (previousOperandValues.every((v): v is number => v != null)) {
        previousComputedValue = evalFormula(f.op, previousOperandValues);
      }
    }

    let value = measuredValue ?? computedValue ?? fallback;
    let previousValue = previousMeasuredValue ?? previousComputedValue;
    if (computedValue != null && measuredValue == null) sourceStatus = "computed";
    let driftValue: number | null = null;
    let driftPercent: number | null = null;
    if (computedValue != null && measuredValue != null) {
      driftValue = measuredValue - computedValue;
      driftPercent = computedValue !== 0 ? driftValue / computedValue : null;
      value = measuredValue;
    }
    const { delta, deltaPercent } = deltaFor(value, previousValue);

    const resolution = {
      value,
      previousValue,
      delta,
      deltaPercent,
      measuredValue,
      previousMeasuredValue,
      computedValue,
      previousComputedValue,
      driftValue,
      driftPercent,
      sourceStatus,
      sourceLabel,
      sourceContext,
      formulaText: formulaText(f),
      sourceError,
    };
    resolutions[nodeId] = resolution;
    return resolution;
  }

  for (const node of tree.nodes) resolveNode(node.id);

  return {
    ...tree,
    nodes: tree.nodes.map((node) => ({
      ...node,
      value: resolutions[node.id]?.value ?? node.value ?? null,
      delta: resolutions[node.id]?.delta ?? node.delta ?? null,
    })),
    nodeResolutions: resolutions,
    liveWarnings: warnings,
  };
}
