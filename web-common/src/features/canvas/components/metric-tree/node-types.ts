export type MetricTreeNodeType =
  | "output"
  | "revenue_stream"
  | "driver"
  | "lever"
  | "surface"
  | "experiment";

export interface NodeTypeTheme {
  label: string;
  color: string;
}

export const NODE_TYPE_THEME: Record<MetricTreeNodeType, NodeTypeTheme> = {
  output: { label: "Output", color: "#993C1D" },
  revenue_stream: { label: "Revenue stream", color: "#534AB7" },
  driver: { label: "Driver", color: "#0F6E56" },
  lever: { label: "Lever", color: "#854F0B" },
  surface: { label: "Surface", color: "#5F5E5A" },
  experiment: { label: "Experiment", color: "#3B6D11" },
};

export const NODE_TYPE_ORDER: MetricTreeNodeType[] = [
  "output",
  "revenue_stream",
  "driver",
  "lever",
  "surface",
  "experiment",
];

export function isMetricTreeNodeType(value: string | null | undefined): value is MetricTreeNodeType {
  return !!value && Object.prototype.hasOwnProperty.call(NODE_TYPE_THEME, value);
}

export function nodeTypeTheme(type: string | null | undefined): NodeTypeTheme {
  return isMetricTreeNodeType(type) ? NODE_TYPE_THEME[type] : NODE_TYPE_THEME.driver;
}
