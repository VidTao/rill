<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    onboardMe,
    onboardDisconnect,
    connectedFromStackSelections,
    getOAuthConfig,
    verifyEmbedStatus,
  } from "$lib/bratrax/onboarding/api";
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";
  import AccountSelectionModal from "./AccountSelectionModal.svelte";
  import ExternalPagesInstallModal from "./ExternalPagesInstallModal.svelte";
  import TrackingTemplateGuide from "$lib/bratrax/TrackingTemplateGuide.svelte";

  // Public OAuth client IDs — fetched from /onboard/oauth-config on mount
  // (BUG-01: not hardcoded in source). Empty until the fetch completes;
  // OAuth-init click handlers guard against empty values.
  let fbAppId = "";
  let googleClientId = "";

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
    // "shopify" needs a shop-domain input page first; "snippet_install" is a
    // copy-paste pixel (no OAuth) — flagged installed when the user clicks
    // "I've installed it" in the snippet modal.
    type: "oauth" | "client_sdk" | "shopify" | "snippet_install";
    authUrlPath?: string;  // for redirect OAuth (TikTok / Klaviyo)
    color: string;
  }

  const platforms: Platform[] = [
    { id: "shopify",        name: "Shopify",                type: "shopify",                                                          color: "#95BF47" },
    { id: "google_ads",     name: "Google Ads",             type: "client_sdk",                                                       color: "#4285F4" },
    { id: "facebook_ads",   name: "Facebook Ads",           type: "client_sdk",                                                       color: "#1877F2" },
    { id: "tiktok_ads",     name: "TikTok Ads",             type: "oauth",            authUrlPath: "/bratrax/onboard/tiktok/auth-url",      color: "#000000" },
    { id: "klaviyo",        name: "Klaviyo",                type: "oauth",            authUrlPath: "/bratrax/onboard/klaviyo/auth-url",     color: "#2D2D2D" },
    { id: "external_pages", name: "External Landing Pages", type: "snippet_install",                                                  color: "#F59E0B" },

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

  // Disconnect confirmation modal — replaces the browser-native confirm() so
  // the prompt matches Bratrax styling.
  let showConfirmModal = false;
  let confirmPlatform: Platform | null = null;

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

  // Shopify App Embed (B13/C19) banner state. Surfaces when Shopify is
  // connected but the merchant hasn't toggled the Theme App Embed on yet —
  // covers customers who onboarded before B13 shipped and never saw the
  // /onboard/embed step.
  let shopifyEmbedEnabled = true;
  let shopifyShopDomain = "";
  let embedRecheckBusy = false;
  let embedRecheckMessage = "";

  // Third-party page pixel (B12) install snippet modal. No OAuth — the
  // merchant pastes the snippet into a non-Shopify page and clicks "I've
  // installed it" to flag external_pages in connected_platforms.
  let showSnippetModal = false;

  $: showShopifyEmbedBanner =
    connectedPlatforms.has("shopify") && !shopifyEmbedEnabled;
  $: themeEditorUrl = shopifyShopDomain
    ? `https://${shopifyShopDomain}/admin/themes/current/editor?context=apps`
    : "";

  // ---------------------------------------------------------------------------
  // Status
  // ---------------------------------------------------------------------------
  async function refreshFromServer() {
    const me = await onboardMe();
    if (!me?.client_id) return;
    clientId = me.client_id;
    stackSelections = me.stack_selections || {};
    shopifyEmbedEnabled = !!me.shopify_embed_enabled;

    const creds = (stackSelections as Record<string, unknown>)
      ?.shopify_credentials as { shop?: string } | undefined;
    shopifyShopDomain = creds?.shop ?? "";

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

    // Live-check the embed when Shopify is connected. The cached
    // me.shopify_embed_enabled flag only ever ticks from false → true (the
    // verify endpoint flips it on detection, nothing flips it back), so a
    // merchant who disabled the embed after our last check would keep a
    // stale "enabled" cache and the banner would never surface. Hit
    // /bratrax/onboard/embed-status on every visit so the banner reflects
    // Shopify's current state, not the snapshot from connect day.
    if (next.has("shopify") && clientId) {
      try {
        const res = await verifyEmbedStatus(clientId);
        shopifyEmbedEnabled = res.enabled;
      } catch {
        // Non-fatal: keep the cached value from onboardMe.
      }
    }
  }

  function explainEmbedReason(reason: string | undefined): string {
    switch (reason) {
      case "not_present":
        return "Not detected yet — make sure you toggled Bratrax on and clicked Save in the Theme Editor.";
      case "auth_error":
        return "Shopify rejected the read request. Disconnect and reconnect Shopify so we can request the new theme-read permission.";
      case "no_active_theme":
        return "Couldn't find an active theme on this store. Publish a theme and try again.";
      case "no_settings_data":
        return "This theme isn't an Online Store 2.0 theme — App embeds aren't supported on legacy themes. Switch to or duplicate into a 2.0 theme.";
      case "invalid_settings_json":
        return "Shopify returned a settings file we couldn't parse. Try saving the theme again.";
      case "api_error":
        return "Shopify Asset API call failed. Try again in a moment.";
      case "no_credentials":
        return "Shopify isn't connected for this client.";
      default:
        return "Couldn't verify the embed. Try again.";
    }
  }

  async function recheckShopifyEmbed() {
    if (!clientId || embedRecheckBusy) return;
    embedRecheckBusy = true;
    embedRecheckMessage = "";
    try {
      const res = await verifyEmbedStatus(clientId);
      if (res.enabled) {
        shopifyEmbedEnabled = true;
        embedRecheckMessage = "";
      } else {
        embedRecheckMessage = explainEmbedReason(res.reason);
        if (res.debug) {
          // eslint-disable-next-line no-console
          console.log("[bratrax embed-status]", res);
        }
      }
    } catch (e) {
      embedRecheckMessage = e instanceof Error ? e.message : String(e);
    } finally {
      embedRecheckBusy = false;
    }
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
        appId: fbAppId,
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
    if (!fbAppId) {
      error = "OAuth config still loading. Please try again in a moment.";
      return;
    }
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
    if (!googleClientId) {
      error = "OAuth config still loading. Please try again in a moment.";
      return;
    }
    error = "";
    loading = "google_ads";

    try {
      const google = (window as any).google;
      if (!google?.accounts?.oauth2) {
        throw new Error("Google SDK not loaded. Please refresh and try again.");
      }

      const client = google.accounts.oauth2.initCodeClient({
        client_id: googleClientId,
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
  function requestDisconnect(platform: Platform) {
    if (!isConnected(platform.id)) return;
    if (!clientId) return;
    confirmPlatform = platform;
    showConfirmModal = true;
  }

  async function confirmDisconnect() {
    const platform = confirmPlatform;
    showConfirmModal = false;
    confirmPlatform = null;
    if (!platform || !clientId) return;
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
    } else if (platform.type === "snippet_install") {
      openSnippetModal();
    } else {
      handleOAuthConnect(platform);
    }
  }

  function openSnippetModal() {
    error = "";
    showSnippetModal = true;
  }

  async function handleExternalPagesInstalled() {
    showSnippetModal = false;
    await refreshFromServer();
  }

  // ---------------------------------------------------------------------------
  // Init
  // ---------------------------------------------------------------------------
  onMount(async () => {
    await refreshFromServer();
    // Public OAuth client IDs (BUG-01: not hardcoded in source).
    try {
      const cfg = await getOAuthConfig();
      fbAppId = cfg.fb_app_id;
      googleClientId = cfg.google_client_id;
    } catch {
      // Non-fatal — connect-button handlers guard against empty values.
    }
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

<div class="connectors-canvas">
  <div class="connectors-inner">
    <div class="mb-7">
      <h1 class="connectors-title">Manage your connections</h1>
      <p class="connectors-subhead">
        Connect, view, or disconnect your data sources. Changes apply immediately.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    {#if showShopifyEmbedBanner}
      <div class="mb-4 border border-bratrax-acid/40 bg-bratrax-acid/10 px-4 py-3">
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1">
            <p class="mb-1 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid">
              Action required
            </p>
            <p class="text-sm text-bratrax-text-body">
              Enable the Bratrax tracker in your Shopify Theme Editor so we
              can capture email signups and on-site events. Takes about 30
              seconds.
            </p>
            {#if embedRecheckMessage}
              <p class="mt-2 font-mono text-[11px] text-bratrax-text-muted">
                {embedRecheckMessage}
              </p>
            {/if}
          </div>
          <div class="flex shrink-0 items-center gap-2">
            {#if themeEditorUrl}
              <a
                href={themeEditorUrl}
                target="_blank"
                rel="noopener noreferrer"
                class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90"
              >
                Open Theme Editor →
              </a>
            {/if}
            <button
              type="button"
              on:click={recheckShopifyEmbed}
              disabled={embedRecheckBusy}
              class="border border-bratrax-border px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted transition-colors hover:border-bratrax-acid hover:text-bratrax-acid disabled:opacity-40"
            >
              {embedRecheckBusy ? "Checking…" : "Re-check"}
            </button>
          </div>
        </div>
      </div>
    {/if}

    <div class="connector-list-card">
      {#each platforms as platform}
        {@const connected = isConnected(platform.id)}
        <div class="connector-row" class:row-disconnected={!connected}>
          <span class="connector-icon" style="background-color: {platform.color}"></span>
          <div class="connector-mid">
            <span class="connector-name">{platform.name}</span>
            {#if connected}
              <span class="connector-pill">Connected</span>
            {:else}
              <span class="connector-status-muted">Not connected</span>
            {/if}
          </div>
          <div class="connector-actions">
            {#if connected}
              <button
                on:click={() => platform.type === "snippet_install" ? openSnippetModal() : handleView(platform)}
                disabled={loading === platform.id}
                class="btn-bratrax btn-neutral btn-compact"
              >
                {platform.type === "snippet_install" ? "View snippet" : "View accounts"}
              </button>
              <button
                on:click={() => requestDisconnect(platform)}
                disabled={loading === platform.id}
                class="btn-bratrax btn-destructive btn-compact"
              >
                {loading === platform.id ? "Disconnecting…" : "Disconnect"}
              </button>
            {:else}
              <button
                on:click={() => handleCardConnect(platform)}
                disabled={loading === platform.id}
                class="btn-bratrax btn-primary btn-compact"
              >
                {#if loading === platform.id}
                  Connecting…
                {:else if platform.type === "snippet_install"}
                  Install →
                {:else}
                  Connect →
                {/if}
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <TrackingTemplateGuide />
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

{#if showConfirmModal && confirmPlatform}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    on:click={() => (showConfirmModal = false)}
    on:keydown={(e) => e.key === "Escape" && (showConfirmModal = false)}
    role="presentation"
  >
    <div
      class="relative w-full max-w-md border border-bratrax-border bg-bratrax-surface p-6"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-tomato"></div>

      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80">
        Confirm disconnect
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Disconnect {confirmPlatform.name}?
      </h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        This removes all stored credentials and account selections for {confirmPlatform.name} from your workspace. You can reconnect at any time.
      </p>

      <div class="mt-6 flex items-center justify-end gap-2">
        <button
          on:click={() => (showConfirmModal = false)}
          class="btn-bratrax btn-neutral btn-compact"
        >
          Cancel
        </button>
        <button
          on:click={confirmDisconnect}
          class="btn-bratrax btn-destructive btn-compact"
        >
          Disconnect
        </button>
      </div>
    </div>
  </div>
{/if}

{#if showSnippetModal}
  <ExternalPagesInstallModal
    {clientId}
    {shopifyShopDomain}
    connected={isConnected("external_pages")}
    connectedAt={connectedAt["external_pages"] || ""}
    on:installed={handleExternalPagesInstalled}
    on:close={() => (showSnippetModal = false)}
  />
{/if}

<style lang="postcss">
  /* Round 3 §2 layout — page sits on the cream canvas, content centered. */
  .connectors-canvas {
    @apply h-full w-full overflow-y-auto py-12 px-6;
    background: var(--color-bg);
  }
  .connectors-inner {
    @apply mx-auto w-full max-w-3xl;
  }
  .connectors-title {
    font-family: "Outfit", sans-serif;
    font-size: 30px;
    font-weight: 900;
    color: var(--color-text);
    letter-spacing: -0.3px;
    line-height: 1.15;
    margin-bottom: 10px;
  }
  .connectors-subhead {
    font-family: "Outfit", sans-serif;
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.6;
  }

  /* Single white card with 4px acid top accent. */
  .connector-list-card {
    position: relative;
    background: var(--color-elevated);
    border: 0.5px solid var(--color-border);
  }
  .connector-list-card::before {
    content: "";
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 4px;
    background: var(--color-acid);
    pointer-events: none;
    z-index: 1;
  }

  /* Each row sits directly on the white card; rows separated by hairlines —
     no sub-cream surface between rows. Disconnected rows get a soft warm tint
     so the eye lands on the action item. */
  .connector-row {
    display: grid;
    grid-template-columns: 28px 1fr auto;
    align-items: center;
    gap: 16px;
    padding: 18px 24px;
    border-bottom: 0.5px solid rgba(0, 0, 0, 0.06);
  }
  .connector-row:first-of-type {
    padding-top: 22px;
  }
  .connector-row:last-child {
    border-bottom: none;
  }
  .row-disconnected {
    background: rgba(255, 196, 117, 0.05);
  }

  .connector-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    display: block;
  }

  .connector-mid {
    @apply min-w-0;
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }

  .connector-name {
    font-family: "Space Mono", monospace;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text);
  }

  .connector-pill {
    font-family: "Space Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    padding: 3px 8px;
    background: var(--color-acid);
    color: #0A0A0A;
  }

  .connector-status-muted {
    font-family: "Space Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  .connector-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
</style>
