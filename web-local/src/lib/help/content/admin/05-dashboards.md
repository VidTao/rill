---
title: Canvas dashboards
audience: admin
order: 3
group: building
status: ready
---

# Canvas dashboards

A **canvas dashboard** is the layout viewers see. It's a YAML file in `/dashboards/` that arranges tiles in a grid powered by metrics views.

```loom
id: 58f3efa9a75e483892ebe484a6c59afa
label: Customizing charts
duration: 1:28
```

## A minimum canvas

```yaml
type: canvas
display_name: Performance overview

rows:
  - height: 140px
    items:
      - width: 12
        kpi_grid:
          metrics_view: order_metrics
          measures:
            - total_revenue
            - order_count
            - aov

  - height: 360px
    items:
      - width: 8
        chart:
          type: line
          metrics_view: order_metrics
          x: order_date
          y: total_revenue
      - width: 4
        chart:
          type: bar
          metrics_view: order_metrics
          x: channel
          y: total_revenue
```

Save and the dashboard appears in the sidebar for everyone with access.

## The grid

- Each **row** has a `height` (in pixels or `auto`) and a list of `items`
- Each **item** has a `width` out of 12 columns
- Items in a row sum to 12 (or less; remaining space stays blank)

## Component types

**`kpi_grid`** — big numbers across the top.

```yaml
kpi_grid:
  metrics_view: order_metrics
  measures: [total_revenue, order_count, aov]
```

**`chart`** — line, bar, area, or stacked bar.

```yaml
chart:
  type: line
  metrics_view: order_metrics
  x: order_date
  y: total_revenue
  color: channel       # optional: splits into one line per channel
```

**`table`** — sortable detailed breakdown.

```yaml
table:
  metrics_view: order_metrics
  dimensions: [channel, country]
  measures: [total_revenue, order_count]
```

**`markdown`** — section headers and notes.

```yaml
markdown:
  content: "## Top of funnel"
```

## Default time range

Viewers can pick any time range, but you can set a sensible default:

```yaml
type: canvas
display_name: Last 30 days
default_time_range: P30D    # ISO 8601 duration

rows:
  - ...
```

## What viewers see

Viewers see exactly what you build, minus the file editor — they get a clean, interactive version of the canvas with the time picker, filters, and drill-downs all enabled.

## Full reference

For every component option, see the [Rill canvas docs](https://docs.rilldata.com/build/canvas-dashboards).
