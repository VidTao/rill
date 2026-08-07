---
title: Best practices
audience: admin
order: 5
group: building
status: ready
---

# Best practices

Conventions that keep your dashboards reliable and fast as the data grows.

## Naming

- **Models** — prefix by purpose:
  - `dim_*` for dimension-style tables (one row per entity: `dim_customers`, `dim_products`)
  - `fct_*` for fact tables (one row per event: `fct_orders`, `fct_sessions`)
  - `mart_*` for stitched/aggregated outputs (`mart_daily_performance`)
- **Metrics views** — match the model name where possible (`order_metrics` over `fct_orders`)
- **Dashboards** — kebab-case in the filename, human-readable in `display_name`

## When to make a new model vs. reuse one

- **Reuse** if an existing model has the data you need at the right grain.
- **New model** if you need a different grain (e.g. weekly when the existing one is daily) or a meaningfully different filter.

Avoid copy-pasting a model and editing one line. Either parameterize the original or use it as-is and filter at the metrics-view level.

## Performance

- **Filter early.** Push `WHERE` clauses as close to the source as possible.
- **Avoid `SELECT *`** on raw tables. Pick only the columns you need.
- **Materialize heavy models.** Add `-- @materialize: true` at the top of any model that takes more than 2 seconds. The result gets cached and refreshed automatically.
- **Use date filters** even when a dashboard shows "all time." Limit the underlying model to the longest range you actually need (e.g. 2 years).

## Dashboard design

- **One question per dashboard.** Don't try to answer everything in one canvas — make multiple focused dashboards.
- **KPIs at the top, charts in the middle, tables at the bottom.** This is what people expect.
- **Always include a time picker.** Don't hardcode a range.
- **Use comparisons.** A number alone isn't actionable; "X, up 12% from last week" is.

## Saving and testing

- Save often. There's no draft state — the editor validates as you go.
- After a non-trivial change, open the dashboard as a viewer would and click around. Check filters, time ranges, and drill-downs.
- For changes that affect many dashboards, do them late in the day or on a weekend so fewer people are affected if you have to roll back.

## When in doubt

Look at the dashboards that came with your account — they're the canonical examples. Copy their structure rather than inventing new patterns.
