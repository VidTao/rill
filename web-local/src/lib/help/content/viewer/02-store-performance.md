---
title: Store Performance
audience: viewer
order: 1
group: dashboards
status: ready
---

# Store Performance

Use [**Store Performance**](https://bratrax.com/canvas/performance_overview) as the daily health check. It answers: "Is the business up or down, and is marketing spend producing enough sales?"

```loom
id: a8f9b4964a8e4c6baf6df6c71cd65a96
label: Store Performance
duration: 1:46
```

## Look first at

- **Total Sales** — Shopify-aligned sales for the selected period.
- **Returns** — refunds and return pressure.
- **Total Spend** — paid media and custom marketing spend.
- **Paid ROAS** — paid-attributed sales divided by ad spend.
- **MER** — Total Sales divided by total marketing spend. This is the broad business-efficiency number.

## How to read it

Start at the top KPI tiles. The value is for the selected date range. The comparison shows how it changed versus the previous period.

Then move down:

- **Total Sales over time** shows whether the period changed because of one spike or a steady trend.
- **New vs Returning Order Revenue** shows whether growth came from new customers or the existing base.
- **Order Count by Type** separates one-time, subscription-first, and subscription-renewal behavior.
- **MER vs Paid ROAS** separates blended business efficiency from paid-channel attribution.
- **Subscription Snapshot** shows current subscription health and is not always comparable to period sales charts.

## Common misreads

- **Total Sales and Attributed Sales are different.** Total Sales is store truth. Attributed Sales is sales assigned to channels or campaigns.
- **MER and Paid ROAS are different.** MER uses total store sales. Paid ROAS uses paid-attributed sales.
- **Today may be incomplete.** If the latest refresh happened mid-day, today's sales and spend are partial.
- **Returns are dated by refund date.** A refund today may belong to an older order.

## Drilldown moves

- If Total Sales dropped, check whether order count, AOV, or returns changed.
- If Paid ROAS dropped, open [**Attribution**](https://bratrax.com/canvas/campaign_deep_dive) and find the channel or campaign that moved.
- If MER dropped but Paid ROAS is stable, look for non-paid sales weakness, custom spend, or subscription mix.
- If new-customer orders dropped, open **Attribution** and check new-customer ROAS/CPA by source.

## Good Claude questions

- "Summarize Store Performance for the last 7 days versus the previous 7 days."
- "Was the sales change caused by order count, AOV, or returns?"
- "Why did MER move in the selected period?"
- "Compare Paid ROAS and MER and explain the difference."
