# Client switcher

`ClientSwitcher.svelte` is the header dropdown that lets a **super_admin** (or a
**multi-store** user) jump between clients without re-logging-in. It is mounted in
the application header's `header-extras` slot from `web-local/src/routes/+layout.svelte`,
gated by `isSuper || isMultiStore`, and sits next to `AddStoreButton` and the theme
toggle.

It is a **custom popover, not a native `<select>`** — a native select can't host a
search field, and super_admins routinely have dozens of clients with similar names.
The popover carries a search box on top and a two-line row per client (company name +
admin email) so the list stays usable at that scale.

## How a switch works

```
listClients ─▶ render dropdown ─▶ user picks a client ─▶ switchClient(id) ─▶ window.location.href = "/"
```

- `bratraxListClients()` (`GET /bratrax/auth/clients`) returns
  `{ clients: BratraxClientSummary[], active_client_id }`. Each client is
  `{ client_id, company_name, admin_email? }`. See `auth.ts`.
- `bratraxSwitchClient(id)` (`POST /bratrax/auth/switch-client`) flips the
  `bratrax_active_client` cookie and persists `last_client_id` server-side.
- The switch is followed by a **hard reload to `/`**, not client-side navigation.
  The Go proxy rebuilds the per-client Rill instance (DuckDB cache, metrics views,
  dashboards) lazily off the cookie, so a full reload is what re-inits the runtime
  stores cleanly — same effect as a fresh login. Do not replace this with SvelteKit
  `goto`.
- Selecting the **already-active** client is a no-op: the popover just closes, no
  reload.
- If `bratraxListClients()` fails, the load error is swallowed and the component
  renders a disabled em-dash (`—`) so the rest of the header still renders.

## UI

**Trigger** — a 32px button matching `.bratrax-add-store-btn` / `.bratrax-theme-toggle`
(neutral border, acid focus ring). Shows a building glyph, the active client's name
(ellipsized), and a chevron that rotates 180° when open. During a switch it shows
`Switching…` and is disabled.

**Panel** — sharp-edged, right-aligned under the trigger, with the design system's
4px acid top bar (`::before`). Top to bottom:

1. **Search** (`Search clients…`) — autofocuses on open; filters case-insensitively
   across both `company_name` and `admin_email`.
2. **List** — one row per client. The company name renders in Outfit 14px/600 (easy to
   read), the admin email below it in muted Space Mono 11px (disambiguates similar
   names). The matched substring of the current query is wrapped in an acid `<mark>`
   in both the name and the email. The active client's row gets an acid-mid background,
   a 2px acid left border, and a check mark. Empty filter shows a `No clients match …`
   state.
3. **Footer** — client count (`N / total clients` while filtering, else `N clients`).

## Keyboard & focus

- Trigger: `Enter` / `Space` / `ArrowDown` opens it.
- Search field: `↑` / `↓` move the highlight (wraps, scrolls into view), `Enter`
  switches to the highlighted client, `Esc` closes and refocuses the trigger.
- Mouse hover tracks the highlight (standard combobox behaviour).
- Click-outside (`svelte:window` click) and global `Esc` close the popover.

## Theming & design tokens

Everything reads from the Bratrax design-system tokens in
`web-local/src/bratrax-theme.css`, so light/dark are handled by the tokens flipping —
only the popover's drop shadow is overridden under `:global(.dark)`.

| Concern            | Token / rule                                                  |
| ------------------ | ------------------------------------------------------------- |
| Surfaces           | `--color-elevated` (panel), `--color-border-strong`, `--color-border` |
| Text               | `--color-text`, `--color-text-secondary`, `--color-text-muted` |
| Accent             | `--color-acid` (top bar, match, active border + focus ring), `--color-acid-dim` (hover), `--color-acid-mid` (active row) |
| Type               | Space Mono caps for the trigger/footer; Outfit for the readable client name |

### Gotchas

- **The search input inherits global input styling.** `bratrax-theme.css` §13 forces
  `background-color` / `border-color` and the acid focus ring on every `input` with
  `!important`. That's why the search field picks up the acid focus automatically — and
  why its border color can't be restyled locally without fighting `!important`.
- **Sharp edges are global.** `bratrax-theme.css` sets `border-radius: 0 !important`
  app-wide; don't add radii expecting them to take.
- **Match highlight uses solid acid + black text** (`#0A0A0A`) on purpose, so it stays
  legible both on the translucent acid-mid active row and in dark mode.
- **`z-index: 90`** sits above dashboard content but below the `AddStoreButton` modal
  (`z-[100]`); keep that ordering if either changes.

## Files / API

| Symbol                                   | Where        | Role                                            |
| ---------------------------------------- | ------------ | ----------------------------------------------- |
| `ClientSwitcher.svelte`                  | this dir     | The popover component                           |
| `bratraxListClients()` / `bratraxSwitchClient()` | `auth.ts` | Data source + switch endpoint clients        |
| `BratraxClientSummary` / `BratraxClientList` | `auth.ts` | Response shapes                                 |
| `header-extras` slot                     | `routes/+layout.svelte` | Mount point (gated on `isSuper || isMultiStore`) |
