-- Cross-Object Metrics Model (Attribution + Spend)
-- UNION ALL of two data sources:
--   1. Attribution touchpoints (pixel-tracked orders with channel/campaign from UTMs)
--   2. Platform spend (ad platform API data with spend/impressions/clicks)
-- All 5 attribution models included — user selects via attribution_model dimension.
-- Channels appear if they have EITHER attributed sales OR ad spend.

-- Attribution: one row per (order, touchpoint, model) from attribution_weights
SELECT
  aw.channel_group,
  aw.medium,
  aw.source,
  aw.campaign,
  aw.adset,
  aw.ad,
  aw.model AS attribution_model,
  aw.weight,
  aw.order_id,
  CAST(aw.conversion_ts AS DATE) AS date,
  o.total_price,
  0.0 AS spend,
  CAST(0 AS BIGINT) AS impressions,
  CAST(0 AS BIGINT) AS clicks,
  c.orders_count
FROM attribution_weights aw
LEFT JOIN dim_orders o ON aw.order_id = o.order_id
LEFT JOIN dim_customers c ON o.customer_id = c.customer_id
WHERE o.financial_status = 'paid'

UNION ALL

-- Platform spend: one row per (campaign, date) from dim_ad_performance
-- Joins dim_campaigns for campaign_type → correct channel_group per taxonomy
-- Repeated for each model so spend is consistent regardless of model selection
SELECT
  CASE
    WHEN ap.channel = 'meta' THEN 'Paid Social'
    WHEN ap.channel = 'tiktok' THEN 'Paid Social'
    WHEN ap.channel = 'google' THEN
      CASE cam.campaign_type
        WHEN 'SEARCH' THEN 'Google Search'
        WHEN 'DISPLAY' THEN 'Google Display'
        WHEN 'VIDEO' THEN 'Google Video'
        WHEN 'SHOPPING' THEN 'Google Shopping'
        WHEN 'PERFORMANCE_MAX' THEN 'Google PMax'
        WHEN 'DEMAND_GEN' THEN 'Google Demand Gen'
        ELSE 'Google Ads'
      END
    ELSE ap.channel
  END AS channel_group,
  'cpc' AS medium,
  ap.channel AS source,
  ap.campaign_name AS campaign,
  '' AS adset,
  '' AS ad,
  m.model AS attribution_model,
  0.0 AS weight,
  '' AS order_id,
  ap.date,
  0.0 AS total_price,
  ap.spend,
  ap.impressions,
  ap.clicks,
  0 AS orders_count
FROM dim_ad_performance ap
LEFT JOIN dim_campaigns cam ON ap.campaign_id = cam.campaign_id
CROSS JOIN (
  SELECT 'first_touch' AS model
  UNION ALL SELECT 'last_touch'
  UNION ALL SELECT 'linear'
  UNION ALL SELECT 'time_decay'
  UNION ALL SELECT 'position'
) m
