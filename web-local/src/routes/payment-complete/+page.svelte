<script lang="ts">
  import { onMount } from "svelte";
  import { shopifyAdminAppUrl } from "$lib/bratrax/shopify-embed";
  import { SHOPIFY_APP_HANDLE } from "$lib/bratrax/constants";

  // Landing page for the tab Lemon Squeezy redirects to after an embedded
  // checkout. The merchant's real session is the Shopify admin window behind
  // this one, which is already polling and will advance itself as soon as the
  // LS webhook lands. So this page has exactly one job: tell them they can go
  // back, and give them a way to do it if the original tab is gone.
  //
  // Public by design — this tab may have no bratrax_auth cookie at all. The
  // merchant authenticated inside the iframe via a Shopify session token, and
  // session tokens don't set cookies on this origin. Registered in BOTH the
  // isPublicRoute allowlist in +layout.ts and the Go proxy, per the
  // three-place checklist in CLAUDE.md — missing either one bounces them to
  // /login immediately after they paid, which is the worst possible moment.

  let backUrl = "";
  let closed = false;

  onMount(() => {
    // The shop arrives as a query param, put there when the checkout was
    // created. It cannot come from sessionStorage: this tab was opened fresh
    // by the break-out and has its own empty storage — nothing the embedded
    // window stashed is readable here. Without the param we can still tell
    // them to switch tabs, just not deep-link.
    const shop = new URLSearchParams(window.location.search).get("shop") || "";
    if (/^[a-zA-Z0-9][a-zA-Z0-9-]*\.myshopify\.com$/.test(shop)) {
      backUrl = shopifyAdminAppUrl(shop, SHOPIFY_APP_HANDLE);
    }

    // Best effort: browsers only honour close() for script-opened windows,
    // which this is. Never rely on it — the copy below stands on its own.
    setTimeout(() => {
      try {
        window.close();
        closed = true;
      } catch {
        /* ignore */
      }
    }, 2500);
  });
</script>

<svelte:head><title>Payment received — Bratrax</title></svelte:head>

<div class="onboard-page flex h-screen w-screen items-center justify-center">
  <div class="halftone-bg"></div>

  <div
    class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8 text-center"
  >
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <p
      class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid"
    >
      Payment received
    </p>
    <h1 class="text-2xl font-black text-bratrax-text-headline">
      You're all set
    </h1>
    <p class="mt-3 text-sm font-light text-bratrax-text-body">
      Head back to the Bratrax tab in your Shopify admin — setup continues there
      automatically. You can close this window.
    </p>

    {#if backUrl}
      <a
        href={backUrl}
        class="mt-6 inline-block w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90"
      >
        Back to Shopify →
      </a>
    {/if}

    {#if !closed}
      <p class="mt-4 font-mono text-[10px] text-bratrax-text-muted">
        Safe to close this tab.
      </p>
    {/if}
  </div>
</div>

<style>
  .onboard-page {
    background-color: var(--color-bg);
    position: relative;
    overflow: hidden;
  }

  .halftone-bg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background-image: radial-gradient(
      rgba(212, 255, 0, 0.06) 1.5px,
      transparent 1.5px
    );
    background-size: 8px 8px;
  }

  :global(html:not(.dark)) .halftone-bg {
    background-image: none;
  }
</style>
