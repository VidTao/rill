// Status vocabulary + color theming for metric-tree nodes.
//
// The status value is data-driven (it comes from the metrics view), so unknown
// values fall back to "neutral" rather than erroring. Each status carries a
// light and a dark accent: the canvas dashboard pins --color-* to its theme's
// light values, so dark mode needs an explicit brighter accent (same gotcha the
// order-attribution graph documents).

export type MetricTreeStatus = "good" | "warning" | "bad" | "neutral";

export interface StatusTheme {
  label: string;
  light: string;
  dark: string;
}

export const STATUS_THEME: Record<MetricTreeStatus, StatusTheme> = {
  good: { label: "Good", light: "#16a34a", dark: "#22c55e" },
  warning: { label: "Warning", light: "#d97706", dark: "#f59e0b" },
  bad: { label: "Bad", light: "#dc2626", dark: "#ef4444" },
  neutral: { label: "Neutral", light: "#6b7280", dark: "#9ca3af" },
};

export function statusTheme(status: string | null | undefined): StatusTheme {
  return STATUS_THEME[status as MetricTreeStatus] ?? STATUS_THEME.neutral;
}


export function statusLabel(status: string | null | undefined): string | null {
  if (!status) return null;
  const known = STATUS_THEME[status as MetricTreeStatus];
  if (known) return known.label;
  return status.replaceAll("_", " ").replace(/\b\w/g, (c) => c.toUpperCase());
}
