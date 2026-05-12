---
title: New Customer Source Report
audience: viewer
order: 7
status: ready
---

# New Customer Source Report

Use **New Customer Source Report** when the question is specifically acquisition: "Which sources are bringing new customers, and what did they cost?"

This dashboard is narrower than Store Performance or Attribution. It focuses on new-customer economics.

## Look first at

- **Total Spend** — spend in the selected period.
- **NCV / New Customer Value** — value from new customers.
- **NCP / New Customer Purchases** — new-customer purchase count.
- **NC ROAS** — new-customer value divided by spend.
- **NC CPA** — spend divided by new-customer purchases.

## How to read it

Sort by spend first, then by NC ROAS or NC CPA. Small sources can look excellent or terrible because of low volume, so do not overreact to tiny rows.

Use this dashboard when acquisition performance matters more than total revenue.

## Common misreads

- **This is not total ROAS.** It is focused on new customers.
- **Returning customer revenue is not the goal here.** Use Store Performance or Attribution for total revenue.
- **Low spend can make ratios unstable.** A single purchase can swing NC ROAS or NC CPA.
- **Attribution model matters.** The default is last-touch unless changed.

## Drilldown moves

- If NC CPA rose, check whether spend rose or new purchases fell.
- If NC ROAS fell, check whether new customer value fell or spend shifted.
- If a source has high spend and poor NC ROAS, inspect it in **Attribution** by campaign/adset/ad.
- If new-customer volume dropped, compare with **Store Performance** new-customer orders.

## Good Claude questions

- "Which sources acquired the most new customers last 30 days?"
- "Which source has the best NC CPA above 10 purchases?"
- "Why did NC ROAS fall this week?"
- "Compare new-customer acquisition by source versus the previous period."
