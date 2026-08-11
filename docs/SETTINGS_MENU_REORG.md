# Settings dropdown reorganization

Single file: `web-common/src/layout/SettingsDropdown.svelte`

## Changes

1. **Move the separator down one item.** It currently sits between "Team settings" and
   "Billing"; it should sit between "Billing" and "AI settings" — grouping Billing with
   the account/team items rather than with the integration items.
2. **Add a "Slack" item** as the last entry, after MCP, pointing at `/settings/slack`.

## Resulting menu

```
Getting started
─────────────────
Connectors
Cost settings
─────────────────
Account settings
Team settings
Billing
─────────────────
AI settings
MCP
Slack          ← new
```

## Notes

- `/settings/slack` already exists — `slack` is a registered tab in
  `web-local/src/routes/settings/[tab]/+page.svelte` (`TabId` union + `TABS` array).
  No routing or backend work needed.
- Item labels are written in sentence case in source; the `.settings-dropdown-item`
  CSS uppercases them, so "Slack" renders as "SLACK".
- No role gating on the new item, matching the existing AI settings / MCP entries.
- Frontend-only change, but `web-local`'s build output is `//go:embed`-ed, so shipping
  it needs a Go rebuild (`rill-rebuild.sh` / `make cli`), not just a restart.
