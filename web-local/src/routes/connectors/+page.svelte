<script lang="ts">
  import { onMount } from "svelte";
  import type { PlatformConfig, AdvertisingConnection, CrmConnection } from "$lib/bratrax/connectors/types";
  import {
    CATEGORY_ORDER,
    CATEGORY_LABELS,
    getPlatformsByCategory,
  } from "$lib/bratrax/connectors/platforms";
  import {
    platformConnections,
    adConnections,
    crmConnections,
    connectionsLoading,
    fetchAllConnections,
  } from "$lib/bratrax/connectors/connection-store";
  import {
    getAuthUrl,
    getShopifyAuthUrl,
    submitOutbrainCredentials,
    exchangeFacebookTokens,
    getGoogleAdAccounts,
    exchangeGoogleTokens,
    type AdAccountInfo,
    type GoogleAdAccount,
  } from "$lib/bratrax/connectors/api";
  import PlatformCard from "./PlatformCard.svelte";
  import ConnectionDetailModal from "./ConnectionDetailModal.svelte";
  import ShopifyUrlModal from "./ShopifyUrlModal.svelte";
  import OutbrainLoginModal from "./OutbrainLoginModal.svelte";
  import AccountSelectionModal from "./AccountSelectionModal.svelte";

  // ── SDK config ──
  const FB_APP_ID = "3518566538435783";
  const GOOGLE_CLIENT_ID = "452833261444-1amauhc3bsundipofc2qvf3sonikknpa.apps.googleusercontent.com";

  // ── Error state ──
  let errorMessage = "";

  // ── Modal state ──
  let shopifyModalOpen = false;
  let outbrainModalOpen = false;
  let detailModalOpen = false;
  let detailPlatformName = "";
  let detailAdAccounts: AdvertisingConnection[] = [];
  let detailCrmAccounts: CrmConnection[] = [];

  // ── Account selection modal state ──
  let accountModalOpen = false;
  let accountModalPlatform = "";
  let accountModalAccounts: Array<{ id: string; name: string }> = [];
  let accountModalLoading = false;

  // Facebook-specific state
  let fbAccessToken = "";
  let fbEmail = "";

  // Google-specific state
  let googleAuthCode = "";

  // Loading state per platform
  let loadingPlatform = "";

  // ── SDK loading ──
  let fbSdkReady = false;
  let googleSdkReady = false;

  onMount(() => {
    loadFacebookSdk();
    loadGoogleSdk();
  });

  function loadFacebookSdk() {
    if (typeof window === "undefined") return;
    // @ts-ignore
    window.fbAsyncInit = () => {
      // @ts-ignore
      window.FB?.init({
        appId: FB_APP_ID,
        cookie: true,
        xfbml: false,
        version: "v18.0",
      });
      fbSdkReady = true;
    };
    if (document.getElementById("facebook-jssdk")) {
      fbSdkReady = true;
      return;
    }
    const script = document.createElement("script");
    script.id = "facebook-jssdk";
    script.src = "https://connect.facebook.net/en_US/sdk.js";
    script.async = true;
    script.defer = true;
    document.head.appendChild(script);
  }

  function loadGoogleSdk() {
    if (typeof window === "undefined") return;
    if (document.getElementById("google-gsi")) {
      googleSdkReady = true;
      return;
    }
    const script = document.createElement("script");
    script.id = "google-gsi";
    script.src = "https://accounts.google.com/gsi/client";
    script.async = true;
    script.defer = true;
    script.onload = () => { googleSdkReady = true; };
    document.head.appendChild(script);
  }

  // ── Helpers ──

  function isConnected(platform: PlatformConfig): boolean {
    return !!$platformConnections[platform.connectionKey];
  }

  function getConnectedDate(platform: PlatformConfig): string {
    const conn = $platformConnections[platform.connectionKey];
    if (!conn) return "";
    try {
      return new Date(conn.created_at).toLocaleDateString();
    } catch {
      return conn.created_at;
    }
  }

  function showError(msg: string) {
    errorMessage = msg;
    setTimeout(() => { errorMessage = ""; }, 8000);
  }

  // ── Facebook flow ──

  function handleFacebookConnect() {
    if (!fbSdkReady) {
      showError("Facebook SDK is still loading. Please try again in a moment.");
      return;
    }
    loadingPlatform = "facebook";
    // @ts-ignore
    window.FB?.login(
      (response: { authResponse?: { accessToken: string } }) => {
        if (!response.authResponse) {
          loadingPlatform = "";
          return;
        }
        const token = response.authResponse.accessToken;
        fbAccessToken = token;
        // Get user email
        // @ts-ignore
        window.FB?.api("/me", { fields: "name,email" }, (userInfo: { email?: string }) => {
          fbEmail = userInfo?.email ?? "";
          // Get ad accounts
          // @ts-ignore
          window.FB?.api(
            "/me/adaccounts",
            { access_token: token, fields: "account_id,name" },
            (accountsResponse: { data?: Array<{ account_id: string; name: string }> }) => {
              loadingPlatform = "";
              const data = accountsResponse?.data ?? [];
              if (data.length === 0) {
                showError("No Facebook ad accounts found for this user.");
                return;
              }
              accountModalPlatform = "Facebook Ads";
              accountModalAccounts = data.map((a) => ({
                id: a.account_id,
                name: a.name || a.account_id,
              }));
              accountModalOpen = true;
            },
          );
        });
      },
      { scope: "email,ads_read" },
    );
  }

  async function handleFacebookAccountsSelected(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      await exchangeFacebookTokens(fbAccessToken, fbEmail, selected);
      accountModalOpen = false;
      accountModalLoading = false;
      await fetchAllConnections();
    } catch (e) {
      accountModalLoading = false;
      showError(e instanceof Error ? e.message : "Failed to connect Facebook accounts");
    }
  }

  // ── Google flow ──

  function handleGoogleConnect() {
    if (!googleSdkReady) {
      showError("Google SDK is still loading. Please try again in a moment.");
      return;
    }
    loadingPlatform = "google-ads";
    try {
      // @ts-ignore
      const client = google.accounts.oauth2.initCodeClient({
        client_id: GOOGLE_CLIENT_ID,
        scope: "https://www.googleapis.com/auth/adwords",
        ux_mode: "popup",
        callback: async (response: { code?: string; error?: string }) => {
          if (response.error || !response.code) {
            loadingPlatform = "";
            if (response.error !== "access_denied") {
              showError(response.error ?? "Google login was cancelled.");
            }
            return;
          }
          googleAuthCode = response.code;
          // Fetch available ad accounts
          try {
            const accounts = await getGoogleAdAccounts(response.code);
            loadingPlatform = "";
            // Flatten the hierarchy for selection
            const flat: Array<{ id: string; name: string }> = [];
            function flatten(list: GoogleAdAccount[]) {
              for (const acc of list) {
                flat.push({
                  id: acc.customer_id,
                  name: acc.name || acc.customer_id,
                });
                if (acc.children) flatten(acc.children);
              }
            }
            flatten(Array.isArray(accounts) ? accounts : [accounts]);
            if (flat.length === 0) {
              showError("No Google Ads accounts found.");
              return;
            }
            accountModalPlatform = "Google Ads";
            accountModalAccounts = flat;
            accountModalOpen = true;
          } catch (e) {
            loadingPlatform = "";
            showError(e instanceof Error ? e.message : "Failed to fetch Google Ads accounts");
          }
        },
      });
      client.requestCode();
    } catch (e) {
      loadingPlatform = "";
      showError(e instanceof Error ? e.message : "Failed to start Google login");
    }
  }

  async function handleGoogleAccountsSelected(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      await exchangeGoogleTokens(googleAuthCode, selected);
      accountModalOpen = false;
      accountModalLoading = false;
      await fetchAllConnections();
    } catch (e) {
      accountModalLoading = false;
      showError(e instanceof Error ? e.message : "Failed to connect Google Ads accounts");
    }
  }

  // ── Generic connect handler ──

  async function handleConnect(platform: PlatformConfig) {
    errorMessage = "";

    // Facebook: client-side SDK flow
    if (platform.id === "facebook") {
      handleFacebookConnect();
      return;
    }

    // Google Ads: client-side SDK flow
    if (platform.id === "google-ads") {
      handleGoogleConnect();
      return;
    }

    if (platform.flowType === "oauth-modal-input" && platform.id === "shopify") {
      shopifyModalOpen = true;
      return;
    }

    if (platform.flowType === "credential-modal" && platform.id === "outbrain") {
      outbrainModalOpen = true;
      return;
    }

    loadingPlatform = platform.id;
    try {
      const params: Record<string, string> = {};
      if (platform.flowType === "oauth-redirect-region" && platform.regions) {
        const region = platform.regions[0];
        params.region = region;
        sessionStorage.setItem(`bratrax-region-${platform.apiSlug}`, region);
      }
      const url = await getAuthUrl(
        platform.apiSlug,
        Object.keys(params).length > 0 ? params : undefined,
      );
      window.location.href = url;
    } catch (e) {
      loadingPlatform = "";
      showError(e instanceof Error ? e.message : `Failed to connect ${platform.name}`);
    }
  }

  async function handleShopifySubmit(shopUrl: string) {
    loadingPlatform = "shopify";
    try {
      const url = await getShopifyAuthUrl(shopUrl);
      sessionStorage.setItem("bratrax-shopify-shop", shopUrl);
      window.location.href = url;
    } catch (e) {
      loadingPlatform = "";
      showError(e instanceof Error ? e.message : "Failed to connect Shopify");
    }
  }

  async function handleOutbrainSubmit(encodedAuth: string) {
    loadingPlatform = "outbrain";
    try {
      await submitOutbrainCredentials(encodedAuth);
      outbrainModalOpen = false;
      loadingPlatform = "";
      await fetchAllConnections();
    } catch (e) {
      loadingPlatform = "";
      showError(e instanceof Error ? e.message : "Failed to connect Outbrain");
    }
  }

  function handleViewDetails(platform: PlatformConfig) {
    detailPlatformName = platform.name;
    detailAdAccounts = $adConnections[platform.connectionKey] ?? [];
    detailCrmAccounts = $crmConnections[platform.connectionKey] ?? [];
    detailModalOpen = true;
  }

  // Account selection callback router
  function handleAccountsSubmit(selected: AdAccountInfo[]) {
    if (accountModalPlatform === "Facebook Ads") {
      handleFacebookAccountsSelected(selected);
    } else if (accountModalPlatform === "Google Ads") {
      handleGoogleAccountsSelected(selected);
    }
  }
</script>

<div class="mx-auto w-full max-w-6xl px-6 py-12">
  <div class="mb-10">
    <div class="mb-3 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
      01 — CONNECTORS
    </div>
    <h1 class="text-3xl font-black tracking-tight text-bratrax-text-headline">
      Connect to <span class="font-serif italic text-bratrax-acid">Platforms</span>
    </h1>
    <p class="mt-2 text-[15px] font-light text-bratrax-text-body">
      Link your advertising, e-commerce, and analytics accounts
    </p>
  </div>

  {#if errorMessage}
    <div class="mx-auto mb-6 max-w-xl border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-4 py-3 font-mono text-xs text-bratrax-tomato">
      <div class="flex items-start gap-2">
        <svg class="mt-0.5 h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        <span>{errorMessage}</span>
        <button class="ml-auto text-bratrax-tomato/60 hover:text-bratrax-tomato" on:click={() => { errorMessage = ""; }}>
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  {/if}

  {#if $connectionsLoading}
    <div class="flex justify-center py-12">
      <svg class="h-6 w-6 animate-spin text-bratrax-acid" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>
  {:else}
    {#each CATEGORY_ORDER as category, i}
      {@const platforms = getPlatformsByCategory(category)}
      {#if platforms.length > 0}
        <section class="mb-10">
          <h2 class="mb-4 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
            {String(i + 1).padStart(2, '0')} — {CATEGORY_LABELS[category]}
          </h2>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {#each platforms as platform (platform.id)}
              <PlatformCard
                name={platform.name}
                logo={platform.logo}
                description={platform.description}
                connected={isConnected(platform)}
                connectedDate={getConnectedDate(platform)}
                loading={loadingPlatform === platform.id}
                onConnect={() => handleConnect(platform)}
                onViewDetails={() => handleViewDetails(platform)}
              />
            {/each}
          </div>
        </section>
      {/if}
    {/each}
  {/if}
</div>

<ShopifyUrlModal
  open={shopifyModalOpen}
  loading={loadingPlatform === "shopify"}
  onSubmit={handleShopifySubmit}
  onClose={() => { shopifyModalOpen = false; }}
/>

<OutbrainLoginModal
  open={outbrainModalOpen}
  loading={loadingPlatform === "outbrain"}
  onSubmit={handleOutbrainSubmit}
  onClose={() => { outbrainModalOpen = false; }}
/>

<ConnectionDetailModal
  open={detailModalOpen}
  platformName={detailPlatformName}
  adAccounts={detailAdAccounts}
  crmAccounts={detailCrmAccounts}
  onClose={() => { detailModalOpen = false; }}
/>

<AccountSelectionModal
  open={accountModalOpen}
  loading={accountModalLoading}
  platformName={accountModalPlatform}
  accounts={accountModalAccounts}
  onSubmit={handleAccountsSubmit}
  onClose={() => { accountModalOpen = false; }}
/>
