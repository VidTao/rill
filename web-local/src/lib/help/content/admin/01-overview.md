---
title: Admin overview
audience: admin
order: 1
group: building
status: ready
---

# Admin overview

As an admin you can do everything a viewer can, plus build and modify the things that power dashboards. This page is the starting point for what's under your control and what isn't.

## What admins can do

- **Connect platforms** — Shopify, Facebook, Google, TikTok, Klaviyo and more, from the Connectors tab.
- **Edit dashboards** — change layout, add tiles, build new dashboards from scratch.
- **Define metrics** — declare which numbers are available and how they're calculated.
- **Tune Claude** — give the AI analyst business context so it answers questions correctly.

## The three authoring folders

Open the file tree on the left. You'll see three folders you can edit:

- **`/dashboards/`** — one YAML file per dashboard. This is the layout viewers see.
- **`/metrics/`** — one YAML file per metrics view. Declares dimensions and measures.
- **`/knowledge/`** — text files used to give Claude business context.

The `/sources/` folder is also visible but is read-only — those files are auto-generated when you connect a platform.

You won't see a `/models/` folder. The SQL that prepares your data — joining orders
to touchpoints, rolling up campaign spend, building the attribution tables — is
maintained by Bratrax in the warehouse rather than authored per workspace. Metrics
views read those prepared tables directly.

## What you can't change

- **The underlying schema** — the objects (User, Order, Campaign) and how they relate. This is templated and managed by Bratrax.
- **Raw data ingestion** — how data flows from platforms into your warehouse.
- **User management** — adding or removing users is handled by the Bratrax team.

If you need any of those changed, contact support.

## How a saved change goes live

There's no separate publish step. When you save a metric or dashboard file:

1. The file validates immediately. Errors appear inline.
2. If valid, it goes live for everyone with access.
3. Viewers see the change on their next page load.

For larger or risky changes, do them in a quieter window and click around as a viewer would before walking away.

## Where to go next

- **Connecting platforms** — add a data source
- **Models, Metrics views, Canvas dashboards** — build new things
- **Best practices** — naming and performance conventions that keep things tidy
