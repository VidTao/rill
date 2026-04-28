<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    onboardMe,
    onboardDisconnect,
    connectedFromStackSelections,
  } from "$lib/bratrax/onboarding/api";
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";
  import AccountSelectionModal from "./AccountSelectionModal.svelte";

  const FB_APP_ID = "3518566538435783";

  // ---------------------------------------------------------------------------
  // Platform registry
  //
  // Only the 4 platforms migrated to the new rill_onboarding_state surface are
  // active. Pinterest / Amazon / Shopify / Outbrain remain commented out for
  // re-enablement one-by-one as their backends migrate off the legacy
  // bratrax_advertising_connections + bratrax_user_platform_credentials path.
  // ---------------------------------------------------------------------------
  interface Platform {
    id: string;          // matches connected_platforms[].platform + stack_selections key prefix
    name: string;
    type: "oauth" | "client_sdk" | "shopify";  // shopify needs a shop-domain input page first
    authUrlPath?: string;  // for redirect OAuth (TikTok / Klaviyo)
    color: string;
  }

  const platforms: Platform[] = [
    { id: "shopify",      name: "Shopify",      type: "shopify",                                                          color: "#95BF47" },
    { id: "google_ads",   name: "Google Ads",   type: "client_sdk",                                                       color: "#4285F4" },
    { id: "facebook_ads", name: "Facebook Ads", type: "client_sdk",                                                       color: "#1877F2" },
    { id: "tiktok_ads",   name: "TikTok Ads",   type: "oauth",      authUrlPath: "/bratrax/onboard/tiktok/auth-url",      color: "#000000" },
    { id: "klaviyo",      name: "Klaviyo",      type: "oauth",      authUrlPath: "/bratrax/onboard/klaviyo/auth-url",     color: "#2D2D2D" },

    // --- Re-enable as each platform is migrated to rill_onboarding_state ---
    // { id: "pinterest_ads", name: "Pinterest",   type: "oauth", authUrlPath: "/bratrax/connectors/pinterest/auth-url",   color: "#E60023" },
    // { id: "amazon_ads",    name: "Amazon Ads",  type: "oauth", authUrlPath: "/bratrax/connectors/amazon-ads/auth-url",  color: "#FF9900" },
    // { id: "outbrain",      name: "Outbrain",    type: "oauth", authUrlPath: "/bratrax/connectors/outbrain/auth-url",    color: "#EE6E33" },
  ];

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------
  let connectedPlatforms: Set<string> = new Set();
  let stackSelections: Record<string, any> = {};
  let connectedAt: Record<string, string> = {};
  let clientId = "";
  let loading = "";
  let error = "";

  // "View accounts" modal — generic shape for ad platforms (accounts list)
  // plus a Shopify-specific shape (single store with shop + name + currency).
  let showInfoModal = false;
  let infoModalPlatform = "";
  let infoModalAccounts: Array<{ id: string; name: string }> = [];
  let infoModalConnectedAt = "";
  let infoModalShopify: { shop: string; shopName: string; currency: string } | null = null;

  // OAuth completion modal (used by Google + Facebook popup flows; TikTok &
  // Klaviyo use redirect → land on /onboard/stack → bounce back here).
  let showAccountModal = false;
  let accountModalPlatform = "";
  let accountModalAccounts: Array<{ id: string; name: string }> = [];
  let accountModalLoading = false;
  let googleAuthCode = "";
  let fbAccessToken = "";
  let fbEmail = "";
  let fbSdkReady = false;
  let googleManagerMap: Record<string, string> = {};

  // ---------------------------------------------------------------------------
  // Status
  // ---------------------------------------------------------------------------
  async function refreshFromServer() {
    const me = await onboardMe();
    if (!me?.client_id) return;
    clientId = me.client_id;
    stackSelections = me.stack_selections || {};

    const next = new Set<string>();
    const dates: Record<string, string> = {};
    for (const p of me.connected_platforms || []) {
      if (typeof p === "object" && p?.platform) {
        next.add(p.platform);
        if (p.connected_at) dates[p.platform] = p.connected_at;
      }
    }
    for (const p of connectedFromStackSelections(stackSelections)) next.add(p);
    connectedPlatforms = next;
    connectedAt = dates;
  }

  $: isConnected = (id: string): boolean => connectedPlatforms.has(id);

  // ---------------------------------------------------------------------------
  // OAuth init handlers (mirror /onboard/stack — only difference: return-to
  // is set to /connectors so the dispatcher in /onboard/stack bounces here
  // after Klaviyo / TikTok complete).
  // ---------------------------------------------------------------------------
  async function handleOAuthConnect(platform: Platform) {
    if (isConnected(platform.id)) return;
    if (!platform.authUrlPath) return;
    error = "";
    loading = platform.id;

    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}${platform.authUrlPath}`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Failed to get auth URL for ${platform.name}`);
      const data = await res.json();

      sessionStorage.setItem("onboard_oauth_platform", platform.id);
      sessionStorage.setItem("onboard_oauth_return", "/connectors");
      if (data.state) {
        sessionStorage.setItem(`${platform.id}_oauth_state`, data.state);
      }

      window.location.href = data.url;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = "";
    }
  }

  function loadFacebookSdk() {
    if ((window as any).FB) {
      fbSdkReady = true;
      return;
    }
    (window as any).fbAsyncInit = () => {
      (window as any).FB?.init({
        appId: FB_APP_ID,
        cookie: true,
        xfbml: false,
        version: "v19.0",
      });
      fbSdkReady = true;
    };
    if (document.getElementById("facebook-jssdk")) return;
    const script = document.createElement("script");
    script.id = "facebook-jssdk";
    script.src = "https://connect.facebook.net/en_US/sdk.js";
    script.async = true;
    script.defer = true;
    document.head.appendChild(script);
  }

  function handleFacebookConnect() {
    if (isConnected("facebook_ads")) return;
    if (!fbSdkReady) {
      error = "Facebook SDK is still loading. Please try again in a moment.";
      return;
    }
    error = "";
    loading = "facebook_ads";

    (window as any).FB?.login(
      (response: { authResponse?: { accessToken: string } }) => {
        if (!response.authResponse) {
          loading = "";
          return;
        }
        const token = response.authResponse.accessToken;
        fbAccessToken = token;
        (window as any).FB?.api(
          "/me",
          { fields: "name,email" },
          (userInfo: { email?: string }) => {
            fbEmail = userInfo?.email ?? "";
            (window as any).FB?.api(
              "/me/adaccounts",
              { access_token: token, fields: "account_id,name" },
              (accountsResponse: {
                data?: Array<{ account_id: string; name: string }>;
              }) => {
                loading = "";
                const data = accountsResponse?.data ?? [];
                if (data.length === 0) {
                  error = "No Facebook ad accounts found for this user.";
                  return;
                }
                accountModalPlatform = "Facebook Ads";
                accountModalAccounts = data.map((a) => ({
                  id: a.account_id,
                  name: a.name || a.account_id,
                }));
                showAccountModal = true;
              },
            );
          },
        );
      },
      { scope: "email,ads_read,ads_management" },
    );
  }

  interface GoogleAdAccountNode {
    customer_id: string;
    name?: string;
    descriptive_name?: string;
    manager_account_id?: string;
    children?: GoogleAdAccountNode[];
  }

  function flattenAccounts(accounts: GoogleAdAccountNode[]): Array<{ id: string; name: string }> {
    const flat: Array<{ id: string; name: string }> = [];
    googleManagerMap = {};
    function walk(list: GoogleAdAccountNode[]) {
      for (const acc of list) {
        const id = String(acc.customer_id);
        flat.push({ id, name: acc.name || acc.descriptive_name || id });
        if (acc.manager_account_id) {
          googleManagerMap[id] = String(acc.manager_account_id);
        }
        if (acc.children) walk(acc.children);
      }
    }
    walk(accounts);
    return flat;
  }

  async function handleGoogleAdsConnect() {
    if (isConnected("google_ads")) return;
    error = "";
    loading = "google_ads";

    try {
      const google = (window as any).google;
      if (!google?.accounts?.oauth2) {
        throw new Error("Google SDK not loaded. Please refresh and try again.");
      }

      const client = google.accounts.oauth2.initCodeClient({
        client_id: "452833261444-1amauhc3bsundipofc2qvf3sonikknpa.apps.googleusercontent.com",
        scope: "https://www.googleapis.com/auth/adwords",
        ux_mode: "popup",
        callback: async (response: { code?: string; error?: string }) => {
          if (response.error || !response.code) {
            error = response.error || "Authorization cancelled";
            loading = "";
            return;
          }

          googleAuthCode = response.code;
          loading = "google_ads";

          try {
            const host = get(runtime).host;
            const res = await fetch(
              `${host}/bratrax/onboard/google/accounts?code=${encodeURIComponent(response.code)}`,
              { credentials: "include" },
            );
            if (!res.ok) {
              const body = await res.json().catch(() => ({}));
              throw new Error((body as any).error || "Failed to list accounts");
            }
            const data = await res.json();
            const accounts = (data as any).data || [];
            accountModalAccounts = flattenAccounts(accounts);
            accountModalPlatform = "Google Ads";
            showAccountModal = true;
          } catch (e) {
            error = e instanceof Error ? e.message : String(e);
          } finally {
            loading = "";
          }
        },
      });
      client.requestCode();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = "";
    }
  }

  // ---------------------------------------------------------------------------
  // Account-selection dispatcher (Google + Facebook only — TikTok/Klaviyo
  // complete on /onboard/stack and bounce here via goto). No return-to
  // navigation needed here since we're already on /connectors.
  // ---------------------------------------------------------------------------
  async function handleAccountSelection(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      const host = get(runtime).host;
      const isFacebook = accountModalPlatform === "Facebook Ads";

      let endpoint: string;
      let body: Record<string, unknown>;

      if (isFacebook) {
        endpoint = `${host}/bratrax/onboard/facebook/connect`;
        body = {
          short_lived_token: fbAccessToken,
          email: fbEmail,
          client_id: clientId,
          selectedAccounts: selected,
        };
      } else {
        // Google Ads
        endpoint = `${host}/bratrax/onboard/google/connect`;
        body = {
          code: googleAuthCode,
          client_id: clientId,
          selectedAccounts: selected.map((a) => ({
            ...a,
            managerAccountId: googleManagerMap[a.accountId] || "",
          })),
        };
      }

      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const resp = await res.json().catch(() => ({}));
        throw new Error((resp as any).error || "Failed to connect accounts");
      }

      showAccountModal = false;
      await refreshFromServer();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      accountModalLoading = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Disconnect
  // ---------------------------------------------------------------------------
  async function handleDisconnect(platform: Platform) {
    if (!isConnected(platform.id)) return;
    if (!clientId) return;
    if (!confirm(
      `Disconnect ${platform.name}? This removes all stored credentials and account selections for this platform.`,
    )) return;
    error = "";
    loading = platform.id;
    try {
      await onboardDisconnect(clientId, platform.id);
      await refreshFromServer();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = "";
    }
  }

  // ---------------------------------------------------------------------------
  // View accounts modal
  // ---------------------------------------------------------------------------
  function handleView(platform: Platform) {
    const cred = stackSelections[`${platform.id}_credentials`] || {};
    infoModalConnectedAt = connectedAt[platform.id] || cred.connected_at || "";
    infoModalPlatform = platform.name;

    if (platform.id === "shopify") {
      // Single-store shape — no accounts list.
      infoModalShopify = {
        shop: cred.shop || "",
        shopName: cred.shop_name || "",
        currency: cred.currency || "",
      };
      infoModalAccounts = [];
    } else {
      // Multi-account shape: TikTok stores them under `advertisers`; everyone else uses `accounts`.
      const list: any[] = cred.accounts || cred.advertisers || [];
      infoModalAccounts = list.map((a: any) => ({
        id: String(a.accountId || a.id || a.account_id || ""),
        name: a.accountName || a.name || a.account_name || a.id || "(unnamed)",
      }));
      infoModalShopify = null;
    }
    showInfoModal = true;
  }

  function handleShopifyConnect() {
    if (isConnected("shopify")) return;
    // Shopify needs a shop-domain input first — delegate to the existing
    // /onboard/shopify page (which has the input form + redirect-to-Shopify
    // OAuth init). The callback at /onboard/shopify/callback honors
    // onboard_oauth_return and bounces back here.
    sessionStorage.setItem("onboard_oauth_return", "/connectors");
    goto("/onboard/shopify");
  }

  function handleCardConnect(platform: Platform) {
    if (platform.type === "client_sdk") {
      if (platform.id === "google_ads") handleGoogleAdsConnect();
      else if (platform.id === "facebook_ads") handleFacebookConnect();
    } else if (platform.type === "shopify") {
      handleShopifyConnect();
    } else {
      handleOAuthConnect(platform);
    }
  }

  // ---------------------------------------------------------------------------
  // Init
  // ---------------------------------------------------------------------------
  onMount(async () => {
    await refreshFromServer();
    loadFacebookSdk();
    if (!document.getElementById("gis-sdk")) {
      const script = document.createElement("script");
      script.id = "gis-sdk";
      script.src = "https://accounts.google.com/gsi/client";
      script.async = true;
      document.head.appendChild(script);
    }
  });
</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="relative w-full max-w-2xl border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        CONNECTORS
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Manage your <span class="font-serif italic text-bratrax-acid">connections</span>
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Connect, view, or disconnect your data sources. Changes apply immediately.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="flex flex-col gap-3">
      {#each platforms as platform}
        <div class="flex items-center justify-between gap-4 border border-bratrax-border bg-bratrax-bg px-4 py-3">
          <div class="flex items-center gap-3 min-w-0">
            <span class="h-3 w-3 flex-shrink-0" style="background-color: {platform.color}"></span>
            <span class="font-mono text-xs font-bold uppercase tracking-wider text-bratrax-text-headline truncate">
              {platform.name}
            </span>
            {#if isConnected(platform.id)}
              <span class="flex items-center gap-1 border border-bratrax-acid/50 bg-bratrax-acid/10 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-bratrax-acid">
                <span class="h-1.5 w-1.5 rounded-full bg-bratrax-acid"></span>
                Connected
              </span>
            {:else}
              <span class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">
                Not connected
              </span>
            {/if}
          </div>

          <div class="flex items-center gap-2 flex-shrink-0">
            {#if isConnected(platform.id)}
              <button
                on:click={() => handleView(platform)}
                disabled={loading === platform.id}
                class="border border-bratrax-border bg-bratrax-surface px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover disabled:opacity-50"
              >
                View accounts
              </button>
              <button
                on:click={() => handleDisconnect(platform)}
                disabled={loading === platform.id}
                class="border border-bratrax-tomato/40 bg-bratrax-surface px-3 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-tomato hover:bg-bratrax-tomato/10 disabled:opacity-50"
              >
                {loading === platform.id ? "Disconnecting…" : "Disconnect"}
              </button>
            {:else}
              <button
                on:click={() => handleCardConnect(platform)}
                disabled={loading === platform.id}
                class="bg-bratrax-acid px-4 py-1.5 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-bg hover:opacity-90 disabled:opacity-50"
              >
                {loading === platform.id ? "Connecting…" : "Connect"}
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>

<AccountSelectionModal
  open={showAccountModal}
  loading={accountModalLoading}
  platformName={accountModalPlatform}
  accounts={accountModalAccounts}
  onSubmit={handleAccountSelection}
  onClose={() => { showAccountModal = false; }}
/>

{#if showInfoModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (showInfoModal = false)}
    on:keydown={(e) => e.key === "Escape" && (showInfoModal = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-4 flex items-baseline justify-between">
        <h2 class="text-lg font-black text-bratrax-text-headline">
          {infoModalPlatform}
        </h2>
        <button
          on:click={() => (showInfoModal = false)}
          class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted hover:text-bratrax-text-body"
        >
          Close
        </button>
      </div>

      {#if infoModalConnectedAt}
        <p class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted">
          Connected: {new Date(infoModalConnectedAt).toLocaleString()}
        </p>
      {/if}

      {#if infoModalShopify}
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-acid/70">
          Store
        </div>
        <ul class="flex flex-col gap-1.5">
          {#if infoModalShopify.shopName}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">Name</div>
              <div class="font-mono text-xs font-bold text-bratrax-text-headline truncate">
                {infoModalShopify.shopName}
              </div>
            </li>
          {/if}
          {#if infoModalShopify.shop}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">Domain</div>
              <div class="font-mono text-xs font-bold text-bratrax-text-headline truncate">
                {infoModalShopify.shop}
              </div>
            </li>
          {/if}
          {#if infoModalShopify.currency}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted">Currency</div>
              <div class="font-mono text-xs font-bold text-bratrax-text-headline truncate">
                {infoModalShopify.currency}
              </div>
            </li>
          {/if}
        </ul>
      {:else}
        <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-acid/70">
          Accounts ({infoModalAccounts.length})
        </div>

        {#if infoModalAccounts.length === 0}
          <p class="font-mono text-xs text-bratrax-text-muted">No accounts on file.</p>
        {:else}
          <ul class="flex max-h-80 flex-col gap-1.5 overflow-y-auto">
            {#each infoModalAccounts as acct}
              <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
                <div class="font-mono text-xs font-bold text-bratrax-text-headline truncate">
                  {acct.name}
                </div>
                <div class="font-mono text-[10px] text-bratrax-text-muted truncate">
                  {acct.id}
                </div>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </div>
  </div>
{/if}
