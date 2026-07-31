import type { V1MetricsViewRowsResponseDataItem } from "@rilldata/web-common/runtime-client";

// One row of order_timeline_v1 after deserialization. The actual response
// items are typed loosely (Record<string, unknown>); these helpers project
// the columns we know are present on each row_type.

export type RowType =
  | "behavior_event"
  | "attribution_touchpoint"
  | "resolver_evidence"
  | "conversion";

export interface OrderTimelineRow extends V1MetricsViewRowsResponseDataItem {
  order_id: string;
  order_number?: string | number;
  customer_id?: string;
  email?: string;
  conversion_ts?: string;
  revenue?: number;
  event_ts?: string;
  event_rank?: number;
  row_type?: RowType;
  event_type?: string;
  event_source?: string;
  activity_id?: string;
  anonymous_id?: string;
  url?: string;
  referrer?: string;
  channel_group?: string;
  source?: string;
  medium?: string;
  campaign?: string;
  campaign_id?: string;
  adset?: string;
  ad?: string;
  is_conversion?: 0 | 1;
  is_attribution_touchpoint?: 0 | 1;
  is_attribution_winner?: 0 | 1;
  is_first_touch_winner?: 0 | 1;
  is_last_touch_winner?: 0 | 1;
  is_linear_winner?: 0 | 1;
  is_time_decay_winner?: 0 | 1;
  is_position_winner?: 0 | 1;
  resolution_status?: string;
  resolution_reason?: string;
  resolution_confidence?: string;
  first_touch_weight?: number;
  last_touch_weight?: number;
  linear_weight?: number;
  time_decay_weight?: number;
  position_weight?: number;
}

// One row in Modal 1's order list. Aggregated client-side from the winning
// touchpoint rows returned for a clicked cell — one entry per unique order_id.
export interface OrderListRow {
  order_id: string;
  order_number: string | number | undefined;
  email: string | undefined;
  conversion_ts: string | undefined;
  revenue: number | undefined;
  last_touch_weight: number | undefined;
  resolution_confidence: string | undefined;
  resolution_reason: string | undefined;
  channel_group: string | undefined;
  campaign: string | undefined;
  adset: string | undefined;
  ad: string | undefined;
}

// Summary banner shown at the top of Modal 2.
export interface WinnerSummary {
  channel_group: string | undefined;
  campaign: string | undefined;
  adset: string | undefined;
  ad: string | undefined;
  resolution_reason: string | undefined;
  resolution_confidence: string | undefined;
  last_touch_weight: number | undefined;
}

export interface CancelledOrderItemRow
  extends V1MetricsViewRowsResponseDataItem {
  attribution_slice_id?: string;
  attribution_model?: string;
  attribution_weight?: number;
  order_id?: string;
  order_number?: string | number;
  email?: string;
  order_created_at?: string;
  cancelled_at?: string;
  cancel_reason?: string;
  order_total?: number;
  currency?: string;
  line_item_id?: string;
  product_title?: string;
  variant_title?: string;
  sku?: string;
  quantity?: number;
  unit_price?: number;
  discount_amount?: number;
  item_value?: number;
  channel_group?: string;
  source?: string;
  medium?: string;
  campaign?: string;
  adset?: string;
  ad?: string;
}

export interface CancelledOrderItem {
  line_item_id: string;
  product_title?: string;
  variant_title?: string;
  sku?: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  item_value: number;
}

export interface CancelledOrderGroup {
  order_id: string;
  order_number?: string | number;
  email?: string;
  order_created_at?: string;
  cancelled_at?: string;
  cancel_reason?: string;
  order_total: number;
  currency: string;
  attribution_weight: number;
  items: CancelledOrderItem[];
}
