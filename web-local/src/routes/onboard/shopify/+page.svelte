<script lang="ts">
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  let shop = "";
  let error = "";
  let loading = false;

  async function handleConnect() {
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

      // Store shop URL for the callback page
      sessionStorage.setItem("onboard_shopify_shop", trimmed);

      // Redirect to Shopify OAuth
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
      <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center bg-bratrax-acid/10">
        <svg class="h-10 w-10" viewBox="0 0 24 24" fill="none">
          <path
            d="M15.34 3.8c-.13-.06-.28-.02-.36.09-.07.11-1.67 2.57-1.79 2.75-.12.18-.38.3-.63.3h-.01c-.05 0-.1-.01-.15-.02-.45-.12-1.02-.18-1.6-.18-.58 0-1.15.06-1.6.18-.05.01-.1.02-.15.02-.26 0-.51-.12-.63-.3-.12-.18-1.72-2.64-1.79-2.75a.28.28 0 00-.36-.09c-.13.06-.19.21-.14.35l1.07 3.06c.04.12.01.25-.08.34-.18.18-.33.39-.44.62-.11.23-.18.48-.2.74-.02.26 0 .53.07.79.07.26.18.51.33.73.3.44.74.78 1.27.98.53.2 1.12.3 1.72.3s1.19-.1 1.72-.3c.53-.2.97-.54 1.27-.98.15-.22.26-.47.33-.73.07-.26.09-.53.07-.79a2.1 2.1 0 00-.2-.74c-.11-.23-.26-.44-.44-.62a.33.33 0 01-.08-.34l1.07-3.06c.05-.14-.01-.29-.14-.35z"
            fill="#95BF47"
          />
          <path
            d="M17.63 7.16c-.02-.14-.14-.24-.28-.24h-2.2s-.44-3.04-.47-3.19c-.03-.15-.17-.25-.32-.23L12.6 3.8c-.18-.56-.72-.96-1.35-.96s-1.17.4-1.35.96l-1.76-.3c-.15-.02-.29.08-.32.23-.03.15-.47 3.19-.47 3.19h-2.2c-.14 0-.26.1-.28.24L4 20.32c-.02.15.09.28.24.3l6.51 1.06h.04c.12 0 .22-.08.25-.2l.21-.84.21.84c.03.12.13.2.25.2h.04l6.51-1.06c.15-.02.26-.15.24-.3L17.63 7.16z"
            fill="#5E8E3E"
          />
        </svg>
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Connect <span class="font-serif italic text-bratrax-acid">Shopify</span>
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Connect your store to start pulling orders, products, and customers.
        We never modify your store.
      </p>
    </div>

    <div class="flex flex-col gap-4">
      {#if error}
        <div class="border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
          {error}
        </div>
      {/if}

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
        on:click={handleConnect}
        disabled={loading}
        class="w-full bg-bratrax-acid px-4 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
      >
        {loading ? "CONNECTING..." : "CONNECT SHOPIFY →"}
      </button>

      <p class="text-center font-mono text-[10px] text-bratrax-text-muted">
        We'll request read access to orders, products, and customers.
      </p>
    </div>
  </div>
</div>

<style>
  .onboard-page {
    background-color: #0A0A0A;
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
</style>
