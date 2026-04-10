-- Cross-Object Metrics Model
-- Auto-generated: joins dim tables via attribution_weights bridge
-- Enables ROAS, CPA, attributed revenue, and other cross-object KPIs

SELECT
  aw.campaign AS campaign_id,
  aw.channel,
  aw.model AS attribution_model,
  aw.weight,
  aw.order_id,
  CAST(aw.conversion_ts AS TIMESTAMP) AS date,
  aw.revenue AS aw_revenue,
  o.total_price,
  o.financial_status,
  o.customer_id AS order_customer_id,
  ap.spend,
  ap.impressions,
  ap.clicks,
  c.orders_count,
  c.total_spent AS customer_total_spent
FROM attribution_weights aw
LEFT JOIN dim_orders o ON aw.order_id = o.order_id
LEFT JOIN dim_ad_performance ap ON aw.campaign = ap.campaign_id AND CAST(aw.conversion_ts AS DATE) = ap.date
LEFT JOIN dim_customers c ON o.customer_id = c.customer_id
WHERE aw.model = 'last_touch'
  AND o.financial_status = 'paid'
