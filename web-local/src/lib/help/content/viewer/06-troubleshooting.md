---
title: Troubleshooting
audience: viewer
order: 12
status: ready
---

# Troubleshooting

The most common issues and what to do about them.

## "No data" on a dashboard

Almost always one of two things:

1. **The time range is too narrow.** Try expanding to *Last 30 days* or *Year to date*.
2. **A filter is excluding everything.** Click **Clear filters** at the top of the dashboard.

If neither helps, the data may still be loading. Newly connected platforms take 15–60 minutes for the first backfill.

## Numbers don't match Facebook / Google / Shopify

This is expected, not a bug. There are two reasons:

1. **Attribution model.** Bratrax uses a blended, deduplicated view. Facebook and Google each claim credit for the same conversion, so adding their reported totals overstates reality. Bratrax shows the actual unduplicated number.
2. **Lookback window.** Bratrax uses a 30-day window by default. Platforms vary (Facebook defaults to 7-day click). A longer window finds more attributed conversions.

If the difference is large — more than ~30% — contact support, as there may be a connector issue.

## Direct looks too high

Direct means Bratrax could not find a reliable eligible marketing touch for those orders.

Common causes:

1. Customers returned without a tracked click.
2. Browser tracking was blocked or unavailable.
3. Shopify did not preserve landing/referrer data for that checkout path.
4. The touch happened outside the attribution lookback window.
5. Subscription, marketplace, or app-driven orders are not normal acquisition visits.

Direct is not automatically wrong. A sudden Direct spike should be investigated by checking whether Shopify landing data and browser checkout events were present for those orders.

## Today looks wrong

Today is often partial. Spend, orders, email events, and attribution refresh on schedules and may not land at the same minute.

Use **Yesterday** or **Last 7 days** when you need a stable read.

## Attribution changed but Store Performance did not

This can happen. Store Performance is store truth. Attribution decides which channel gets credit for the same store sales.

If total sales are stable but Attribution changed, the business may be fine while the source mix changed.

## Subscription, TikTok Shop, or Shop App rows look strange

Some rows do not behave like normal website sessions:

- **Subscription** rows can be renewals, not new acquisition.
- **TikTok Shop** orders may come from TikTok's commerce path.
- **Shop App** orders may have limited storefront attribution.

Use acquisition metrics for acquisition decisions, and use Store Performance for total business health.

## Numbers look stale

Every dashboard shows an "as of" timestamp. Store orders refresh hourly; every other integration every 4 hours.

If store data is more than about two hours stale — or other integrations more than about five — a refresh is overdue. Wait 15 minutes and reload. If it's still stale, contact support.

## The page won't load or is blank

In order:

1. Hard reload (Ctrl+Shift+R or Cmd+Shift+R)
2. Sign out and back in
3. Try a different browser or an incognito window

If the issue persists, contact support with the URL you were on and any error messages on screen.

## I see data that isn't mine

Stop using the dashboard and contact support immediately. Include the URL where you saw it.

## How to reach support

Use the chat icon in the bottom-right of any page, or email your Bratrax contact directly.
