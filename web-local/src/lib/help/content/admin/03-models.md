---
title: Models (SQL)
audience: admin
order: 3
status: ready
---

# Models (SQL)

A **model** is a saved SQL query that prepares data for metrics views and dashboards. Models live in the `/models/` folder. Open one to read it, or click **+ New file** to create one.

## A minimum model

A model is a single SQL statement. Save it as a `.sql` file and the result becomes a table you can query, named after the file.

```sql
-- /models/orders_by_day.sql
SELECT
  date_trunc('day', order_created_at) AS order_date,
  count()                             AS orders,
  sum(total)                          AS revenue
FROM dim_orders
GROUP BY order_date
```

After saving, `orders_by_day` is available for any metrics view to read from.

## Common patterns

**Filter by date range:**

```sql
SELECT *
FROM dim_orders
WHERE order_created_at >= now('UTC') - INTERVAL 90 DAY
```

**Join two tables:**

```sql
SELECT
  o.order_id,
  o.total,
  c.first_acquisition_channel
FROM dim_orders o
LEFT JOIN dim_customers c ON o.customer_id = c.customer_id
```

**First order per customer** (ClickHouse has no `QUALIFY` — use a subquery, then filter):

```sql
SELECT * EXCEPT (rn)
FROM (
  SELECT
    *,
    row_number() OVER (
      PARTITION BY customer_id
      ORDER BY order_created_at
    ) AS rn
  FROM dim_orders
)
WHERE rn = 1
```

**Rolling 7-day revenue:**

```sql
SELECT
  order_date,
  sum(revenue) OVER (
    ORDER BY order_date
    ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
  ) AS revenue_7d
FROM orders_by_day
```

## SQL dialect cheatsheet

Models run on ClickHouse. The functions you'll reach for most:

- **Dates** — `date_trunc('day' | 'week' | 'month', col)`, `now('UTC')`, `col - INTERVAL 30 DAY`
- **Strings** — `lower()`, `concat(a, '-', b)`, `extract(url, 'utm_source=([^&]+)')`
- **Aggregation** — `sum`, `count()`, `count(DISTINCT ...)` (or `uniqExact(...)`), `avg`
- **Conditional** — `CASE WHEN ... THEN ... ELSE ... END` (or `multiIf(...)`), `coalesce(a, b)`, `nullif(a, 0)`

For the full reference, see the [ClickHouse SQL reference](https://clickhouse.com/docs/en/sql-reference).

## Performance

- **Filter early.** `WHERE` clauses on date columns and IDs are dramatically faster than filtering after a join.
- **Avoid `SELECT *`** on large tables. Pick only the columns you need.
- **Materialize slow models.** Add a comment at the top to cache the result so downstream metrics don't re-run it every time:

```sql
-- @materialize: true
SELECT ...
```

## Validating

The editor flags syntax errors and missing tables as you type. A green check at the bottom means the model compiled and is ready to use.
