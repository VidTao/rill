---
title: External & custom landing pages
audience: admin
order: 9
group: workspace
status: ready
---

# External & custom landing pages

Run pages outside your store — advertorials, pre-sell or bridge pages, quizzes,
or a custom (non-Shopify) checkout? You can track those too, so a visitor who
reads your advertorial and then buys is stitched to that order instead of landing
in **Direct**.

Set it up from **[Connectors → External Landing Pages](https://bratrax.com/connectors)**.
What you pick depends on how your funnel is built.

## Funnelish funnels
Choose **Funnelish** — a purpose-built integration. Copy the Bratrax pixel
snippet onto your funnel and paste our webhook URL into Funnelish; orders and
visitor journeys then flow into Bratrax. The modal walks you through it.

## Any other external page (Custom / Other)
For advertorials, pre-sell/bridge pages, quizzes, or a third-party checkout that
syncs orders into your Shopify/WooCommerce store, choose **Custom / Other**.
You'll get a snippet to paste into the `<head>` of every page you want tracked —
it works from a plain HTML template or a Google Tag Manager **Custom HTML** tag.
When the snippet is live, click **Mark installed** in the modal; we don't
auto-detect it.

Once it's in place, Bratrax records each page view with its UTM tags and click
IDs, and tags every link on the page that points back to your store. That's what
stitches a reader of your advertorial to the order they place later.

**Connect your store first.** The snippet auto-fills your storefront domain, and
that's what lets these outside pages link up to real orders. Paste the snippet
before connecting your store and the links back won't be tagged — the pages can't
be matched to purchases.

## Declare the other domains you own
If your advertorials and bridge pages live on their own domains, add them under
**[Settings → Account → Your other domains](https://bratrax.com/settings/account)**.
Otherwise a hop from one of your domains into your store can be miscredited as a
referral that steals the ad's credit.

## What you'll see
Recovered journeys move out of the **Direct** bucket and show up against the
campaign that actually drove them on
[Attribution](https://bratrax.com/canvas/campaign_deep_dive). Click a blue
**Attributed Orders** number to open a single order and see the full path,
including the external page.

## Not yet supported
**ClickFunnels** and **GoHighLevel** are on the roadmap. If that's your setup,
email [support@bratrax.com](mailto:support@bratrax.com) and we'll let you know
when they land.
