---
title: Connecting platforms
audience: admin
order: 2
status: ready
---

# Connecting platforms

Connect ad platforms, your store, and other data sources from the **Connectors** tab in the top nav.

**Video walkthrough**

```loom
d3631b9edef54f58abed0cef24de3dfb
```

## Adding a platform

1. Click **Connectors** in the top nav.
2. Click the platform you want to add.
3. Sign in to that platform when redirected.
4. Pick which accounts, ad accounts, or properties to include.
5. Bratrax starts pulling data immediately.

## How long until data shows up

| Platform | First data | Ongoing refresh |
|---|---|---|
| Store (Shopify, WooCommerce) | Recent orders in minutes; ~1 year backfilled over hours | Hourly |
| Ad platforms (Meta, Google, TikTok, Microsoft/Bing, Pinterest, Taboola, Outbrain) | 15–60 min to start; ~60 days backfilled over hours to days | Every 4 hours |
| Email / SMS (Klaviyo, Bloomreach) | 30–60 minutes | Every 4 hours |

You'll see progress on the connector card while the backfill is running.

## What each platform contributes

| Platform | What you get |
|---|---|
| Shopify | Orders, customers, products, sessions — including Shopify-native subscriptions |
| WooCommerce | Orders, customers, products (via the Bratrax WordPress plugin) |
| Meta (Facebook / Instagram) | Spend, impressions, clicks, conversions per campaign / ad set / ad |
| Google Ads | Same as Meta, plus search keyword data |
| TikTok Ads | Same as Meta |
| Microsoft (Bing) Ads | Same as Meta |
| Pinterest Ads | Same as Meta |
| Taboola | Spend, impressions, clicks, conversions per campaign |
| Outbrain | Spend, impressions, clicks, conversions per campaign |
| Klaviyo | Email/SMS campaigns & flows, opens, clicks, attributed revenue |
| Bloomreach | Email/SMS & engagement campaigns, attributed revenue |
| External landing pages / Funnelish | Orders + visitor journeys via the Bratrax tracking pixel (no ad connector) |

## Reconnecting an expired token

OAuth tokens expire periodically — Shopify after 60 days, others between 30 and 90 days. When that happens:

1. The connector card turns yellow with a "Reconnect" prompt.
2. Click **Reconnect** and sign in to the platform again.
3. Backfill resumes from where it left off.

You don't lose any historical data when reconnecting.

## Disconnecting

Click the platform card and select **Disconnect**. Historical data stays in your warehouse — only future updates stop. To remove the data entirely, contact support.
