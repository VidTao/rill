<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";

  // Top-level launcher for redirect-OAuth connectors, used only from the
  // Shopify admin iframe.
  //
  // The problem it solves: /onboard/stack normally does
  // `window.location.href = authUrl` in place. Inside the iframe that tries to
  // render the provider's login page in the frame, and every serious identity
  // provider refuses — Microsoft sends X-Frame-Options: DENY, so the merchant
  // sees "login.microsoftonline.com refused to connect".
  //
  // The obvious fix, opening the provider in a new tab, breaks the callback
  // instead. /onboard/stack cannot tell providers apart from their callback
  // params — Klaviyo, Bing, Pinterest and Amazon Ads all come back as
  // `?code=&state=` — so it matches the returned `state` against
  // sessionStorage[`${platform}_oauth_state`]. sessionStorage is per-tab, so a
  // freshly opened tab has none of it and the callback can't be attributed.
  //
  // Hence this page. It runs IN the new tab and establishes that tab's own
  // sessionStorage before leaving for the provider, so when the callback lands
  // here the existing state-matching logic in /onboard/stack works untouched.
  //
  // Deliberately fetches the auth URL itself rather than accepting one as a
  // query param: a caller-supplied redirect target would be an open redirect.
  // `authPath` is constrained to our own auth-url endpoints by AUTH_PATH_RE.

  const AUTH_PATH_RE = /^\/bratrax\/onboard\/[a-z0-9-]+\/auth-url$/;

  let error = "";

  onMount(async () => {
    const params = $page.url.searchParams;
    const platform = params.get("platform") ?? "";
    const authPath = params.get("authPath") ?? "";
    // Optional extra query for the endpoints that need one (Amazon SP takes a
    // region, since the Seller Central consent host differs per region).
    const extra = params.get("extra") ?? "";

    if (!platform || !AUTH_PATH_RE.test(authPath)) {
      error =
        "This connection link is malformed. Please start again from Bratrax.";
      return;
    }

    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}${authPath}${extra}`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Failed to get auth URL (${res.status})`);
      const data = await res.json();
      if (!data?.url) throw new Error("No auth URL returned");

      // Mark this tab so /onboard/stack knows, once the callback completes,
      // that it is the popped tab and should tell the merchant to go back to
      // Shopify rather than carrying on with onboarding here.
      sessionStorage.setItem("oauth_popup_tab", "1");
      sessionStorage.setItem("onboard_oauth_platform", platform);
      sessionStorage.setItem("onboard_oauth_return", "/onboard/stack");
      if (data.state) {
        sessionStorage.setItem(`${platform}_oauth_state`, data.state);
      }

      // replace() rather than assign() so the provider's page doesn't leave
      // this launcher in the tab's history for the back button to land on.
      window.location.replace(data.url);
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  });
</script>

<svelte:head><title>Connecting… — Bratrax</title></svelte:head>

<div class="flex h-screen w-screen items-center justify-center bg-bratrax-bg">
  <div
    class="w-full max-w-sm border border-bratrax-border bg-bratrax-surface p-8 text-center"
  >
    <div class="absolute-none h-1 bg-bratrax-acid"></div>
    {#if error}
      <p
        class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-tomato"
      >
        Couldn't start
      </p>
      <p class="text-sm text-bratrax-text-body">{error}</p>
      <p class="mt-4 font-mono text-[10px] text-bratrax-text-muted">
        You can close this tab.
      </p>
    {:else}
      <div
        class="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-bratrax-border border-t-bratrax-acid"
      ></div>
      <p
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted"
      >
        Redirecting you to sign in…
      </p>
    {/if}
  </div>
</div>
