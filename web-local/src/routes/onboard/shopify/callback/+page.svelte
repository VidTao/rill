<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    onboardConnect,
    onboardMe,
    verifyEmbedStatus,
  } from "$lib/bratrax/onboarding/api";

  let status = "Completing Shopify connection...";
  let error = "";

  onMount(async () => {
    const code = $page.url.searchParams.get("code");
    const shopParam = $page.url.searchParams.get("shop");
    const shop = shopParam || sessionStorage.getItem("onboard_shopify_shop") || "";

    if (!code || !shop) {
      error = "Missing authorization code or shop. Please try connecting again.";
      return;
    }

    try {
      // Step 1: Exchange code for access token via Lite endpoint
      // (skips legacy write_key/CRM tables that cause FK violations)
      status = "Exchanging authorization code...";
      const host = get(runtime).host;
      const me = await onboardMe();
      const clientId = me?.client_id || sessionStorage.getItem("onboard_client_id") || "";
      const tokenRes = await fetch(`${host}/bratrax/onboard/shopify/token`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ code, shop, client_id: clientId }),
      });

      if (!tokenRes.ok) {
        const body = await tokenRes.json().catch(() => ({}));
        throw new Error(body.error ?? "Failed to connect Shopify");
      }

      // Step 2: Honor the return-to set by the originating page. /onboard/shopify
      // (the default onboarding path) sets it to "/onboard/stack"; /connectors
      // sets it to "/connectors". Falls back to /onboard/stack for the legacy
      // path that didn't set sessionStorage.
      status = "Connected! Redirecting...";
      const returnTo = sessionStorage.getItem("onboard_oauth_return") || "/onboard/stack";

      // Reconnect flow (returnTo points outside /onboard, e.g. /connectors):
      // the backend keeps step="ready" for fully-onboarded users, so the
      // layout's embed_pending hard-gate won't fire. Mirror the initial
      // onboard by checking the Theme App Embed ourselves and detouring
      // through /onboard/embed when it's off — that page honors
      // onboard_oauth_return and will route back to returnTo once enabled.
      if (clientId && !returnTo.startsWith("/onboard")) {
        try {
          const embed = await verifyEmbedStatus(clientId);
          if (!embed.enabled) {
            await goto("/onboard/embed");
            return;
          }
        } catch {
          // Non-fatal: fall through to returnTo. The /connectors banner
          // will surface the missing-embed state on the next render.
        }
      }

      sessionStorage.removeItem("onboard_oauth_return");
      await goto(returnTo);
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  });
</script>

<div class="flex h-screen w-screen items-center justify-center bg-surface">
  <div class="w-full max-w-sm rounded-lg border border-border bg-white p-8 shadow-sm text-center">
    {#if error}
      <div class="mb-4 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
        {error}
      </div>
      <a
        href="/onboard/shopify"
        class="text-sm text-indigo-600 hover:underline"
      >
        Try again
      </a>
    {:else}
      <div class="mb-4">
        <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600"></div>
      </div>
      <p class="text-sm text-fg-secondary">{status}</p>
    {/if}
  </div>
</div>
