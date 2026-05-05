---
title: Glossary
audience: shared
order: 1
status: ready
---

# Glossary

Plain-English definitions of every term used across Bratrax dashboards, Claude answers, and help pages.

## Revenue and orders

- **Revenue** — gross sales from your store, before refunds. Includes tax and shipping unless the dashboard explicitly says "net."
- **Net revenue** — revenue minus refunds, taxes, and shipping (when a dashboard calls it out).
- **Orders** — count of completed checkouts.
- **AOV** (Average Order Value) — Revenue ÷ Orders.

## Ad spend and efficiency

- **Ad Spend** — total amount paid to ad platforms in the period.
- **ROAS** (Return on Ad Spend) — Revenue ÷ Ad Spend. ROAS of 3 means $3 of revenue per $1 spent.
- **MER** (Marketing Efficiency Ratio) — Total Revenue ÷ total marketing spend, blended across all channels. Sometimes called "blended ROAS."
- **CPA / CAC** (Cost per Acquisition / Customer Acquisition Cost) — Ad Spend ÷ new customers acquired.
- **CPM** — cost per 1,000 ad impressions.
- **CPC** — cost per ad click.
- **CTR** (Click-Through Rate) — clicks ÷ impressions, as a percentage.
- **Conversion Rate** — orders ÷ sessions (or clicks, depending on context), as a percentage.

## Customer

- **LTV** (Lifetime Value) — average total revenue per customer, from their first order onward. Cohorted by acquisition month.
- **New customer** — placed their first-ever order in the selected period.
- **Returning customer** — placed a non-first order in the period.
- **Repeat rate** — percentage of customers who place more than one order.
- **Cohort** — a group of customers sharing an attribute (usually acquisition month).

## Attribution

- **Blended attribution** — Bratrax's deduplicated view, combining first-party data with platform data. Most KPIs default to this.
- **Platform-reported** — what each ad platform claims it drove. Tends to over-report because every platform claims credit for the same conversion.
- **First-touch** — credit goes to the first ad or source the customer interacted with.
- **Last-touch** — credit goes to the last source before purchase.
- **Lookback window** — how far back attribution looks for a touch. Default: 30 days.
- **Direct** — orders without an attributable touch (typed-in URL, ad blocker, expired window).

## Data freshness

- **As of** — timestamp of the last successful refresh. Numbers reflect that moment, not "now."
- **Refresh window** — how often Bratrax pulls fresh data. Default: every 1–3 hours.
- **Backfill** — the initial historical pull when you first connect a platform.

## Building blocks (admin)

- **Source** — a connection to an external platform (Shopify, Facebook, etc.).
- **Model** — a saved SQL query that prepares data.
- **Metrics view** — a declaration of which dimensions and measures are available.
- **Canvas / dashboard** — a layout of tiles powered by metrics views.
- **Dimension** — something you group or filter by (channel, product, country).
- **Measure** — a number you aggregate (revenue, orders, clicks).
