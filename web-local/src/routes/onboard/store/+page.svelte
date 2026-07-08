<script lang="ts">
  import { get } from "svelte/store";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { onMount } from "svelte";
  import { getAuthConfig } from "$lib/bratrax/auth";
  import { onboardMe } from "$lib/bratrax/onboarding/api";

  // Store-provider chooser. Shopify is connectable by everyone. WooCommerce is
  // gated behind the ALLOW_WOOCOMMERCE env flag (surfaced via /bratrax/auth/config)
  // during the token-validation phase — when off it renders as a disabled
  // "coming soon" placeholder. Shopify is pre-selected so the common flow is
  // friction-free.
  type StoreProvider = "shopify" | "woocommerce";
  let selectedStore: StoreProvider = "shopify";
  // Set when the user arrived via a store-locked signup link — forces this store
  // and disables the other card. Read from onboarding stack_selections in onMount.
  let lockedStore: StoreProvider | "" = "";

  // ALLOW_WOOCOMMERCE on the Go proxy. Fetched on mount; defaults to false so a
  // transient failure keeps WooCommerce gated rather than exposing it.
  let allowWoocommerce = false;
  onMount(async () => {
    allowWoocommerce = (await getAuthConfig()).allow_woocommerce;
    // Store-locked signup link: preselect + lock the store from onboarding state
    // (stack_selections.locked_store_platform, stamped at onboard_start).
    try {
      const me = await onboardMe();
      const locked = (me?.stack_selections as Record<string, unknown> | undefined)?.[
        "locked_store_platform"
      ];
      if (locked === "shopify" || locked === "woocommerce") {
        lockedStore = locked;
        selectedStore = locked;
      }
    } catch {
      // No lock / not resolvable → normal store chooser.
    }
  });

  // /connectors sends super_admins here with ?provider=woocommerce to jump
  // straight to the WooCommerce form. Apply once (one-shot) so a manual switch
  // back to Shopify isn't immediately overridden.
  $: providerParam = $page.url.searchParams.get("provider") || "";
  let appliedProvider = false;
  $: if (!appliedProvider && providerParam === "woocommerce" && allowWoocommerce) {
    selectedStore = "woocommerce";
    appliedProvider = true;
  }

  // Shopify
  let shop = "";
  // WooCommerce (gated by ALLOW_WOOCOMMERCE)
  let wooStoreUrl = "";

  let error = "";
  let loading = false;

  function selectStore(provider: StoreProvider) {
    // A store lock (from a store-locked signup link) forbids switching away.
    if (lockedStore && provider !== lockedStore) return;
    // WooCommerce is only selectable when ALLOW_WOOCOMMERCE is on.
    if (provider === "woocommerce" && !allowWoocommerce) return;
    selectedStore = provider;
    error = "";
  }

  async function handleShopifyConnect() {
    error = "";
    const trimmed = shop.trim().replace(/^https?:\/\//, "").replace(/\/$/, "");
    if (!trimmed || !trimmed.includes(".myshopify.com")) {
      error = "Please enter your Shopify store URL (e.g., mystore.myshopify.com)";
      return;
    }

    loading = true;
    try {
      const host = get(runtime).host;
      const res = await fetch(
        `${host}/bratrax/connectors/shopify/shop-auth-url?shop=${encodeURIComponent(trimmed)}`,
        { credentials: "include" },
      );
      if (!res.ok) {
        throw new Error("Failed to get Shopify auth URL");
      }
      const data = await res.json();

      // Store shop URL for the callback page (lands at /onboard/shopify/callback).
      sessionStorage.setItem("onboard_store_shop", trimmed);

      // Redirect to Shopify OAuth
      window.location.href = data.url;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = false;
    }
  }

  // WooCommerce wc-auth (Auth Endpoint) flow: we only collect the store URL,
  // then ask the backend to build the authorize URL and redirect the merchant
  // to their own WordPress admin to approve. WooCommerce POSTs the generated key
  // pair to our public callback and sends the browser back to return_url.
  async function handleWooConnect() {
    error = "";
    const url = wooStoreUrl.trim();
    if (!url) {
      error = "Enter your WooCommerce store URL.";
      return;
    }

    loading = true;
    try {
      const host = get(runtime).host;
      // Post-approval landing. An in-onboarding connect goes straight to
      // /onboard/stack — the next step, since WooCommerce has no Shopify-style
      // embed step and the callback advances created → platforms_connected. A
      // /connectors-initiated connect stashes onboard_oauth_return="/connectors"
      // first, so ready super_admins still return there.
      const returnTo = sessionStorage.getItem("onboard_oauth_return") || "/onboard/stack";
      const res = await fetch(
        `${host}/bratrax/onboard/woocommerce/auth-url?store_url=${encodeURIComponent(url)}&return=${encodeURIComponent(returnTo)}`,
        { credentials: "include" },
      );
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error((data as any).error || "Failed to start WooCommerce connection");
      }
      // Redirect to the store's wc-auth authorize screen.
      window.location.href = data.url;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = false;
    }
  }
</script>

<div class="onboard-page flex h-screen w-screen items-center justify-center">
  <div class="halftone-bg"></div>

  <div class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6 text-center">
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Connect your store
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Pick your platform to start pulling orders, products, and customers.
        We never modify your store.
      </p>
    </div>

    <div class="flex flex-col gap-4">
      <!-- Store-provider cards -->
      <div class="grid grid-cols-2 gap-3">
        <!-- Shopify (active) -->
        <button
          type="button"
          on:click={() => selectStore("shopify")}
          disabled={lockedStore === "woocommerce"}
          class="flex flex-col items-center gap-2 border px-4 py-4 transition-all
            {lockedStore === 'woocommerce'
              ? 'cursor-not-allowed border-bratrax-border bg-bratrax-bg text-bratrax-text-muted opacity-50'
              : selectedStore === 'shopify'
                ? 'border-bratrax-acid bg-bratrax-acid/10 bratrax-acid-text ring-1 ring-bratrax-acid'
                : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover'}"
          aria-pressed={selectedStore === "shopify"}
        >
          <svg class="h-8 w-8" viewBox="0 0 24 24" fill="none">
            <path
              d="M15.34 3.8c-.13-.06-.28-.02-.36.09-.07.11-1.67 2.57-1.79 2.75-.12.18-.38.3-.63.3h-.01c-.05 0-.1-.01-.15-.02-.45-.12-1.02-.18-1.6-.18-.58 0-1.15.06-1.6.18-.05.01-.1.02-.15.02-.26 0-.51-.12-.63-.3-.12-.18-1.72-2.64-1.79-2.75a.28.28 0 00-.36-.09c-.13.06-.19.21-.14.35l1.07 3.06c.04.12.01.25-.08.34-.18.18-.33.39-.44.62-.11.23-.18.48-.2.74-.02.26 0 .53.07.79.07.26.18.51.33.73.3.44.74.78 1.27.98.53.2 1.12.3 1.72.3s1.19-.1 1.72-.3c.53-.2.97-.54 1.27-.98.15-.22.26-.47.33-.73.07-.26.09-.53.07-.79a2.1 2.1 0 00-.2-.74c-.11-.23-.26-.44-.44-.62a.33.33 0 01-.08-.34l1.07-3.06c.05-.14-.01-.29-.14-.35z"
              fill="#95BF47"
            />
            <path
              d="M17.63 7.16c-.02-.14-.14-.24-.28-.24h-2.2s-.44-3.04-.47-3.19c-.03-.15-.17-.25-.32-.23L12.6 3.8c-.18-.56-.72-.96-1.35-.96s-1.17.4-1.35.96l-1.76-.3c-.15-.02-.29.08-.32.23-.03.15-.47 3.19-.47 3.19h-2.2c-.14 0-.26.1-.28.24L4 20.32c-.02.15.09.28.24.3l6.51 1.06h.04c.12 0 .22-.08.25-.2l.21-.84.21.84c.03.12.13.2.25.2h.04l6.51-1.06c.15-.02.26-.15.24-.3L17.63 7.16z"
              fill="#5E8E3E"
            />
          </svg>
          <span class="font-mono text-xs font-bold uppercase tracking-wider">Shopify</span>
        </button>

        <!-- WooCommerce — live when ALLOW_WOOCOMMERCE is on, else "coming soon" -->
        <button
          type="button"
          on:click={() => selectStore("woocommerce")}
          disabled={!allowWoocommerce || lockedStore === "shopify"}
          class="flex flex-col items-center gap-2 border px-4 py-4 transition-all
            {(!allowWoocommerce || lockedStore === 'shopify')
              ? 'cursor-not-allowed border-bratrax-border bg-bratrax-bg text-bratrax-text-muted'
              : selectedStore === 'woocommerce'
                ? 'border-bratrax-acid bg-bratrax-acid/10 bratrax-acid-text ring-1 ring-bratrax-acid'
                : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover'}"
          aria-pressed={selectedStore === "woocommerce"}
        >
          <svg class="h-8 w-8 {allowWoocommerce ? '' : 'opacity-50'}" viewBox="0 0 24 24" fill="none">
            <rect x="2" y="6" width="20" height="12" rx="2" fill="#7F54B3" />
            <path
              d="M6.2 9.6c.5 0 .8.3.9.8.1.6 0 1.4-.3 2.2.4-.7.7-1.2 1-1.5.2-.3.5-.4.8-.4.4 0 .6.3.7.7.1.4 0 1 .2 1.8.1-.9.4-1.6.8-2 .2-.2.4-.4.7-.4.4 0 .6.4.4 1-.3.9-.4 1.6-.4 2.1 0 .2 0 .4.1.5.3 0 .6-.4.9-1l.4.5c-.5.9-1 1.4-1.5 1.4-.4 0-.7-.4-.8-1.1-.1-.5-.1-1 0-1.5-.3.9-.6 1.6-.9 2-.2.3-.5.5-.8.5-.4 0-.6-.4-.7-1.1-.1-.5-.1-1.1-.1-1.9-.4 1.3-.8 2.2-1.2 2.6-.2.2-.4.4-.7.4-.3 0-.5-.2-.6-.6-.3-1-.3-2.4 0-4.2 0-.3.2-.4.5-.4z"
              fill="#fff"
            />
          </svg>
          <span class="font-mono text-xs font-bold uppercase tracking-wider">WooCommerce</span>
          {#if !allowWoocommerce}
            <span class="font-mono text-[10px] normal-case tracking-normal text-bratrax-text-muted">
              coming soon
            </span>
          {/if}
        </button>
      </div>

      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

      {#if selectedStore === "shopify"}
        <div class="flex flex-col gap-1">
          <label for="shop" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Store URL
          </label>
          <input
            id="shop"
            type="text"
            bind:value={shop}
            required
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="mystore.myshopify.com"
          />
        </div>

        <button
          on:click={handleShopifyConnect}
          disabled={loading}
          class="w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {loading ? "CONNECTING..." : "CONNECT SHOPIFY →"}
        </button>

        <p class="text-center font-mono text-[10px] text-bratrax-text-muted">
          We'll request read access to orders, products, and customers.
        </p>
      {:else if selectedStore === "woocommerce"}
        <div class="flex flex-col gap-1">
          <label for="woo-url" class="font-mono text-[11px] font-bold uppercase tracking-[1.5px] text-bratrax-text-muted">
            Store URL
          </label>
          <input
            id="woo-url"
            type="text"
            bind:value={wooStoreUrl}
            class="border border-bratrax-border bg-bratrax-bg px-3 py-2.5 text-sm text-bratrax-text-primary outline-none transition-colors focus:border-bratrax-acid"
            placeholder="https://yourstore.com"
            autocomplete="off"
          />
        </div>

        <button
          on:click={handleWooConnect}
          disabled={loading}
          class="w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
        >
          {loading ? "REDIRECTING..." : "CONNECT WOOCOMMERCE →"}
        </button>

        <p class="text-center font-mono text-[10px] text-bratrax-text-muted">
          You'll approve a Read-only API key in your store's WordPress admin, then
          land back here.
        </p>
      {/if}
    </div>
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

  /* The acid-dot texture is tuned for the dark canvas; hide it in light so the
     background matches the rest of the app (clean cream, like Settings). */
  :global(html:not(.dark)) .halftone-bg {
    background-image: none;
  }
</style>
