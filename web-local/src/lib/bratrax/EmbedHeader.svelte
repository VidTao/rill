<script lang="ts">
  import BratraxLogo from "@rilldata/web-common/layout/BratraxLogo.svelte";
  import HeaderToggleButton from "@rilldata/web-common/layout/HeaderToggleButton.svelte";
  import SupportChatToggle from "@rilldata/web-common/features/chat/layouts/sidebar/SupportChatToggle.svelte";
  import { reserveTopLevelTab } from "$lib/bratrax/shopify-embed";
  import { createEmbedHandoff } from "$lib/bratrax/onboarding/api";
  import { theme, toggleTheme } from "$lib/bratrax/theme";

  /**
   * Header for the Shopify admin embed, above the canvas filter bar.
   *
   * Deliberately not ApplicationHeader: that one carries dashboard tabs, the
   * client switcher and the avatar menu, none of which belong in a ~1150px
   * panel inside someone else's admin. The root layout suppresses it on
   * /embed/* (onEmbedPage) and this stands in — same shell dimensions and
   * colours, so the two read as one design system.
   *
   * It does carry the theme switch and the "?" support toggle, rebuilt here
   * from the same HeaderToggleButton cluster: those two are the merchant's
   * only controls once ApplicationHeader is gone, and neither depends on the
   * chrome the embed drops.
   *
   * It exists mainly to answer a question the embed otherwise leaves open: the
   * merchant sees exactly one dashboard and has no way to know the rest of the
   * product is there. Hence the centre line and the escape hatch on the right.
   */

  let opening = false;
  let openError = "";

  // "Open in Bratrax" — the escape hatch to the full app.
  //
  // Cannot be a plain <a href="/canvas/...">: inside the iframe the merchant is
  // authenticated by a Shopify App Bridge session token, and there is no
  // bratrax_auth cookie on this origin (that cookie is third-party here and
  // blocked). A new tab would land on /login. So we mint a single-use hand-off
  // token first and open the URL that exchanges it for a real cookie.
  async function openInBratrax() {
    opening = true;
    openError = "";

    // Reserve the tab synchronously — the click's popup grant does not survive
    // the await below, so opening afterwards would be blocked almost always.
    const tab = reserveTopLevelTab();

    try {
      const { url } = await createEmbedHandoff();
      if (tab.ok) {
        tab.navigate(url);
      } else {
        // Popup blocked outright. The token is single-use and expires in 60s,
        // so it can't be offered as a persistent link — ask for the click again.
        openError = "Allow pop-ups for this page, then try again.";
      }
    } catch {
      tab.close();
      openError = "Couldn't open Bratrax. Please try again.";
    } finally {
      opening = false;
    }
  }
</script>

<header class="embed-header">
  <div class="embed-header-left">
    <!-- Not a link: navigating the iframe to "/" would redirect into the full
         app shell inside Shopify's admin. See BratraxLogo. -->
    <BratraxLogo href={null} />
  </div>

  <!-- Names what's actually behind the button rather than restating what the
       merchant can already see. They know they're on one dashboard — the screen
       says so; what they can't know is that customer, product and email
       analytics are one click away.

       The count is deliberately conservative: the stack templates ship more
       canvases than this, several of them pilot or feature-gated, so 7 is the
       floor a merchant reliably gets rather than the template total. -->
  <p class="embed-header-message">
    This is 1 of 7 dashboards — unlock your customer journeys, product
    analytics, extra connectors &amp; more
    <!-- Same action as the button on the right, so the sentence itself is the
         escape hatch rather than a label pointing at one. Must be a <button>,
         not an <a href>: see openInBratrax — the URL is minted per click and
         dies in 60s, so there is nothing to put in an href. -->
    <button
      type="button"
      class="embed-header-cta"
      disabled={opening}
      on:click={openInBratrax}>in Bratrax →</button
    >
  </p>

  <div class="embed-header-right">
    {#if openError}
      <span class="embed-open-error">{openError}</span>
    {/if}

    <!-- No active state: this switches the theme rather than opening a panel,
         so it stays muted and acid keeps meaning "panel open". Mirrors the
         theme button in ApplicationHeader, which the embed suppresses. -->
    <HeaderToggleButton
      onClick={toggleTheme}
      title={$theme === "dark"
        ? "Switch to light theme"
        : "Switch to dark theme"}
      aria-label="Toggle theme"
    >
      {#if $theme === "dark"}
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

    <!-- The root layout mounts SupportChatSidebar on /embed/* for this button
         to drive; it overlays the canvas here rather than taking a column. -->
    <SupportChatToggle />

    <button
      type="button"
      class="embed-open-link"
      disabled={opening}
      on:click={openInBratrax}
    >
      {opening ? "Opening…" : "Open in Bratrax ↗"}
    </button>
  </div>
</header>

<style lang="postcss">
  /* Mirrors ApplicationHeader's shell (h-[3.25rem], px-7, bg-surface-base,
     border-b, and the darker #080808 plane in dark mode) so the embed reads as
     the same product rather than a detached widget. */
  .embed-header {
    @apply w-full box-border flex-none;
    @apply grid items-center px-7 border-b;
    @apply h-[3.25rem] bg-surface-base;
    /* Three tracks with equal side columns so the message is optically centred
       in the header regardless of how wide the logo or button happen to be. */
    grid-template-columns: 1fr auto 1fr;
    gap: 22px;
  }

  :global(.dark) .embed-header {
    background: #080808;
    border-bottom: 1px solid var(--color-border-strong);
  }

  .embed-header-left {
    @apply flex items-center justify-self-start;
  }

  .embed-header-right {
    @apply flex items-center justify-self-end;
    gap: 8px;
  }

  /* Reads as a deliberate callout, not a caption. At 13px in the secondary
     colour it disappeared into the band; the size bump plus the primary
     colour is what actually makes it legible, and the bold acid tail gives
     the eye somewhere to land and doubles as the link.

     Plain text flow, not flex: the CTA is an inline run inside the sentence,
     so it has to share the text's baseline and inter-word spacing. A flex
     container would make the copy and the CTA separate anonymous items and
     the space between them would have to be faked with `gap`. */
  .embed-header-message {
    @apply text-center;
    font-size: 16px;
    font-weight: 500;
    letter-spacing: 0.01em;
    color: var(--color-text, var(--fg-primary));
    white-space: nowrap;
  }

  /* A button that reads as a link — same handler as .embed-open-link, so the
     whole sentence is clickable, not just the control on the right. */
  .embed-header-cta {
    background: none;
    border: 0;
    padding: 0;
    margin: 0;
    font: inherit;
    font-weight: 700;
    vertical-align: baseline;
    cursor: pointer;
    /* --color-acid-text, NOT --color-acid: plain acid is the same #D4FF00 in
       both themes and fails contrast on the light theme's cream. The -text
       token resolves to dark olive there and full acid on near-black. Same
       split as HeaderToggleButton.active. */
    color: var(--color-acid-text, var(--color-acid, #d4ff00));
  }
  .embed-header-cta:hover:not(:disabled),
  .embed-header-cta:focus-visible {
    text-decoration: underline;
  }
  .embed-header-cta:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .embed-header-cta:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }

  /* Below this the logo and controls start crowding the message. The message is
     `nowrap`, so it pushes rather than reflows — and because grid items have
     `min-width: auto` the tracks refuse to shrink under their content, so an
     overrun overflows the header instead of squeezing. The cutoff therefore has
     to clear logo + message + controls + gaps + padding with room to spare.
     Measured at 16px: logo ~180 + message ~840 + controls ~208 (theme 32 +
     support 32 + button ~128 + gaps) + padding 56 + grid gaps 44 = ~1330.

     KNOWN LIMITATION: the Shopify admin content column is ~1150px, so this copy
     does NOT fit there and the message hides inside the iframe — the surface it
     was written for. Hiding is the safe failure (the alternative is an
     overflowing header), but it means the upsell only shows in a wide top-level
     window. Fixing it needs a layout change, not a bigger number: give the
     message its own full-width row under the header, or drop the wordmark below
     ~1200px to reclaim its ~200px. The button never hides — it is the only way
     out of the embed. */
  @media (max-width: 1360px) {
    .embed-header-message {
      display: none;
    }
  }

  .embed-open-error {
    font-size: 12px;
    color: var(--color-text-secondary, var(--fg-secondary));
    white-space: nowrap;
  }

  /* Matches .canvas-edit-link on /canvas/[name] so the control reads as part
     of the app rather than a foreign element. */
  .embed-open-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 26px;
    padding: 0 10px;
    background: var(--color-surface, transparent);
    border: 1px solid var(--color-border-strong, var(--border));
    color: var(--color-text-secondary, var(--fg-secondary));
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }
  .embed-open-link:hover:not(:disabled) {
    background: var(--color-acid-dim, rgba(212, 255, 0, 0.06));
    color: var(--color-text, var(--fg-primary));
    border-color: var(--color-acid, #d4ff00);
  }
  .embed-open-link:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .embed-open-link:focus-visible {
    outline: 2px solid var(--color-acid, #d4ff00);
    outline-offset: 2px;
  }
</style>
