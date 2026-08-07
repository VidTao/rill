---
title: Commerce Profile Graph
audience: viewer
order: 7
group: dashboards
status: ready
---

# Commerce Profile Graph

[**Commerce Profile Graph**](https://bratrax.com/canvas/commerce_profile_graph) is the aggregate view of Bratrax's identity graph —
how well your store, email/SMS platform, and on-site activity resolve into
known people, and what those people are worth. Where [**Profile Explorer**](https://bratrax.com/canvas/profile_explorer) is a
person-level directory you work through, Commerce Profile Graph answers
portfolio questions: how big is the reachable audience, which segments exist,
which acquisition sources produce valuable customers, and where attribution
evidence is weak.

**Video walkthrough**

```loom
f67431e8682f41a389b7e0e9ef3e3505
```

## The five sections

### 1. Profile Coverage

Counts of known customers, known non-buyers, anonymous profiles, and
email-reachable profiles observed across Shopify, Klaviyo, and first-party
activity. This is your identity-resolution scoreboard: the higher the share of
known and reachable profiles, the more of your traffic you can actually act on.

### 2. Activation Segments

Every profile is placed into one operational segment (abandoned checkout,
known lead, active subscriber, first-time buyer, high-LTV customer, winback,
and so on) with a recommended action. The bar chart shows segment sizes; the
table pairs each segment with its recommended action, reach, revenue, and
intent counts (product views, add-to-carts, checkouts). Use it to size an
audience before building a campaign or flow. The full segment list is in the
[Profile Explorer](/help/viewer/profile-explorer) article.

### 3. Customer Intelligence

Customer value rolled up from attributed sales: customers, total revenue,
average LTV, orders, subscribers, and active subscribers — plus a breakdown of
customer value by **acquisition channel and campaign**. Acquisition is based
on the customer's first attributed order, so subscription renewals stay
separate from acquisition and retention channels don't get credit for
acquiring customers they merely kept.

The "Customers by acquisition confidence" chart shows how sure Bratrax is
about where each customer came from — high, medium, low, or none.

### 4. Lead & Intent Signals

Non-buyer and anonymous behavior that can become retargeting, email-capture,
or Klaviyo follow-up: product interest (which products non-buyers browse),
add-to-carts, checkout events, and abandoned checkouts by segment. This is the
demand you've already paid for but haven't converted.

### 5. Attribution Diagnostics

Customers where revenue exists but sale evidence is weak or missing:
direct-only customers, direct orders, evidence-gap orders, and orders by
confidence tier — plus a review table of the specific customers involved. Blue
numbers throughout the dashboard are clickable and open the underlying list of
profiles or customers for inspection.
A growing evidence-gap number usually means a tracking problem (pixel
coverage, UTMs, consent banners), not a marketing problem.

## Common misreads

- **Anonymous profiles are devices, not verified people.** The same person on
  two devices counts twice until they identify.
- **Coverage counts are all-time by default.** The dashboard defaults to the
  full history; narrow the date range for recent behavior.
- **Acquisition = first attributed order.** Don't compare these channel
  numbers to a last-click report and expect them to match.
- **Reachable ≠ subscribed.** Reachable means an email/phone exists on the
  profile; your ESP's consent status governs what you may send.
- **Direct is not a channel.** A large Direct/evidence-gap share is a signal
  to fix tracking, not evidence that ads don't work.

## Drilldown moves

- Low known/reachable share → improve email capture and checkout identification.
- A segment with high intent but low reach → run capture before retargeting.
- High-LTV customers concentrated in one channel → shift budget or build
  lookalikes from that channel's customers.
- Rising evidence-gap orders → open the review table, then check pixel and
  UTM coverage on the affected orders.

## Good Claude questions

- "What share of my profiles are known and reachable?"
- "Which acquisition channel produces the highest average LTV?"
- "Which segments have the most reachable profiles with checkout intent?"
- "How many customers have low or no acquisition confidence?"

## Related

- **Profile Explorer** — the person-level directory built from the same
  profile graph, with workflow queues for acting on individual segments.
