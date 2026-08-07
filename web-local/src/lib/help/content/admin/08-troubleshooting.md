---
title: Troubleshooting builds
audience: admin
order: 6
group: building
status: ready
---

# Troubleshooting

The errors you'll actually hit and how to fix them.

## "YAML parse error" when saving a metric or dashboard

The editor shows a line and column. The most common causes:

- **Indentation.** YAML is whitespace-sensitive — use 2 spaces, never tabs.
- **Missing colon.** `name customer` should be `name: customer`.
- **Unquoted strings starting with special characters.** Anything starting with `*`, `&`, `?`, `:`, `-`, or `$` needs quotes: `description: "$ amount"`.

## "Metrics view references unknown column"

The metrics view names a column that isn't on the table it reads from.

1. Check the spelling exactly — ClickHouse is case-sensitive.
2. Confirm the column exists on that table. If you're unsure which columns are
   available, ask Claude or contact support.
3. If the column should exist and doesn't, the prepared table may not have
   refreshed yet — see "A dashboard shows no data" below.

## "Canvas references unknown metrics view"

The dashboard points to a `metrics_view: foo` that doesn't exist.

- Check the metrics view file name matches exactly (without the `.yaml` extension).
- Confirm the metrics view file saved without errors.

## Dashboard renders but every number is 0 or empty

Three usual suspects:

1. **The time range excludes all data.** Try *Year to date*.
2. **A filter is too restrictive.** Click *Clear filters*.
3. **The prepared table is empty.** Check another dashboard reading the same data — if it's also empty, the table hasn't refreshed and support can confirm.

## "Connector token expired"

A platform's OAuth token has lapsed (typically every 30–90 days).

1. Go to **Connectors**.
2. Find the yellow card.
3. Click **Reconnect** and sign in again.

No data is lost; the backfill resumes from where it left off.

## Source X is failing to refresh

Check the connector card — there's usually an error message. Common causes:

- Token expired (see above)
- Permission revoked on the platform side
- Rate-limited (will retry automatically)
- WooCommerce only: your site's firewall (Cloudflare or a WordPress security plugin) may be blocking our requests — see [the firewall fix](/help/admin/connecting-platforms)

If the message doesn't make sense, contact support with the connector name and the exact error.

## A dashboard is slow

- **Narrow the default time range** on the dashboard — a year of daily data is
  slower to draw than a month.
- **Reduce tiles per dashboard** — each one is its own query. Split a crowded
  dashboard in two.
- **Prefer fewer, coarser dimensions** in a table tile; high-cardinality
  breakdowns are the usual cause.

If a dashboard takes more than ~5 seconds and you've tried the above, contact support.

## When to escalate

Email support with:

- The file path or dashboard URL
- The exact error message
- A screenshot if it's a UI issue

For data-correctness questions ("this number doesn't match Facebook"), include the metric, the date range, and the discrepancy.
