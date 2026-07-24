---
title: Profile Explorer
audience: viewer
order: 8
status: ready
---

# Profile Explorer

**Profile Explorer** is a directory of every person Bratrax can see — paying
customers, known leads (people whose email or phone you have but who haven't
bought), and anonymous visitors. Bratrax stitches store data, email/SMS data,
and on-site activity into one deduplicated profile per person, then segments
each profile and suggests what to do with it.

**Video walkthrough**

```loom
05bc62dcba1f4cce8e6e08d2b2ca5bf1
```

## How profiles are built

- **Customers** are built from attributed sales — a profile becomes a customer
  when Bratrax can tie revenue to it.
- **Known leads and anonymous visitors** are built from activity — page views,
  add-to-carts, checkouts, email opens and clicks.
- Known profiles are keyed by email; anonymous profiles are keyed by a visitor
  identifier. When an anonymous visitor later identifies (checkout, email
  signup), their history merges into the known profile.

## Look first at

- **Profiles** — everyone Bratrax can see, across all three types.
- **Customers** — profiles with attributed purchase revenue.
- **Known Leads** — reachable people who have not bought yet.
- **Anonymous Profiles** — visitors with activity but no identity yet.
- **Reachable Profiles** — profiles you can actually contact (email/phone).
- **Revenue** — total revenue rolled up to profiles.

## The Profiles table

The main table groups profiles by **type → segment → profile**, with revenue,
orders, product views, email clicks, and evidence-gap orders per row. Expand a
segment to see the individual profiles inside it. Blue numbers are clickable —
they open the underlying list of profiles, and from there you can open an
individual profile's path through your store.

## Segments and recommended actions

Every profile lands in exactly one segment, each with a recommended action:

| Segment | Who is in it | Recommended action |
|---|---|---|
| `anonymous_high_intent` | Anonymous visitor with cart/checkout activity or repeat product views | Retarget and capture email |
| `abandoned_checkout` | Known non-buyer who reached checkout | Send abandoned checkout flow |
| `known_lead` | Known non-buyer, no checkout yet | Nurture in Klaviyo |
| `subscriber_active` | Customer with an active subscription | Protect retention and expansion |
| `subscriber_churned` | Customer whose subscription lapsed | Run subscriber winback |
| `first_time_buyer` | Customer with exactly one order | Drive second purchase |
| `direct_only_customer` | Customer whose orders all lack attribution evidence | Diagnose missing attribution evidence |
| `high_ltv_customer` | High-revenue customer | Build lookalike or VIP audience |
| `buyer_winback` | Customer with no order in 90+ days | Run buyer winback |
| `buyer` | Everyone else who has bought | Monitor value and repeat purchase |

Profiles with no meaningful signal fall into a low-intent bucket ("keep
observing").

## Workflow queues

Below the directory are two working views:

- **Profiles by segment** — segment × recommended action with reach and
  intent counts. Use this to size an activation play before building it.
- **Attribution review queue** — profiles whose revenue exists but whose
  acquisition evidence is weak (direct or evidence-gap orders). Use this to
  spot where tracking, not marketing, is the problem.

## Common misreads

- **Anonymous profiles are devices, not verified people.** One person on two
  devices can appear as two anonymous profiles until they identify.
- **Customers and leads are built differently.** Customer counts come from
  attributed sales; lead counts come from observed activity. Don't expect them
  to reconcile against a single platform export.
- **Reachable ≠ subscribed.** Reachable means Bratrax has an email or phone on
  the profile; your ESP's consent status still governs what you may send.
- **Evidence-gap orders are a tracking signal, not a marketing signal.** They
  mean the sale happened but the journey evidence was weak — start in the
  attribution review queue, not in your ad account.
- **The default view is all-time.** Narrow the date range if you want recent
  behavior only.

## Drilldown moves

- Big anonymous-high-intent segment → launch retargeting + email capture.
- Many abandoned-checkout profiles → check the abandoned-cart flow is firing.
- Growing direct-only customers → investigate pixel/UTM coverage.
- Large "known lead" pool with revenue stuck → review nurture flows in Klaviyo.

## Good Claude questions

- "Which segments hold the most reachable non-buyers?"
- "Show my highest-revenue profiles with evidence-gap orders."
- "How many anonymous high-intent visitors did we get this month?"
- "Which recommended actions would touch the most revenue?"

## Related

- **Commerce Profile Graph** — the aggregate view of the same profile data:
  coverage, segment sizes, customer value by acquisition, and attribution
  diagnostics.
