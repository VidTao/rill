/** Cost settings types — matches Flask /cost-settings API responses. */

export interface ProductCogs {
  product_id: string;
  title: string;
  handle?: string;
  cogs_amount: number | null;
  handling_fee: number | null;
}

export interface GatewayFee {
  gateway_name: string;
  percentage_fee: number;
  fixed_fee: number;
  is_shopify_imported: boolean;
}

export interface ShippingProfile {
  entity_id: string;
  data: {
    profile_name: string;
    rate_type: "flat_rate" | "weight_based" | "zone_based";
    flat_rate: number | null;
    weight_tiers: WeightTier[];
    zones: ShippingZone[];
    is_default: boolean;
    _deleted?: boolean;
  };
  updated_at: string;
}

export interface WeightTier {
  min_kg: number;
  max_kg: number;
  rate: number;
}

export interface ShippingZone {
  zone_name: string;
  countries: string[];
  rate: number;
}

export interface ExpenseRule {
  entity_id: string;
  data: {
    expense_type: "fixed" | "variable";
    title: string;
    category: string;
    fixed_amount: number | null;
    period: "daily" | "monthly" | "yearly";
    variable_metric: "" | "revenue" | "gross_sales" | "ad_spend" | "orders";
    variable_percentage: number | null;
    is_ad_spend: boolean;
    start_date: string | null;
    end_date: string | null;
    is_active: boolean;
    _deleted?: boolean;
  };
  updated_at: string;
}

export type MediaSpendScopeChannel =
  | ""
  | "meta"
  | "google"
  | "tiktok"
  | "bing"
  | "taboola"
  | "outbrain"
  | "pinterest";

export type MediaSpendScopeMatchField =
  | "campaign_name"
  | "campaign_id"
  | "ad_set_name"
  | "ad_set_id"
  | "ad_id"
  | "ad_name"
  | "account_id";

export type MediaSpendScopeOperator =
  | "equals"
  | "prefix"
  | "contains"
  | "regex";
export type MediaSpendScopeAction = "include" | "exclude";

export interface MediaSpendScopeRuleData {
  name: string;
  channel: MediaSpendScopeChannel;
  account_id: string;
  match_field: MediaSpendScopeMatchField;
  operator: MediaSpendScopeOperator;
  match_value: string;
  action: MediaSpendScopeAction;
  priority: number;
  start_date: string;
  end_date: string;
  is_active: boolean;
}

export interface MediaSpendScopeRule {
  entity_id: string;
  data: MediaSpendScopeRuleData;
  updated_at: string;
}

export interface MediaSpendScopeAccountOption {
  channel: MediaSpendScopeChannel;
  account_id: string;
}

export interface MediaSpendScopeGuidance {
  title: string;
  body: string;
  examples?: Array<{
    rule: string;
    matches: string;
    does_not_match: string;
  }>;
  unmatched_policy?: string;
}

export type StoreSettings = Record<string, unknown>;

export interface ProfitReadinessCost {
  key: "cogs" | "fees" | "shipping" | "ad_spend" | "custom_costs";
  label: string;
  complete: boolean;
  coverage: number;
  status: "ready" | "estimated" | "missing";
}

export interface ProfitReadiness {
  available: boolean;
  reason?: string;
  profit_ready?: boolean;
  orders?: number;
  costs?: ProfitReadinessCost[];
}

export type CostTab =
  | "cogs"
  | "shipping"
  | "gateway"
  | "expenses"
  | "media_scope"
  | "calculation";
