<script lang="ts">
  /**
   * The Bratrax wordmark, with the light/dark variant swap.
   *
   * Two <img> tags rather than one with a dynamic src: the swap is driven by
   * the `.dark` class on <html> (see web-local/src/lib/bratrax/theme.ts), so
   * doing it in CSS means no flash while a reactive statement catches up, and
   * both files sit in the browser cache after the first theme toggle.
   *
   * `href = null` renders a plain <span> instead of a link. The Shopify admin
   * embed needs that: a link to "/" would navigate the *iframe* to the apex,
   * which redirects to /canvas/<name> and drops the merchant into the full app
   * shell inside Shopify's admin — the exact thing /embed/* exists to avoid.
   */
  export let href: string | null = "/";
</script>

{#if href}
  <a {href} class="bratrax-logo-link">
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
{:else}
  <span class="bratrax-logo-link">
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
  </span>
{/if}

<style lang="postcss">
  /* These rules live here, not in the consumer. Svelte scopes styles per
     component, so a copy left behind in ApplicationHeader would silently stop
     matching once the markup moved and the logo would render unstyled at its
     intrinsic size. */
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
</style>
