<script lang="ts">
  import { page } from "$app/stores";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu";

  // Phase 1 of the nav restructure (NAV_RESTRUCTURE_HANDOFF.MD): consolidates
  // the configuration tabs (Connectors, Cost Settings, Settings) plus AI/Billing
  // sub-pages under a single SETTINGS ▾ trigger in Ribbon 1. The GETTING STARTED
  // top item is wired to /onboarding now; Phase 2 also adds the matching pill
  // in Ribbon 1 for incomplete-setup admins.
  //
  // Phase 4 polish: bind to the bits-ui open state so the trigger reads as
  // "active" while the menu is showing — gives the click an immediate visual
  // confirmation even when the user is on a non-settings surface.

  let menuOpen = false;

  $: pathname = $page.url.pathname;
  $: onSettingsPage =
    pathname.startsWith("/onboarding") ||
    pathname.startsWith("/connectors") ||
    pathname.startsWith("/cost-settings") ||
    pathname.startsWith("/settings");
  $: triggerActive = menuOpen || onSettingsPage;
</script>

<DropdownMenu.Root bind:open={menuOpen}>
  <DropdownMenu.Trigger
    class={triggerActive ? "settings-trigger active" : "settings-trigger"}
  >
    Settings
    <span class="chev">▾</span>
  </DropdownMenu.Trigger>
  <DropdownMenu.Content class="settings-dropdown-content p-1" sideOffset={4}>
    <DropdownMenu.Item href="/onboarding" class="settings-dropdown-item">
      Getting started
    </DropdownMenu.Item>
    <DropdownMenu.Separator />
    <DropdownMenu.Item href="/connectors" class="settings-dropdown-item">
      Connectors
    </DropdownMenu.Item>
    <DropdownMenu.Item href="/cost-settings" class="settings-dropdown-item">
      Cost settings
    </DropdownMenu.Item>
    <DropdownMenu.Separator />
    <DropdownMenu.Item
      href="/settings/account"
      class="settings-dropdown-item"
    >
      Account settings
    </DropdownMenu.Item>
    <DropdownMenu.Item href="/settings/team" class="settings-dropdown-item">
      Team settings
    </DropdownMenu.Item>
    <DropdownMenu.Item
      href="/settings/billing"
      class="settings-dropdown-item"
    >
      Billing
    </DropdownMenu.Item>
    <DropdownMenu.Separator />
    <DropdownMenu.Item href="/settings/ai" class="settings-dropdown-item">
      AI settings
    </DropdownMenu.Item>
    <DropdownMenu.Item href="/settings/mcp" class="settings-dropdown-item">
      MCP
    </DropdownMenu.Item>
    <DropdownMenu.Item href="/settings/slack" class="settings-dropdown-item">
      Slack
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>

<style lang="postcss">
  /* The trigger lives inside bits-ui's <button>, which sits outside Svelte's
     scoped-CSS tree for this component. :global() bypasses scoping so the
     class lands on the rendered button. Mirrors .bratrax-nav-link in the
     root +layout.svelte so the trigger reads identically to its neighbours. */
  :global(.settings-trigger) {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: transparent;
    border: none;
    cursor: pointer;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted);
    padding: 10px 12px;
    border-bottom: 2px solid transparent;
    transition:
      color 0.2s,
      border-color 0.2s,
      background-color 0.2s;
  }
  :global(.settings-trigger:hover) {
    color: var(--color-text-secondary);
  }
  /* Brightness-only active state — no acid (reserved for the section tabs). */
  :global(.settings-trigger.active) {
    color: var(--bratrax-text-headline, var(--color-text));
    border-bottom-color: var(--color-border-strong);
  }
  :global(.settings-trigger:focus-visible) {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }

  :global(.settings-dropdown-content) {
    background: var(--color-surface, var(--background));
    border: 1px solid var(--color-border-strong, var(--border));
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6);
    min-width: 220px;
  }

  /* Scoped under .settings-dropdown-content (specificity 0,2,0) so it beats
     bratrax-theme.css §15, which floors every .text-xs element (and bits-ui
     applies text-xs to each item) to 14px !important. Without the extra class
     the two !important rules tie on specificity and source order wins, leaving
     the items at 14px — visibly larger than the 11px Tier 1 links. Matches the
     Tier 1 nav size (11px / 1.5px tracking). */
  :global(.settings-dropdown-content .settings-dropdown-item) {
    padding: 10px 16px !important;
    font-family: "Space Mono", "JetBrains Mono", monospace !important;
    font-size: 11px !important;
    letter-spacing: 1.5px !important;
    font-weight: 700 !important;
    text-transform: uppercase !important;
    color: var(--color-text) !important;
    cursor: pointer;
  }

  :global(.settings-dropdown-content .settings-dropdown-item:hover) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06)) !important;
  }

  :global(.settings-trigger > .chev) {
    color: var(--color-text-muted);
    font-size: 10px;
    margin-left: 4px;
  }
</style>
