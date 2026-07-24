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
