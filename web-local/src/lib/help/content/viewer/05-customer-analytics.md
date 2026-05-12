---
title: Customer Analytics
audience: viewer
order: 5
status: ready
---

# Customer Analytics

Use **Customer Analytics** to understand customer quality, subscriber health, MRR, LTV, retention, and which acquisition sources create valuable subscribers.

## Look first at

- **Active Subscribers** — current active subscriber count.
- **MRR** — current monthly recurring revenue estimate.
- **Average Subscriber LTV** — average value of subscribers.
- **Churn Rate** — share of subscribers who stopped being active.
- **Subscriber Acquisition** — which channels and campaigns created subscribers.

## Two sections behave differently

The dashboard has two kinds of data:

- **Right Now** shows the current state. It is not meant to be filtered by the selected date range.
- **Selected Period** changes with your date range and comparison.

This is intentional. Current MRR and selected-period order revenue answer different questions.

## How to read it

Start with **Right Now** to understand the business base today. Then use **Selected Period** to understand what changed during the date range.

The **Subscriber Acquisition** table traces subscribers back to the attribution on their first order. This prevents subscription renewals from making retention channels look like acquisition channels.

## Common misreads

- **MRR is reconstructed.** It is based on subscription order history and active-window logic, not always a direct subscription-platform export.
- **Subscriber acquisition is first-order attribution.** Renewals should not take credit for acquiring the subscriber.
- **LTV moves slowly.** Short date ranges can be noisy.
- **Churn needs context.** A small subscriber base can produce volatile churn percentages.

## Drilldown moves

- If MRR dropped, check active subscribers and churn.
- If subscriber LTV changed, check acquisition source and cohort retention.
- If new subscribers dropped, inspect Subscriber Acquisition by channel and campaign.
- If retention changed, inspect cohort rows by start month.

## Good Claude questions

- "What changed in active subscribers and MRR this month?"
- "Which channels acquired the highest-LTV subscribers?"
- "Is churn worsening or is the subscriber base just small?"
- "Which subscriber cohorts have the weakest retention?"
