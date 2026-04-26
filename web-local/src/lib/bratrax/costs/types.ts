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

export type StoreSettings = Record<string, unknown>;

export type CostTab = "cogs" | "shipping" | "gateway" | "expenses";
