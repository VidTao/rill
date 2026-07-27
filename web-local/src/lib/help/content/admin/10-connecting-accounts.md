---
title: Connecting accounts (FAQ)
audience: admin
order: 10
status: ready
---

# Connecting accounts (FAQ)

## My Facebook ad account isn't showing up in the connector. What do I do?

This almost always means the ad account lives in a Business Manager that isn't associated with the email you signed up to Bratrax with. Two ways to fix it:

1. **Search inside the connector.** Use the search field in the Facebook connection step to look for your ad account by name or ID.
2. **Add your Bratrax email to the ad account.** In Facebook Business Settings → Ad Accounts → People, add your Bratrax-connected email as a user on the account you want to track. Once added, return to Bratrax and re-trigger the Facebook connection — the account will appear.

If neither works, post in the Bratrax Slack #support channel with your ad account name (not the ID) and we'll dig in.

## Can I connect ad accounts across multiple Business Managers?

Yes. Bratrax pulls every Facebook ad account your connected login can access — across all of your Business Managers. There's no separate "add a Business Manager" step: when you connect Facebook, just select all the ad accounts you want to track, whichever BM they live in.

If an ad account doesn't appear, the Facebook login you connected with doesn't have access to it. Grant that login access in Business Settings → Ad Accounts → People, or reconnect with a login that already has access — then the account shows up for selection.

## I connected Google Ads but it doesn't show as connected on the dashboard. What now?

Give it 30–60 seconds and refresh the connectors page at [bratrax.com/connectors](https://bratrax.com/connectors) — the connection state can lag briefly after authorization.

If it still shows as disconnected after a refresh, email [support@bratrax.com](mailto:support@bratrax.com) and we'll trace it. It's usually a permissions or scope issue we can sort out quickly.

## My WooCommerce orders stopped syncing — is my firewall blocking Bratrax?

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
