<script lang="ts">
  import { featureFlags } from "../feature-flags";
  import { themeControl } from "../themes/theme-control";
  import HeaderChatToggle from "@rilldata/web-common/features/chat/layouts/sidebar/HeaderChatToggle.svelte";
  import HeaderToggleButton from "@rilldata/web-common/layout/HeaderToggleButton.svelte";

  // Bratrax-fork right cluster on canvas preview: theme toggle + AI chat
  // toggle. The Edit button used to live here but moved to the canvas
  // filter-bar in 2026-06-10 (mirrors edit-mode's Preview button placement)
  // — see /canvas/[name]/+page.svelte's `filters-extras` slot. The
  // isAdminOrSuper prop is left in place for forward-compat even though
  // unused today.
  export let isAdminOrSuper: boolean = false;
  // Suppress unused-prop warnings from svelte-check until we either drop it
  // or wire something else to it.
  $: void isAdminOrSuper;

  const { dashboardChat } = featureFlags;

  $: themePreference = themeControl.preference;
</script>

<!-- Same cluster as the header's own theme switch, so it shares its component.
     No active state: it switches the theme rather than opening a panel. -->
<HeaderToggleButton
  onClick={() =>
    themeControl.set[$themePreference === "dark" ? "light" : "dark"]()}
  title={$themePreference === "dark"
    ? "Switch to light theme"
    : "Switch to dark theme"}
  aria-label="Toggle theme"
>
  {#if $themePreference === "dark"}
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
         stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
  {:else}
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
         stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  {/if}
</HeaderToggleButton>

{#if $dashboardChat}
  <HeaderChatToggle />
{/if}
