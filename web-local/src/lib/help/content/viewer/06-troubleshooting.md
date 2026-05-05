---
title: Troubleshooting
audience: viewer
order: 6
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

## Numbers look stale

Every dashboard shows an "as of" timestamp. Bratrax refreshes every 1–3 hours.

If the timestamp is older than 3 hours, a refresh is overdue. Wait 15 minutes and reload. If it's still stale, contact support.

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
