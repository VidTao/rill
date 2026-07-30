---
title: Metric Trees
audience: admin
order: 15
status: ready
---

# Metric Trees

**Metric Trees** (in the top nav) turns your top-line KPI into a living
decision tree: the output metric at the root, the drivers that produce it
underneath, the **levers** your team owns, the **surfaces** where you can act,
and the concrete **experiments** you're running at the leaves. Each node pulls
its live value from your dashboards' metrics, so the tree shows not just how
revenue decomposes, but which branch actually moved.

**Video walkthrough**

```loom
f7aad7273a09467fbb60b17e916a1ee0
```

## The anatomy of a tree

Nodes come in six types, from root to leaf:

1. **Output** — the metric you're trying to grow (e.g. Total revenue).
2. **Revenue stream** — a slice of the output (e.g. One-time sales,
   Subscriptions).
3. **Driver** — an intermediate metric (Orders, AOV, Traffic, Conversion rate).
4. **Lever** — a driver someone is accountable for, with an owner, a
   baseline, a target, and a target date.
5. **Surface** — a place to act (Checkout, Product page, Cart, Discovery).
6. **Experiment** — a specific test with a hypothesis, an ICE score
   (impact / confidence / ease), a status, and a measurement binding.

Parents carry a formula over their children (multiply, add, subtract, ratio),
so values roll up: `Total revenue = One-time sales + Subscriptions`,
`One-time sales = Orders × AOV`, `Orders = Traffic × Conversion rate`, and
so on.

## Getting a tree

Use **Seed template** to instantiate a starter tree. Templates include a
full D2C hybrid tree (one-time + subscription revenue), plus focused trees
for email revenue, paid growth, subscriptions, retention, and margin/profit.
Seeding copies the template into your workspace; from there every node is
editable — relabel, re-own, add children, or delete.

**Validate** checks that the tree is set up correctly and lists any errors or
warnings in the left panel — run it after restructuring nodes or formulas.

## Working in the tree

- **Time presets** — Last 7 / 30 / 90 days or Year to date. Node values and
  the prior-period comparison recompute for the selected window.
- **Click a node** to open its detail panel: value, formula, owner, and (for
  levers) baseline → target progress; (for experiments) hypothesis, ICE
  score, status, and evidence.
- **Ownership** — assign owners to levers and experiments; the **My work**
  view filters the tree to what you own.
- **Live values** — nodes bind to the same metrics views that power your
  dashboards, so the tree and the dashboards never disagree.

## Review mode

Toggle from **Canvas** to **Review** for a guided "what moved the number"
walkthrough: it starts at the root, descends into the biggest-moving child
until it reaches an owned lever or experiment, and ranks levers and
experiments by projected impact on the root. From there you can log
decisions, update lever plans, record experiment readouts with evidence, and
save the whole thing as a review session to share or revisit.

## Roles

Admins can edit trees — seed templates, edit nodes, manage experiments.
Viewers who open the page see the tree read-only.

## How this differs from a dashboard

Dashboards tell you *what happened*. A metric tree encodes *how your team
believes the business works* — and holds that belief accountable by wiring
every node to live data and every leaf to a testable experiment. Use it for
weekly growth reviews: pick the time preset, run Review mode, and let the
tree point at the lever that actually moved.

## Good Claude questions

- "Which branch of the revenue tree moved most in the last 30 days?"
- "Which levers are behind their targets?"
- "What experiments are queued on the checkout surface?"
- "Draft a hypothesis for improving conversion rate on the product page."
