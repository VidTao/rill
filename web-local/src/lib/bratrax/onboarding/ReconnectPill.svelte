<script lang="ts">
  import { bratraxUser, bratraxNeedsReconnect } from "$lib/bratrax/auth-store";
  import { platformDisplayName } from "./api";

  // Global CTA for a dead connector token. Sits in Ribbon 1's "continue-setup"
  // slot beside ContinueSetupPill, so it shows on every authenticated page and
  // the merchant can't miss that their data has stopped flowing.
  //
  // A direct-extract script sets needs_reconnect on the platform's credentials
  // when it hits an auth error; the flag reaches us through /onboard/me →
  // bratraxNeedsReconnect, which the root layout guard refreshes on every
  // navigation. So unlike ContinueSetupPill this component does NO fetching of
  // its own.
  //
  // Visibility rules:
  //   - role is "admin" or "super_admin". Viewers are excluded — they cannot
  //     reconnect anything, so it would be pure noise. super_admin IS included
  //     (unlike ContinueSetupPill, which is about a client's own setup): a
  //     broken pipeline on the workspace they're acting on is support-relevant.
  //   - at least one connector is flagged.
  //
  // Not rendered inside the Shopify admin iframe, because /embed/* suppresses
  // ApplicationHeader entirely. That's correct — an OAuth popup can't run there.

  $: role = $bratraxUser?.role ?? null;
  $: canAct = role === "admin" || role === "super_admin";
  $: platforms = $bratraxNeedsReconnect ?? [];
  $: visible = canAct && platforms.length > 0;

  // Name the connector when there's exactly one — "Facebook Ads needs
  // reconnect" is actionable in a way that a bare count is not. Past one, the
  // header has no room for a list.
  $: label =
    platforms.length === 1
      ? `${platformDisplayName(platforms[0])} needs reconnect`
      : `${platforms.length} connectors need reconnect`;
</script>

{#if visible}
  <a
    href="/connectors"
    class="reconnect-pill"
    title="{platforms.map(platformDisplayName).join(', ')} — click to reconnect"
  >
    <!-- Square by design: tailwind.config.ts forces borderRadius 0 on every
         key, so rounded-* utilities are inert throughout this app. -->
    <span class="reconnect-dot" aria-hidden="true"></span>
    {label} →
  </a>
{/if}

<style lang="postcss">
  /* Danger colouring, not the acid CTA treatment — this is a fault the user has
     to repair, not an invitation. --bratrax-tomato flips per theme
     (#E63226 light / #FF3B30 dark) via bratrax-theme.css. */
  .reconnect-pill {
    @apply inline-flex items-center;
    @apply px-3.5 py-2;
    gap: 7px;
    background: var(--bratrax-tomato);
    color: #ffffff;
    /* font-size MUST be set here, not via a Tailwind text class:
       bratrax-theme.css section 15 floors every sub-14px text utility
       (.text-xs, .text-[11px], .text-2xs, …) at 0.875rem with !important, which
       would blow out the header's 3.25rem height. Same reason
       .continue-setup-pill and .role-badge hard-code theirs. */
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    white-space: nowrap;
    text-decoration: none;
    cursor: pointer;
    transition: opacity 120ms ease;
  }
  .reconnect-pill:hover {
    opacity: 0.9;
  }
  .reconnect-pill:focus-visible {
    outline: 2px solid var(--bratrax-tomato);
    outline-offset: 2px;
  }

  /* Only the dot blinks. This pill is on every page, so pulsing the whole
     label would make the entire header restless. Timing/easing copied from
     ChecklistItem.svelte's .pulse, the house convention. */
  .reconnect-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    background: #ffffff;
    animation: reconnect-blink 1.2s ease-in-out infinite;
  }
  @keyframes reconnect-blink {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.15;
    }
  }
  /* First always-on animation in the global header, so it's also the first
     thing in this codebase that has to honour the OS setting. */
  @media (prefers-reduced-motion: reduce) {
    .reconnect-dot {
      animation: none;
    }
  }
</style>
