
-- Blended metrics: one row per event (order/ad/refund)
-- No pre-aggregation by date — Rill handles timezone-aware date grouping

SELECT created_at AS ts, subtotal_price + total_discounts AS gross_sales, total_discounts AS discounts, subtotal_price AS net_sales, total_price AS revenue, 0.0 AS spend, 0.0 AS returns, CAST(0 AS BIGINT) AS impressions, CAST(0 AS BIGINT) AS clicks
FROM dim_orders

UNION ALL

SELECT CAST(date AS TIMESTAMP) AS ts, 0.0 AS gross_sales, 0.0 AS discounts, 0.0 AS net_sales, 0.0 AS revenue, spend, 0.0 AS returns, impressions, clicks
FROM dim_ad_performance

UNION ALL

SELECT created_at AS ts, 0.0 AS gross_sales, 0.0 AS discounts, 0.0 AS net_sales, 0.0 AS revenue, 0.0 AS spend, ABS(amount) AS returns, CAST(0 AS BIGINT) AS impressions, CAST(0 AS BIGINT) AS clicks
FROM dim_transactions
WHERE kind = 'refund'
