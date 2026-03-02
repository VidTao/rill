<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { getPlatform } from "$lib/bratrax/connectors/platforms";
  import {
    exchangeToken,
    exchangeShopifyTokens,
    getShopifyAuthUrl,
  } from "$lib/bratrax/connectors/api";

  export let data: { platform: string; params: Record<string, string> };

  let status: "processing" | "success" | "error" = "processing";
  let errorMessage = "";

  onMount(async () => {
    const { platform: platformSlug, params: urlParams } = data;
    const code = urlParams.code ?? urlParams.spapi_oauth_code ?? "";
    const state = urlParams.state ?? "";

    try {
      // Shopify connect step: redirect to Shopify OAuth with shop URL
      if (platformSlug === "shopify-connect") {
        const shop = urlParams.shop ?? sessionStorage.getItem("bratrax-shopify-shop") ?? "";
        if (!shop) throw new Error("Missing Shopify store URL");
        const url = await getShopifyAuthUrl(shop);
        window.location.href = url;
        return;
      }

      // Shopify token exchange
      if (platformSlug === "shopify") {
        const shop = urlParams.shop ?? sessionStorage.getItem("bratrax-shopify-shop") ?? "";
        if (!shop) throw new Error("Missing Shopify store URL");
        await exchangeShopifyTokens(code, shop);
        sessionStorage.removeItem("bratrax-shopify-shop");
        status = "success";
        await goto("/connectors");
        return;
      }

      // Amazon Ads: read region from sessionStorage
      if (platformSlug.startsWith("amazon-ads")) {
        const region = sessionStorage.getItem("bratrax-region-amazon-ads") ?? "us";
        await exchangeToken("amazon-ads", { code, state, region });
        sessionStorage.removeItem("bratrax-region-amazon-ads");
        status = "success";
        await goto("/connectors");
        return;
      }

      // Amazon SP: read region from sessionStorage
      if (platformSlug.startsWith("amazon-sp") || platformSlug === "amazon") {
        const region = sessionStorage.getItem("bratrax-region-amazon") ?? "us";
        const sellingPartnerId = urlParams.selling_partner_id ?? "";
        await exchangeToken("amazon", {
          code,
          region,
          selling_partner_id: sellingPartnerId,
        });
        sessionStorage.removeItem("bratrax-region-amazon");
        status = "success";
        await goto("/connectors");
        return;
      }

      // Generic OAuth callback: look up platform config to get apiSlug
      const platformConfig = getPlatform(platformSlug);
      const apiSlug = platformConfig?.apiSlug ?? platformSlug;

      const body: Record<string, string> = { code };
      if (state) body.state = state;

      await exchangeToken(apiSlug, body);
      status = "success";
      await goto("/connectors");
    } catch (e) {
      status = "error";
      errorMessage = e instanceof Error ? e.message : String(e);
    }
  });
</script>

<div class="flex h-full w-full items-center justify-center">
  <div class="w-full max-w-sm rounded-lg border bg-white p-8 text-center shadow-sm">
    {#if status === "processing"}
      <svg class="mx-auto mb-4 h-8 w-8 animate-spin text-indigo-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      <p class="text-sm text-gray-600">Completing connection...</p>
    {:else if status === "error"}
      <div class="mb-4 text-red-500">
        <svg class="mx-auto h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
      </div>
      <h2 class="mb-2 text-lg font-semibold text-gray-800">Connection Failed</h2>
      <p class="mb-4 text-sm text-red-600">{errorMessage}</p>
      <a
        href="/connectors"
        class="inline-block rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
      >
        Back to Connectors
      </a>
    {:else}
      <p class="text-sm text-green-600">Connected! Redirecting...</p>
    {/if}
  </div>
</div>
