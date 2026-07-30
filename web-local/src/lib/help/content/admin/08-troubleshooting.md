---
title: Troubleshooting
audience: admin
order: 8
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

The metrics view points to a column that doesn't exist on its model.

1. Check the spelling exactly — ClickHouse is case-sensitive.
2. Open the model and confirm the column is in the SELECT list.
3. If you just changed the model, save it first, then the metrics view.

## "Canvas references unknown metrics view"

The dashboard points to a `metrics_view: foo` that doesn't exist.

- Check the metrics view file name matches exactly (without the `.yaml` extension).
- Confirm the metrics view file saved without errors.

## Dashboard renders but every number is 0 or empty

Three usual suspects:

1. **The time range excludes all data.** Try *Year to date*.
2. **A filter is too restrictive.** Click *Clear filters*.
3. **The underlying model returned no rows.** Open the model and run the preview — if it's empty, fix the model first.

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
- WooCommerce only: your site's firewall (Cloudflare or a WordPress security plugin) may be blocking our requests — see [the firewall fix](/help/admin/connecting-accounts)

If the message doesn't make sense, contact support with the connector name and the exact error.

## A dashboard is slow

- **Add `-- @materialize: true`** to the heaviest model upstream of the dashboard.
- **Reduce the date range** the model covers.
- **Avoid wide joins** — join only the columns you need.

If a dashboard takes more than ~5 seconds and you've tried the above, contact support.

## When to escalate

Email support with:

- The file path or dashboard URL
- The exact error message
- A screenshot if it's a UI issue

For data-correctness questions ("this number doesn't match Facebook"), include the metric, the date range, and the discrepancy.
