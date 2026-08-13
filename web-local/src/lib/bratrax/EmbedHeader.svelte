<script lang="ts">
  import BratraxLogo from "@rilldata/web-common/layout/BratraxLogo.svelte";
  import { reserveTopLevelTab } from "$lib/bratrax/shopify-embed";
  import { createEmbedHandoff } from "$lib/bratrax/onboarding/api";

  /**
   * Header for the Shopify admin embed, above the canvas filter bar.
   *
   * Deliberately not ApplicationHeader: that one carries dashboard tabs, the
   * client switcher, the avatar menu and the theme/AI toggles, none of which
   * belong in a ~1150px panel inside someone else's admin. The root layout
   * suppresses it on /embed/* (onEmbedPage) and this stands in — same shell
   * dimensions and colours, so the two read as one design system.
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
       analytics are one click away. -->
  <p class="embed-header-message">
    <span class="embed-header-tick" aria-hidden="true"></span>
    Dashboards, customers, products and store settings —
    <strong>see it all in Bratrax.</strong>
  </p>

  <div class="embed-header-right">
    {#if openError}
      <span class="embed-open-error">{openError}</span>
    {/if}
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
     colour is what actually makes it legible, and the acid rule + bold tail
     give the eye somewhere to land and point it at the button. */
  .embed-header-message {
    @apply flex items-center text-center;
    gap: 10px;
    font-size: 16px;
    font-weight: 500;
    letter-spacing: 0.01em;
    color: var(--color-text, var(--fg-primary));
    white-space: nowrap;
  }

  /* Echoes the acid dot in the wordmark. aria-hidden — purely decorative. */
  .embed-header-tick {
    flex: none;
    width: 2px;
    height: 17px;
    background: var(--color-acid, #d4ff00);
  }

  .embed-header-message strong {
    font-weight: 700;
    color: var(--color-acid, #d4ff00);
  }

  /* Below this the logo and button start crowding the message. The message is
     `nowrap`, so it pushes rather than reflows — the cutoff has to clear
     logo + message + button + gaps + padding with room to spare, not sit right
     on it. At 16px this line is ~400px, which totals ~800px, so 900 leaves
     ~100px of slack. Re-check this if the copy gets longer: the previous
     62-char version needed 1024. The Shopify admin content column is ~1150px
     so it normally shows; this covers a small viewport or an expanded Shopify
     sidebar. The button never hides — it is the only way out of the embed. */
  @media (max-width: 900px) {
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
