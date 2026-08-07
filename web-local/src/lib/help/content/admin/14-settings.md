---
title: Settings
audience: admin
order: 5
group: workspace
status: ready
---

# Settings

The **Settings** area is where workspace admins manage the account itself —
company details, the team, billing, and integrations. Open it from the
**SETTINGS** dropdown in the top header (visible to admins; viewers don't see
this menu).

```loom
id: aeca0d86c03641ee959a15ca745d6f8b
label: Settings
```

Settings is **not** where costs live. COGS, shipping, gateway fees, custom
expenses, and ad-spend scoping have their own page — **Cost settings**, also in
the SETTINGS dropdown — open it at
[bratrax.com/cost-settings](https://bratrax.com/cost-settings).

## The six tabs

### Account

Company name, your email, and the workspace **timezone**. The timezone drives
how dashboard days are bucketed, so set it to match your store's timezone.
Company name and timezone are editable by admins only.

### Team

Everyone with access to the workspace, with their role (**admin** — full
access including settings; **viewer** — dashboards only).

- **Invite teammate** — enter their email and role, and Bratrax generates a
  secure invite link with an expiry. Copy the link and share it yourself; the
  invitee sets their password when accepting.
- Change a member's role inline, or remove them (with confirmation).
- Pending invitations are listed with **Copy link** and **Revoke**.

You can't edit your own role or the workspace owner's.

### Billing

Read-only summary of your plan: price, billing interval, status, and current
period end. **Manage Subscription** opens the external billing portal where
you can update payment details or cancel. See
[Billing & subscription](/help/admin/billing).

### AI

Bring your own Anthropic API key to power Claude chat inside Bratrax. When you
save a key, Bratrax verifies it with a live test call before storing it.
Removing the key disables Claude chat for the whole workspace until a new one
is added. See [AI chat & Anthropic API key](/help/admin/ai-and-api-keys).

### MCP

Connect Claude Desktop (or any MCP client) to your Bratrax data:

- **Generate token** creates a workspace MCP token granting access to the
  full Bratrax tool set.
- The tab shows a ready-to-paste `claude_desktop_config.json` block with a
  copy button, plus instructions for where the file lives.
- **Regenerate** or **Revoke** the token at any time — anyone holding the
  token can query your data, so treat it like a password.

### Slack

Install the `@bratrax` Slack assistant into your Slack workspace so your team
can ask data questions without leaving Slack. Connect via Slack OAuth, create
or link a channel, and disconnect at any time. Connecting is admin-only.

## Roles in short

| Action | Admin | Viewer |
|---|---|---|
| See the SETTINGS menu | ✔ | ✘ |
| Edit company name / timezone | ✔ | ✘ |
| Invite / remove teammates, change roles | ✔ | ✘ |
| Manage billing | ✔ | ✘ |
| Manage AI key, MCP token, Slack | ✔ | ✘ |
| Change their own email | ✔ | ✔ (via account) |

## Notes

- **Passwords** are set when accepting an invite. To change a forgotten
  password, use **Forgot password?** on the login page — there is no password
  field inside Settings.
- **Multi-store accounts** get a client switcher in the header (separate from
  Settings) to jump between stores; each store has its own Settings.
