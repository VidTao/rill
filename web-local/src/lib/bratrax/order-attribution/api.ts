// Query builders + aggregation helpers for the order-attribution drilldown.
// The two modals both query the `order_timeline` metrics view via
// createQueryServiceMetricsViewRows; this module composes the V1Expression
// `where` filters they need and reshapes the response rows.

import {
  createAndExpression,
  createInExpression,
} from "@rilldata/web-common/features/dashboards/stores/filter-utils";
import type { V1Expression } from "@rilldata/web-common/runtime-client";
import type {
  OrderListRow,
  OrderTimelineRow,
  WinnerSummary,
} from "./types";

export const ORDER_TIMELINE_METRICS_VIEW = "order_timeline_metrics";

// Dimensions that exist on order_timeline_metrics. Cell filters from the
// parent dashboard (cross_metrics) can reference dimensions this view doesn't
// expose — most notably `attribution_model`, which only lives on cross_metrics.
// We strip those leaves before forwarding, otherwise the runtime rejects the
// query with `invalid dimension reference`. The dashboard's attribution_model
// filter is translated by buildOrderListWhere into a filter on the right
// per-model `is_*_winner` flag (see MODEL_WINNER_COL).
const ORDER_TIMELINE_DIMENSIONS = new Set([
  "activity_id",
  "ad",
  "adset",
  "anonymous_id",
  "campaign",
  "campaign_id",
  "campaign_type",
  "channel_group",
  "conversion_ts",
  "customer_id",
  "email",
  "event_source",
  "event_ts",
  "event_type",
  "is_attribution_touchpoint",
  "is_attribution_winner",
  "is_first_touch_winner",
  "is_last_touch_winner",
  "is_linear_winner",
  "is_time_decay_winner",
  "is_position_winner",
  "is_conversion",
  "medium",
  "order_id",
  "raw_params",
  "referrer",
  "resolution_confidence",
  "resolution_reason",
  "resolution_status",
  "row_type",
  "source",
  "url",
]);

// The campaign_deep_dive pivot (cross_metrics) names its level-2 dimension
// `source`, but the value is a campaign-type label (Search / Conversion /
// (none)), derived from dim_campaigns — not the raw platform source.
// order_timeline_v1 exposes that same derived label as `campaign_type` and
// keeps the raw platform value (facebook / paworigin.com) under `source` for
// the per-order timeline display. So when forwarding a cell filter to the
// drilldown we remap `source` -> `campaign_type`; otherwise a Meta cell's
// `source='(none)'` (or a Google cell's `source='Search'`) matches zero rows
// against the raw `source` column. Only the pivot dimensions whose meaning
// differs across the two views belong here.
const PIVOT_DIM_RENAME: Record<string, string> = {
  source: "campaign_type",
};

// Per-model "this touchpoint won" flag. The order_timeline_v1 view carries a
// boolean column for each supported attribution model; we filter Modal 1 on
// the one matching the dashboard's current model so the drilldown reflects
// the same view of the world. Default last_touch when the cell filter doesn't
// pin a model (matches the dashboard's default filter).
const MODEL_WINNER_COL: Record<string, string> = {
  first_touch: "is_first_touch_winner",
  last_touch: "is_last_touch_winner",
  linear: "is_linear_winner",
  time_decay: "is_time_decay_winner",
  position: "is_position_winner",
};

export function winnerColumnFor(model: string | undefined): string {
  return MODEL_WINNER_COL[model ?? ""] ?? "is_last_touch_winner";
}

// "last_touch" → "last-touch" for human-readable header text. Falls back to
// the raw model string if it's not one we recognise.
export function formatAttributionModel(model: string | undefined): string {
  if (!model) return "last-touch";
  return model.replace(/_/g, "-");
}

// Walk a filter tree and keep only IN-leaves whose dimension exists on
// order_timeline_metrics. Returns undefined when nothing survives.
function pruneFilterToOrderTimeline(
  expr: V1Expression | undefined,
): V1Expression | undefined {
  if (!expr?.cond) return undefined;
  const op = expr.cond.op;
  const inner = expr.cond.exprs ?? [];
  if (op === "OPERATION_AND" || op === "OPERATION_OR") {
    const kept = inner
      .map((child) => pruneFilterToOrderTimeline(child))
      .filter((c): c is V1Expression => !!c);
    if (kept.length === 0) return undefined;
    if (kept.length === 1) return kept[0];
    return { cond: { op, exprs: kept } };
  }
  // Leaf comparison (IN / NIN / EQ / etc.): remap pivot-specific dimension
  // names to their order_timeline column, then drop if the ident isn't on the
  // drilldown view.
  const ident = inner[0]?.ident;
  if (typeof ident !== "string") return expr;
  const mapped = PIVOT_DIM_RENAME[ident] ?? ident;
  if (!ORDER_TIMELINE_DIMENSIONS.has(mapped)) {
    return undefined;
  }
  if (mapped === ident) return expr;
  // Clone the leaf with the remapped identifier, preserving the values.
  return {
    cond: {
      op: expr.cond.op,
      exprs: [{ ident: mapped }, ...inner.slice(1)],
    },
  };
}

// Truthy-fallback for the human-readable order label. The compiler emits
// order_timeline_v1.order_number as a non-nullable Int64, so any NULL coming
// out of the dim layer lands as a literal 0 in the row. JS nullish-coalescing
// won't catch that (0 is not nullish), so we treat 0 / empty-string the same
// as null and fall back to the raw order_id. Real Shopify order numbers get
// a leading "#" to match the merchant's dashboard view.
export function formatOrderLabel(
  orderNumber: string | number | undefined | null,
  orderId: string | undefined,
): string {
  // Treat all of {null, undefined, 0, "", "0"} as "missing".
  const hasOrderNumber =
    orderNumber !== null &&
    orderNumber !== undefined &&
    orderNumber !== 0 &&
    orderNumber !== "" &&
    orderNumber !== "0";
  if (hasOrderNumber) return `#${orderNumber}`;
  return orderId ?? "—";
}

// Modal 1 (orders list): scope to the winning attribution touchpoints matching
// the clicked cell. We AND in the row_type gate, the per-model winner gate
// (selected from the dashboard's attribution_model filter), and the pruned
// cell filter from the parent dashboard. attribution_model lives on
// cross_metrics but not on order_timeline_metrics, so we pull its value out
// BEFORE pruning and translate it to the matching is_*_winner column here.
export function buildOrderListWhere(
  cellFilters: V1Expression | undefined,
): V1Expression {
  const labels = extractCellLabels(cellFilters);
  const winnerCol = winnerColumnFor(labels["attribution_model"]);
  const exprs: V1Expression[] = [
    createInExpression("row_type", ["attribution_touchpoint"]),
    createInExpression(winnerCol, [1]),
  ];
  const pruned = pruneFilterToOrderTimeline(cellFilters);
  if (pruned) exprs.push(pruned);
  return createAndExpression(exprs);
}

// Modal 2 (per-order timeline): fetch every row for one order_id. No time
// filter — orders span up to 30 days of pre-conversion history.
export function buildOrderTimelineWhere(orderId: string): V1Expression {
  return createInExpression("order_id", [orderId]);
}

// Modal 1 returns at most one row per touchpoint, but a single order can have
// multiple winning touchpoints if the cell isn't fully scoped (e.g. clicking
// at the campaign level rolls up multiple ad-sets). De-dupe by order_id,
// preferring the latest conversion_ts.
export function dedupeOrders(
  rows: OrderTimelineRow[] | undefined,
): OrderListRow[] {
  if (!rows) return [];
  const byOrder = new Map<string, OrderListRow>();
  for (const r of rows) {
    if (!r.order_id) continue;
    const existing = byOrder.get(r.order_id);
    const next: OrderListRow = {
      order_id: r.order_id,
      order_number: r.order_number,
      email: r.email,
      conversion_ts: r.conversion_ts,
      revenue: r.revenue,
      last_touch_weight: r.last_touch_weight,
      resolution_confidence: r.resolution_confidence,
      resolution_reason: r.resolution_reason,
      channel_group: r.channel_group,
      campaign: r.campaign,
      adset: r.adset,
      ad: r.ad,
    };
    if (
      !existing ||
      (next.conversion_ts &&
        existing.conversion_ts &&
        next.conversion_ts > existing.conversion_ts)
    ) {
      byOrder.set(r.order_id, next);
    }
  }
  return Array.from(byOrder.values()).sort((a, b) => {
    const ats = a.conversion_ts ?? "";
    const bts = b.conversion_ts ?? "";
    return bts.localeCompare(ats);
  });
}

// Sort timeline rows for Modal 2 — chronological with the same tiebreakers the
// underlying VIEW projects (event_rank disambiguates events that share a
// timestamp; touchpoint=10, behavior=20, evidence=30, conversion=40).
export function sortTimelineRows(
  rows: OrderTimelineRow[] | undefined,
): OrderTimelineRow[] {
  if (!rows) return [];
  return [...rows].sort((a, b) => {
    const ats = a.event_ts ?? "";
    const bts = b.event_ts ?? "";
    if (ats !== bts) return ats.localeCompare(bts);
    const ar = a.event_rank ?? 0;
    const br = b.event_rank ?? 0;
    if (ar !== br) return ar - br;
    return (a.row_type ?? "").localeCompare(b.row_type ?? "");
  });
}

// Pull the winning touchpoint out of a timeline for the header banner. The
// "winner" is model-specific — the same order has different winning rows
// under first_touch vs last_touch — so the caller passes the attribution
// model from the dashboard and we pick the row flagged for that model.
export function findWinner(
  rows: OrderTimelineRow[] | undefined,
  attributionModel?: string,
): WinnerSummary | null {
  if (!rows) return null;
  const col = winnerColumnFor(attributionModel) as keyof OrderTimelineRow;
  const winner = rows.find(
    (r) => r.row_type === "attribution_touchpoint" && r[col],
  );
  if (!winner) return null;
  return {
    channel_group: winner.channel_group,
    campaign: winner.campaign,
    adset: winner.adset,
    ad: winner.ad,
    resolution_reason: winner.resolution_reason,
    resolution_confidence: winner.resolution_confidence,
    last_touch_weight: winner.last_touch_weight,
  };
}

// Plain-English description of why this order was matched to its campaign,
// derived from the technical resolution_reason code. The resolver appends
// "; recovered_*" modifiers when it fills in missing ad/adset metadata —
// strip those before classifying so the column stays short.
//
// Patterns (see clickhouse/attribution/attribution_sale_level_v2.sql):
//   order_source_rule                            Shopify order's source_name
//   order_url_param:<param>                      UTM/click ID on the order URL
//   order_url_classification                     order URL referrer matched a platform
//   <source>_param:<param>                       tracking param on a viewed page
//   order_param_ledger:<path>:<param>            stored param from a past order
//   <source>_param_ledger:<path>:<param>         stored param from a past visit
//   <source>_classification                      visit URL matched a platform
//   klaviyo_event:<name>:<id_source>             Klaviyo email/SMS click
//   fallback_direct_no_signal                    no signal — attributed to Direct
export function humanizeResolutionReason(
  reason: string | undefined,
): string {
  if (!reason) return "—";
  const base = reason.split(";")[0].trim();
  if (base === "order_source_rule") return "Shopify source field";
  if (base === "order_url_classification") return "Checkout URL classified";
  if (base === "fallback_direct_no_signal") return "No signal — counted as Direct";
  if (base.startsWith("order_url_param:")) return "Tracking on checkout URL";
  if (base.startsWith("order_param_ledger:")) return "Saved checkout tracking";
  if (base.startsWith("klaviyo_event:")) return "Klaviyo email/SMS click";
  if (base.includes("_param_ledger:")) return "Saved visit tracking";
  if (base.includes("_param:")) return "Tracking on visited page";
  if (base.endsWith("_classification")) return "Visit URL classified";
  return base;
}

// Walk an AND-of-INs filter expression and pull the (dim, value) pairs out,
// used to render Modal 1's subtitle as "Meta / Campaign X / Adset Y / Ad Z".
// Only handles the shape getFiltersForCell emits — top-level AND with leaf
// IN-expressions. Anything else is ignored.
export function extractCellLabels(
  expr: V1Expression | undefined,
): Record<string, string> {
  const out: Record<string, string> = {};
  if (!expr?.cond?.exprs) return out;
  const walk = (e: V1Expression) => {
    const inner = e.cond?.exprs;
    if (!inner || inner.length === 0) return;
    // AND: recurse into children.
    if (e.cond?.op === "OPERATION_AND") {
      for (const child of inner) walk(child);
      return;
    }
    // IN: first expr is { ident }, rest are { val }.
    if (e.cond?.op === "OPERATION_IN" && inner[0]?.ident && inner[1]?.val !== undefined) {
      out[inner[0].ident] = String(inner[1].val);
    }
  };
  walk(expr);
  return out;
}
