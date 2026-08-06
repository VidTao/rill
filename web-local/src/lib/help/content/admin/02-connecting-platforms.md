---
title: Connecting platforms
audience: admin
order: 2
group: workspace
status: ready
---

# Connecting platforms

Connect ad platforms, your store, and other data sources from the **Connectors** tab in the top nav.

**Video walkthrough**

```loom
d3631b9edef54f58abed0cef24de3dfb
```

## Adding a platform

1. Click **Connectors** in the top nav.
2. Click the platform you want to add.
3. Sign in to that platform when redirected.
4. Pick which accounts, ad accounts, or properties to include.
5. Bratrax starts pulling data immediately.

## How long until data shows up

| Platform | First data | Ongoing refresh |
|---|---|---|
| Store (Shopify, WooCommerce) | Recent orders in minutes; ~1 year backfilled over hours | Hourly |
| Ad platforms (Meta, Google, TikTok, Microsoft/Bing, Pinterest, Taboola, Outbrain) | 15–60 min to start; ~60 days backfilled over hours to days | Every 4 hours |
| Email / SMS (Klaviyo, Bloomreach) | 30–60 minutes | Every 4 hours |

You'll see progress on the connector card while the backfill is running.

## What each platform contributes

| Platform | What you get |
|---|---|
| Shopify | Orders, customers, products, sessions — including Shopify-native subscriptions |
| WooCommerce | Orders, customers, products (via the Bratrax WordPress plugin) |
| Meta (Facebook / Instagram) | Spend, impressions, clicks, conversions per campaign / ad set / ad |
| Google Ads | Same as Meta, plus search keyword data |
| TikTok Ads | Same as Meta |
| Microsoft (Bing) Ads | Same as Meta |
| Pinterest Ads | Same as Meta |
| Taboola | Spend, impressions, clicks, conversions per campaign |
| Outbrain | Spend, impressions, clicks, conversions per campaign |
| Klaviyo | Email/SMS campaigns & flows, opens, clicks, attributed revenue |
| Bloomreach | Email/SMS & engagement campaigns, attributed revenue |
| External landing pages / Funnelish | Orders + visitor journeys via the Bratrax tracking pixel (no ad connector) |

## Reconnecting an expired token

OAuth tokens expire periodically — Shopify after 60 days, others between 30 and 90 days. When that happens:

1. The connector card turns yellow with a "Reconnect" prompt.
2. Click **Reconnect** and sign in to the platform again.
3. Backfill resumes from where it left off.

You don't lose any historical data when reconnecting.

## Disconnecting

Click the platform card and select **Disconnect**. Historical data stays in your warehouse — only future updates stop. To remove the data entirely, contact support.

## Resuming an interrupted setup

In most cases you don't need to do anything special — the next time you log in, Bratrax drops you straight onto the screen that needs your attention. The product remembers where you left off.

To check the state of your setup manually:

- **Which integrations are connected and which are still pending:** [bratrax.com/connectors](https://bratrax.com/connectors)
- **Account-level settings (API keys, profile, billing):** [bratrax.com/settings](https://bratrax.com/settings)

Once your last integration finishes connecting, dashboards populate within about 10 minutes. If a step won't move forward, email [support@bratrax.com](mailto:support@bratrax.com) — a real person will get back to you, usually within a few hours on business days.

## When a connection doesn't behave

### My Facebook ad account isn't showing up in the connector

This almost always means the ad account lives in a Business Manager that isn't associated with the email you signed up to Bratrax with. Two ways to fix it:

1. **Search inside the connector.** Use the search field in the Facebook connection step to look for your ad account by name or ID.
2. **Add your Bratrax email to the ad account.** In Facebook Business Settings → Ad Accounts → People, add your Bratrax-connected email as a user on the account you want to track. Once added, return to Bratrax and re-trigger the Facebook connection — the account will appear.

If neither works, post in the Bratrax Slack #support channel with your ad account name (not the ID) and we'll dig in.

### Can I connect ad accounts across multiple Business Managers?

Yes. Bratrax pulls every Facebook ad account your connected login can access — across all of your Business Managers. There's no separate "add a Business Manager" step: when you connect Facebook, just select all the ad accounts you want to track, whichever BM they live in.

If an ad account doesn't appear, the Facebook login you connected with doesn't have access to it. Grant that login access in Business Settings → Ad Accounts → People, or reconnect with a login that already has access — then the account shows up for selection.

### I connected Google Ads but it doesn't show as connected

Give it 30–60 seconds and refresh the connectors page at [bratrax.com/connectors](https://bratrax.com/connectors) — the connection state can lag briefly after authorization.

If it still shows as disconnected after a refresh, email [support@bratrax.com](mailto:support@bratrax.com) and we'll trace it. It's usually a permissions or scope issue we can sort out quickly.

### My WooCommerce orders stopped syncing — is my firewall blocking Bratrax?

Possibly. If your WooCommerce store shows as connected but orders have stopped
syncing, your site's firewall may be blocking Bratrax's data requests. This
happens when Cloudflare or a WordPress security plugin sees our hourly order
pulls and blocks them as suspicious traffic. **It isn't a credentials problem** —
the API key you connected still works, so disconnecting and reconnecting won't
fix it. (This only affects WooCommerce/WordPress stores; Shopify and ad-platform
connections can't be blocked by your site's firewall.)

**The fix: allowlist our server's IP address at your firewall —
`135.181.142.189`.** Where to add it depends on what sits in front of your site:

- **Cloudflare** — Security → WAF → Tools → IP Access Rules → add
  `135.181.142.189` with the **Allow** action
- **WordPress security plugin** (Wordfence, iThemes Security, or similar) — add
  the IP to the plugin's allowlist
- **Server-level firewall** (cPanel, fail2ban, your hosting provider) — allow
  inbound HTTPS (port 443) requests from that IP

Once the allowlist is in place, syncing resumes automatically on the next
hourly run — nothing to reconnect. Not sure which firewall is doing the
blocking? Email [support@bratrax.com](mailto:support@bratrax.com) and we'll
help you identify it.
