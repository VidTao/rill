<script lang="ts">
  import { page } from "$app/stores";
  import { useCanvas } from "@rilldata/web-common/features/canvas/selector";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { featureFlags } from "../feature-flags";
  import { themeControl } from "../themes/theme-control";
  import ChatToggle from "@rilldata/web-common/features/chat/layouts/sidebar/ChatToggle.svelte";

  // Driven by Bratrax role from the parent ApplicationHeader. Plain admin and
  // super_admin see the Edit button; viewers do not.
  export let isAdminOrSuper: boolean = false;

  const { dashboardChat } = featureFlags;

  $: ({ instanceId } = $runtime);
  $: canvasName = $page.params.name ?? "";
  $: canvasQuery = canvasName ? useCanvas(instanceId, canvasName) : null;
  $: canvasFilePath = $canvasQuery?.data?.filePath ?? "";
  $: themePreference = themeControl.preference;
</script>

<button
  type="button"
  on:click={() =>
    themeControl.set[$themePreference === "dark" ? "light" : "dark"]()}
  class="bratrax-theme-toggle"
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
</button>

{#if isAdminOrSuper && canvasFilePath}
  <a href={`/files${canvasFilePath}`} class="bratrax-edit-link">Edit</a>
{/if}

{#if $dashboardChat}
  <ChatToggle />
{/if}

<style lang="postcss">
  .bratrax-theme-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease, border-color 120ms ease;
  }
  .bratrax-theme-toggle:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
  }
  .bratrax-theme-toggle:focus-visible {
    outline: 2px solid var(--color-acid, #D4FF00);
    outline-offset: 2px;
  }

  .bratrax-edit-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 32px;
    padding: 0 12px;
    background: transparent;
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 2px;
    text-decoration: none;
    cursor: pointer;
    transition: background-color 120ms ease, color 120ms ease, border-color 120ms ease;
  }
  .bratrax-edit-link:hover {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
    border-color: var(--color-acid, #D4FF00);
  }
  .bratrax-edit-link:focus-visible {
    outline: 2px solid var(--color-acid, #D4FF00);
    outline-offset: 2px;
  }
</style>
