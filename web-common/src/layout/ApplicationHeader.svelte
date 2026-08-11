<script lang="ts">
  import { page } from "$app/stores";
  import LocalAvatarButton from "@rilldata/web-common/features/authentication/LocalAvatarButton.svelte";
  import CanvasPreviewCTAs from "@rilldata/web-common/features/canvas/CanvasPreviewCTAs.svelte";
  import SupportChatToggle from "@rilldata/web-common/features/chat/layouts/sidebar/SupportChatToggle.svelte";
  import {
    chatOpen,
    sidebarActions,
  } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";
  import HeaderToggleButton from "./HeaderToggleButton.svelte";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import ExplorePreviewCTAs from "@rilldata/web-common/features/explores/ExplorePreviewCTAs.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags.ts";
  import { isDeployPage } from "@rilldata/web-common/layout/navigation/route-utils";

  const { developerChat } = featureFlags;

  export let mode: string;
  export let externalUser: {
    email: string;
    name: string;
    role?: string;
  } | null = null;
  export let onLogout: (() => void) | null = null;
  export let onboardingLink: { href: string; label: string } | null = null;
  /**
   * Hide the AI and support toggles. Set by web-local when running inside the
   * Shopify admin iframe: both open slide-over panels, which don't fit beside
   * Shopify's own chrome in ~1150px. Defaults false, so web-admin and the
   * normal web-local app are unchanged.
   */
  export let hideAssistants = false;

  // If externalUser has a role, show a friendly role label instead of the dev/preview mode.
  $: roleLabel =
    externalUser?.role === "super_admin"
      ? "Super Admin"
      : externalUser?.role === "admin"
        ? "Administrator"
        : externalUser?.role === "viewer"
          ? "Viewer"
          : null;
  // Role-specific badge color class (Phase 4 of NAV_RESTRUCTURE_HANDOFF.MD):
  // Admin = acid green, Super Admin = white, Viewer = muted gray. The
  // background stays neutral; only the label color changes.
  $: roleClass =
    externalUser?.role === "admin"
      ? "role-admin"
      : externalUser?.role === "super_admin"
        ? "role-super"
        : externalUser?.role === "viewer"
          ? "role-viewer"
          : "role-neutral";

  // Drives the Edit button in CanvasPreviewCTAs. Viewers don't see it.
  $: isAdminOrSuper =
    externalUser?.role === "admin" || externalUser?.role === "super_admin";

  $: ({
    params: { name: dashboardName },
    route,
  } = $page);

  // Bratrax theme toggle — sun/moon button left of the AI/ChatToggle.
  // themeControl.preference is the persisted preference store (light/dark/system).
  $: themePreference = themeControl.preference;
  $: onDeployPage = isDeployPage($page);
  $: showDeveloperChat = $developerChat && !onDeployPage;
</script>

<header class:border-b={!onDeployPage} class="bg-surface-base">
  {#if !onDeployPage}
    <a href="/" class="bratrax-logo-link">
      <img
        src="/img/bratrax/bratrax-logo-dark.svg"
        alt="Bratrax"
        class="bratrax-logo bratrax-logo-light-mode"
      />
      <img
        src="/img/bratrax/bratrax-logo-light.svg"
        alt="Bratrax"
        class="bratrax-logo bratrax-logo-dark-mode"
      />
    </a>

    <span class="role-badge {roleClass}">{roleLabel ?? mode}</span>
  {/if}

  <!-- Primary nav (e.g. DASHBOARDS link): top-level destinations rendered
       next to the role tag so the user has a constant anchor back to the
       dashboards view from any surface (Settings, Customize, etc.). -->
  <div class="primary-nav-slot">
    <slot name="primary-nav" />
  </div>

  <div class="right-cluster ml-auto flex h-full items-center">
    <!-- Bratrax header-extras (ClientSwitcher for super_admins, AddStoreButton
         for multi-store) render in both preview and dev/edit modes. -->
    <slot name="header-extras" />

    <!-- CONTINUE SETUP pill: conditional CTA for admins with incomplete
         onboarding. Sits between the client switcher and SETTINGS ▾, matching
         the NAV_RESTRUCTURE_HANDOFF mockup. -->
    <slot name="continue-setup" />

    <!-- Bratrax inline nav items (SETTINGS ▾, HELP, SUPERADMINS, CLIENTS) slot
         in from the root layout, sitting between the client switcher and the
         theme/AI/avatar cluster. The 22px gap matches the NAV_RESTRUCTURE_HANDOFF spec. -->
    <div class="nav-tabs-slot">
      <slot name="nav-tabs" />
    </div>

    <div class="icon-cluster flex gap-x-2 h-full items-center py-2">
      {#if mode === "Preview"}
        {#if route.id?.includes("explore")}
          <ExplorePreviewCTAs exploreName={dashboardName} />
        {:else if route.id?.includes("canvas")}
          <CanvasPreviewCTAs {isAdminOrSuper} />
        {/if}
      {:else if showDeveloperChat}
        <!-- No active state: this switches the theme rather than opening a
             panel, so it stays muted and acid keeps meaning "panel open". -->
        <HeaderToggleButton
          onClick={() =>
            themeControl.set[$themePreference === "dark" ? "light" : "dark"]()}
          title={$themePreference === "dark"
            ? "Switch to light theme"
            : "Switch to dark theme"}
          aria-label="Toggle theme"
        >
          {#if $themePreference === "dark"}
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <circle cx="12" cy="12" r="4" />
              <path
                d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
              />
            </svg>
          {:else}
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          {/if}
        </HeaderToggleButton>
        {#if !hideAssistants}
          <HeaderToggleButton
            active={$chatOpen}
            onClick={sidebarActions.toggleChat}
            title="Ask AI"
            aria-label="Ask AI"
            aria-pressed={$chatOpen}
          >
            AI
          </HeaderToggleButton>
        {/if}
      {/if}
      <!-- "?" support-bot toggle: unlike the AI toggle it renders in every
           mode and for every role — product help is not gated on a BYOK key.
           Both are suppressed inside the Shopify admin iframe, where they'd be
           slide-over panels competing for ~1150px with Shopify's own chrome. -->
      {#if !hideAssistants}
        <SupportChatToggle />
      {/if}
      <LocalAvatarButton {externalUser} {onLogout} {onboardingLink} />
    </div>
  </div>
</header>

<style lang="postcss">
  header {
    @apply w-full box-border;
    @apply flex items-center px-7 flex-none;
    @apply h-[3.25rem];
    gap: 22px;
  }

  /* The utility bar reads as a quiet plane behind the louder section tabs:
     a hair darker than the page (#080808 vs #0A0A0A) so it recedes, plus a
     1px neutral hairline so Tier 1 and Tier 2 read as two distinct planes.
     Dark theme only — light theme mirrors the logic in a later pass. */
  :global(.dark) header {
    background: #080808;
    border-bottom: 1px solid var(--color-border-strong);
  }

  .nav-tabs-slot,
  .primary-nav-slot {
    @apply flex items-center h-full;
    gap: 22px;
  }

  /* Right cluster: ClientSwitcher · nav-tabs · icon-cluster (theme/AI/avatar).
     22px gap between the three groups; icons within icon-cluster stay tight. */
  .right-cluster {
    gap: 22px;
  }

  .bratrax-logo-link {
    @apply flex items-center;
  }

  .bratrax-logo {
    height: 1.5rem; /* 24px */
    width: auto;
    display: block;
  }

  /* Light mode (default): show the dark-text logo, hide the light-text one */
  .bratrax-logo-dark-mode {
    display: none;
  }

  /* Dark mode: show the light-text logo, hide the dark-text one */
  :global(.dark) .bratrax-logo-light-mode {
    display: none;
  }

  :global(.dark) .bratrax-logo-dark-mode {
    display: block;
  }

  /* Role badge: same neutral chip background across roles, label color
     differs per role (NAV_RESTRUCTURE_HANDOFF.MD §"Visual notes"). */
  .role-badge {
    display: inline-flex;
    align-items: center;
    padding: 3px 8px;
    font-family: "Space Mono", "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
  }
  /* Admin uses --color-acid-text (the theme-aware acid-as-text token: dark
     olive in light theme for AA contrast on cream, full acid in dark theme
     on near-black). Plain --color-acid would render yellow-on-cream in light
     mode. Super/Viewer use --color-text + --color-text-muted which already
     flip with the theme. */
  .role-badge.role-admin {
    color: var(--color-acid-text);
  }
  .role-badge.role-super {
    color: var(--color-text);
  }
  .role-badge.role-viewer,
  .role-badge.role-neutral {
    color: var(--color-text-muted);
  }
</style>
