export type MetricNodeType =
  | "output"
  | "revenue_stream"
  | "driver"
  | "lever"
  | "surface"
  | "experiment";

export type MetricUnit = "currency" | "count" | "percent" | "ratio" | "days" | "text";
export type FormulaOp = "add" | "multiply" | "subtract" | "ratio" | "custom";
export type GoodDirection = "up" | "down";
export type ExperimentStatus = "backlog" | "do_now" | "running" | "won" | "lost" | "shipped";

export interface MetricFormula {
  op: FormulaOp;
  operands: string[];
  expression?: string;
}

export interface MetricBinding {
  source?: string;
  metricsView?: string;
  metrics_view?: string;
  measure?: string;
  where?: unknown;
  timeMode?: string;
  time_range?: string;
}

export type ExperimentMeasurementType = "traffic_source_test" | "metric_filter_test" | "manual";
export type ExperimentAttributionModel =
  | "first_touch"
  | "last_touch"
  | "linear"
  | "position_based"
  | "time_decay";

export interface MetricBindingFilter {
  dimension: string;
  op: "in";
  values: string[];
}

export interface ExperimentMeasurementBinding {
  type: ExperimentMeasurementType;
  metricsView: string;
  primaryMeasure: string;
  successMetricLabel: string;
  filters: MetricBindingFilter[];
  startDate: string;
  endDate?: string | null;
  attributionModel?: ExperimentAttributionModel | null;
  guardrailNodeIds: string[];
  spendMeasure?: string | null;
  revenueMeasure?: string | null;
  ordersMeasure?: string | null;
  trafficMeasure?: string | null;
}

export interface IceScore {
  impact?: number;
  confidence?: number;
  ease?: number;
}

export interface AuthoredMetricNode {
  id: string;
  treeId?: string;
  parentId: string | null;
  label: string;
  type: MetricNodeType;
  unit: MetricUnit;
  value: number | null;
  delta?: number | null;
  deltaBasis?: "prior_period" | "target" | null;
  goodDirection: GoodDirection;
  formula?: MetricFormula | null;
  metricBinding?: MetricBinding;
  measurementBinding?: ExperimentMeasurementBinding;
  owner?: string | null;
  baseline?: number | null;
  target?: number | null;
  targetDate?: string | null;
  guardrailIds?: string[];
  hypothesis?: string | null;
  ice?: IceScore;
  status?: ExperimentStatus | null;
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
  validation?: MetricTreeValidation;
}

export interface MetricTreeIssue {
  code: string;
  node_id?: string;
  message: string;
}

export interface MetricTreeValidation {
  errors: MetricTreeIssue[];
  warnings: MetricTreeIssue[];
}

export interface RollupResult {
  values: Record<string, number | null>;
  drift: Record<string, number>;
}

export interface PropagationResult {
  projected: Record<string, number | null>;
  delta: Record<string, number>;
  path: string[];
}

export interface MetricTreeEvent {
  event_id: string;
  tree_id: string;
  node_id: string;
  event_type: string;
  payload?: unknown;
  created_at: string;
  created_by: string;
}

export type MetricTreeDecisionEventType =
  | "decision"
  | "experiment_launch"
  | "result_readout"
  | "owner_change"
  | "status_change";

export interface CreateMetricTreeEventPayload {
  eventType: MetricTreeDecisionEventType;
  note: string;
  outcome?: string;
  nextAction?: string;
}

export interface MetricTreeEvidence {
  status: "ok" | "unbound" | "unsupported";
  message?: string;
  resultLabel?: string;
  binding?: ExperimentMeasurementBinding;
  dateRange?: { start?: string; end?: string };
  metrics?: Record<string, number | null>;
  guardrails?: Array<{
    nodeId: string;
    label: string;
    unit?: MetricUnit;
    value?: number | null;
    savedValue?: number | null;
    metricBinding?: MetricBinding;
  }>;
}
