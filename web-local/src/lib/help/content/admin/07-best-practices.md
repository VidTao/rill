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

The prepared tables you read from already follow a convention: `dim_*` holds one
row per entity (`dim_customers`, `dim_products`), `fct_*` one row per event
(`fct_orders`), and `*_v1` tables are pre-computed rollups.

- **Metrics views** — name for the question they answer (`order_metrics`), not the
  table they read (`dim_orders`).
- **Dashboards** — kebab-case in the filename, human-readable in `display_name`.

## When to make a new metrics view vs. reuse one

- **Reuse** if an existing view already exposes the measures and dimensions you
  need. Filter at the dashboard level instead of duplicating the view.
- **New view** if you need a different grain or a genuinely different set of
  measures.

Avoid copy-pasting a view and editing one line — a second view with the same
measures is a second place to keep them correct.

## Performance

- **Filter early.** Push `WHERE` clauses as close to the source as possible.
- **Avoid `SELECT *`** on raw tables. Pick only the columns you need.
- **Set a sensible default time range** rather than "all time." Most questions are answered by the last 30 or 90 days, and a narrower default draws faster.

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
