---
title: Attribution
audience: viewer
order: 3
status: ready
---

# Attribution

Use [**Attribution**](https://bratrax.com/canvas/campaign_deep_dive) when you want to know where sales came from and which channels, campaigns, ad sets, or ads deserve credit.

**Video walkthrough**

```loom
c3a62d5c20d245aab5f86f578a5a851e
```

The dashboard defaults to **last-touch** attribution. That means the last eligible source before the purchase gets credit.

## Look first at

- **Attributed Sales** — sales assigned to channels and campaigns.
- **ROAS** — attributed sales divided by spend.
- **Attributed Orders** — order credit assigned by the attribution model.
- **CPA / NC CPA** — cost per purchase or new-customer purchase.
- **NC ROAS** — new-customer attributed sales divided by spend.

## How to read it

The pivot table is the main object. Read it top to bottom:

1. Start at **channel_group**.
2. Expand into **campaign**.
3. Expand into **adset**.
4. Expand into **ad**.

Use the line charts below the table to confirm whether a campaign is consistently changing or if one day caused the movement.

## Direct

**Direct** means Bratrax could not find a reliable eligible marketing touch for the order in the attribution window.

Direct can include:

- typed or bookmarked visits,
- customers returning without a tracked touch,
- blocked or missing browser tracking,
- expired lookback windows,
- checkout paths where Shopify did not preserve landing/referrer data.

Do not automatically treat Direct as bad data. Treat a sudden Direct spike as a tracking question: did raw Shopify landing data or browser checkout coverage drop?

## Common misreads

- **Platform totals will not match Bratrax.** Facebook, Google, TikTok, and Klaviyo can each claim the same order. Bratrax deduplicates.
- **Last-touch is not first-touch.** A customer may first arrive from Google but purchase later through Email or Direct.
- **Subscription renewals can distort CPA.** Use new-customer metrics when judging acquisition.
- **Spend without attributed sales does not always mean failure.** There may be lag, low volume, or upper-funnel activity.

## Drilldown moves

- If ROAS dropped, expand channel to campaign and campaign to ad set.
- If NC CPA rose, check whether spend increased, new purchases dropped, or AOV fell.
- If Direct rose, compare Direct by day and ask whether pixel/landing evidence was missing.
- If a campaign looks too good, check whether orders are new customers or returning customers.

## Good Claude questions

- "Which channel drove the biggest change in Attributed Sales last week?"
- "Show campaigns with spend above 100 and ROAS below 1 in the last 30 days."
- "Why is Direct high this week?"
- "Which campaigns acquired new customers most efficiently?"
