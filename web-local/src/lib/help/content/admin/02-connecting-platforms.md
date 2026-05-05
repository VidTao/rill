---
title: Connecting platforms
audience: admin
order: 2
status: ready
---

# Connecting platforms

Connect ad platforms, your store, and other data sources from the **Connectors** tab in the top nav.

## Adding a platform

1. Click **Connectors** in the top nav.
2. Click the platform you want to add.
3. Sign in to that platform when redirected.
4. Pick which accounts, ad accounts, or properties to include.
5. Bratrax starts pulling data immediately.

## How long until data shows up

| Platform type | First backfill | Ongoing refresh |
|---|---|---|
| Ad platforms (Facebook, Google, TikTok) | 15–60 minutes | Hourly |
| Shopify | A few minutes for recent orders; longer for full history | Hourly |
| Email / SMS (Klaviyo) | 30–60 minutes | Hourly |
| Stripe (subscriptions) | 30–60 minutes | Hourly |

You'll see progress on the connector card while the backfill is running.

## What each platform contributes

| Platform | What you get |
|---|---|
| Shopify | Orders, customers, products, sessions |
| Facebook Ads | Spend, impressions, clicks, conversions per campaign / ad set / ad |
| Google Ads | Same as Facebook, plus search keyword data |
| TikTok Ads | Same as Facebook |
| Klaviyo | Email/SMS campaigns, opens, clicks, attributed revenue |
| Stripe | Subscription orders and recurring revenue |
| Amazon Ads / SP | Sponsored ads and seller performance |

## Reconnecting an expired token

OAuth tokens expire periodically — Shopify after 60 days, others between 30 and 90 days. When that happens:

1. The connector card turns yellow with a "Reconnect" prompt.
2. Click **Reconnect** and sign in to the platform again.
3. Backfill resumes from where it left off.

You don't lose any historical data when reconnecting.

## Disconnecting

Click the platform card and select **Disconnect**. Historical data stays in your warehouse — only future updates stop. To remove the data entirely, contact support.
