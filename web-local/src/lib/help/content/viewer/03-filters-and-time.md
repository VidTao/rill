---
title: Filters and time range
audience: viewer
order: 9
status: ready
---

# Filters and time range

Every dashboard has controls at the top that scope what you're looking at. Use them to narrow down to a specific question.

## Time range

Click the time range picker to change the window. You'll see presets like **Last 7 days**, **Last 30 days**, **Last 90 days** for rolling windows, plus **Month to date**, **Quarter to date**, and **Year to date** that run from the start of the period to now. Pick **Custom** to set any start and end date.

Use short ranges for daily operations and longer ranges for strategy:

- **Last 7 days** — daily check-in.
- **Last 30 days** — campaign and product review.
- **Month to date** — pacing against the current month.
- **Last 90 days** — customer and subscription patterns.

The time grain is usually set automatically. If a line chart looks too noisy, switch from day to week.

## Comparing periods

Next to the time picker, the **Compare to** control overlays a second period:

- **Previous period** — the same length window immediately before
- **Previous year** — the same dates one year ago
- **Off** — no comparison

When a comparison is on, KPI tiles show a percentage delta and charts draw a second line for the comparison period.

## Dimension filters

Below the time controls, click **+ Filter** to add a dimension filter — channel, campaign, country, product, etc. Multiple filters combine with AND. So *channel = Facebook AND country = United States* shows only US Facebook traffic.

To remove a filter, click the × on its chip.

## Click-to-filter

Many charts, tables, and pivots can filter the whole dashboard when clicked. This is the fastest way to investigate:

- In [**Attribution**](https://bratrax.com/canvas/campaign_deep_dive), click a channel or campaign.
- In [**Products**](https://bratrax.com/canvas/product_performance), click a product or SKU.
- In [**Email & SMS**](https://bratrax.com/canvas/email_marketing), click a campaign or flow.
- In [**Customer Analytics**](https://bratrax.com/canvas/customer_analytics), click a subscriber acquisition channel.

## Clearing filters

If you've narrowed things down too far and the dashboard looks empty, click **Clear filters** at the top. This resets dimension filters but keeps your time range.

## Sharing a filtered view

Filters and the time range are saved in the URL. Copy the URL from your browser and send it — anyone with access to the dashboard will see exactly the same view, with the same filters and time range applied.

## When a dashboard looks empty

Most empty dashboards are caused by a narrow date range or a filter. Click **Clear filters**, then try **Last 30 days**.
