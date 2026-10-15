---
title: Cost settings & COGS
audience: admin
order: 3
group: workspace
status: ready
---

# Cost settings

Your profit metrics — contribution margin, MER, net profit, true ROAS — are only
as accurate as the costs you feed them. **[Cost Settings](https://bratrax.com/cost-settings)**
is where those costs live — open it from the **SETTINGS** dropdown in the top
header. Admin-only.

It's a set of tabs, each feeding a different part of the profit calculation. You
don't have to fill in everything at once — the more complete your costs, the more
accurate your margins, but partial data still produces useful numbers.

## The seven tabs

### Cost of Goods
Per-product / per-SKU product costs. On **Shopify** these are **auto-filled**
from the cost-per-item on your products, so if you already keep costs in Shopify
there's nothing to enter. On **WooCommerce**, enter them manually on the **Cost of Goods** tab of
[Cost Settings](https://bratrax.com/cost-settings) — once entered,
each product's cost is counted on every order that contains it and flows straight
into your profit numbers. Don't have SKU-level costs yet? Set a **single global percentage** (e.g. COGS = 35% of
revenue) instead. Your manual edits always take priority over auto-filled values.

### Amazon COGS
Per-product costs for your **Amazon Seller Central** catalogue. This tab appears
once Amazon Seller Central is connected, and is kept separate from your
Shopify/Woo COGS because the catalogues — and often the costs — differ.

### Shipping
How your shipping cost is figured. Either **use what the customer was charged**
(shipping nets out), or set a **flat per-order cost**, with optional per-region
overrides.

### Gateway Costs
Payment-processing fees per payment method, as a percentage plus a fixed fee.
These come off every order that used that method and flow into your profit
numbers.

- **Shopify:** your gateways (Shopify Payments, PayPal, and others) are listed from
  your transactions.
- **WooCommerce:** every payment method your store has taken an order with is listed
  — including ones like cash on delivery — under the name your store gives it, with
  an **Orders** column showing how many orders used it. Enter a percentage and a
  fixed fee for each one you want counted, on the **Gateway Costs** tab of
  [Cost Settings](https://bratrax.com/cost-settings).

Fees you've saved stay listed even if a method has no recent orders. The list fills
in once your store has orders.

### Custom Expenses
Operating costs that aren't tied to a single order: agency retainers, software,
salaries. Add them as recurring rules or one-time entries so they land in your
net-profit lines.

### Media Scope
Assigns what share of an ad account's spend belongs to this store. Most stores
can ignore it — leave it alone and 100% of each connected ad account counts. It
matters when **several stores share one ad account**: Media Scope splits that
spend by rule so each store carries only its share. See
[Multi-store accounts](/help/admin/multi-store).

### Profit Rules
How the pieces above combine into your profit metrics — which costs are
subtracted where. The defaults match how most D2C brands calculate contribution
margin and net profit; adjust only if your accounting differs.

## Where you'll see the effect
These inputs power the profit columns across your dashboards — contribution
margin, MER, and blended ROAS on
[Store Performance](https://bratrax.com/canvas/performance_overview),
campaign-level profitability on
[Attribution](https://bratrax.com/canvas/campaign_deep_dive), and product margins
on [Products](https://bratrax.com/canvas/product_performance). Saving a change
rebuilds the affected numbers in the background — give it a few minutes to flow
through.
