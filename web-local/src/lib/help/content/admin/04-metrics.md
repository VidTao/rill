---
title: Metrics views
audience: admin
order: 4
status: ready
---

# Metrics views

A **metrics view** is the layer between your data (a model) and a dashboard. It declares which numbers can be measured and how they can be sliced. Metrics views live in `/metrics/` as YAML files.

## Anatomy of a metrics view

```yaml
type: metrics_view
display_name: Order metrics
table: orders_by_day        # the model this reads from
timeseries: order_date      # the date column

dimensions:
  - name: channel
    column: acquisition_channel
    display_name: Channel

  - name: country
    column: country_code

measures:
  - name: total_revenue
    expression: sum(revenue)
    format_preset: currency_usd

  - name: order_count
    expression: count(*)

  - name: aov
    expression: sum(revenue) / nullif(count(*), 0)
    format_preset: currency_usd
```

Save the file and the metrics view is immediately queryable from any dashboard.

## Dimensions

Dimensions are the things you group or filter by — channel, product, country, etc.

- **`name`** — what dashboards refer to it as
- **`column`** — the column on the model
- **`display_name`** — what users see in the UI
- **`description`** — optional, surfaces as a tooltip

## Measures

Measures are the numbers — sums, counts, ratios.

- **`name`** — what dashboards refer to it as
- **`expression`** — a SQL aggregation
- **`format_preset`** — how it displays. Common values: `currency_usd`, `percentage`, `humanize`, `integer`
- **`description`** — optional

For ratios, **always wrap the denominator in `nullif(..., 0)`** to avoid divide-by-zero errors when there's no data.

## Time grain

`timeseries:` declares the date column. Add `smallest_time_grain: day` (or `hour`, `week`, `month`) to control the finest aggregation viewers can pick.

## View-level filters

If you always want to exclude something — for example, test orders — filter at the view level:

```yaml
table: orders_by_day
where: order_status != 'test'
```

## Common validation errors

- **"Unknown column"** — the column doesn't exist on the model. Check spelling, then save the model.
- **"Duplicate measure name"** — two measures share a `name`. Rename one.
- **"Invalid expression"** — SQL won't parse. The error points to the line.

## Full reference

For every option available, see the [Rill metrics view docs](https://docs.rilldata.com/build/metrics-view).
