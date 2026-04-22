<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    onboardConnect,
    onboardActivate,
    onboardMe,
    connectedFromStackSelections,
  } from "$lib/bratrax/onboarding/api";
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";
  import AccountSelectionModal from "../../connectors/AccountSelectionModal.svelte";

  const FB_APP_ID = "3518566538435783";

  // ---------------------------------------------------------------------------
  // Platform definitions
  // ---------------------------------------------------------------------------
  interface Platform {
    id: string;
    name: string;
    type: "oauth" | "oauth_direct" | "client_sdk" | "selection" | "coming_soon";
    authUrlPath?: string;
    directAuthUrl?: string;
    color: string;
  }

  interface Category {
    title: string;
    subtitle: string;
    platforms: Platform[];
  }

  const categories: Category[] = [
    {
      title: "YOUR STORE",
      subtitle: "",
      platforms: [
        { id: "shopify", name: "Shopify", type: "selection", color: "#95BF47" },
      ],
    },
    {
      title: "AD PLATFORMS",
      subtitle: "Connect at least 1",
      platforms: [
        {
          id: "facebook_ads",
          name: "Facebook Ads",
          type: "client_sdk",
          color: "#1877F2",
        },
        {
          id: "google_ads",
          name: "Google Ads",
          type: "client_sdk",
          color: "#4285F4",
        },
        {
          id: "tiktok_ads",
          name: "TikTok Ads",
          type: "oauth",
          authUrlPath: "/bratrax/onboard/tiktok/auth-url",
          color: "#000000",
        },
        {
          id: "pinterest_ads",
          name: "Pinterest",
          type: "oauth",
          authUrlPath: "/bratrax/connectors/pinterest/auth-url",
          color: "#E60023",
        },
        {
          id: "amazon_ads",
          name: "Amazon Ads",
          type: "oauth",
          authUrlPath: "/bratrax/connectors/amazon-ads/auth-url",
          color: "#FF9900",
        },
      ],
    },
    {
      title: "EMAIL & SMS",
      subtitle: "Optional",
      platforms: [
        {
          id: "klaviyo",
          name: "Klaviyo",
          type: "oauth",
          authUrlPath: "/bratrax/connectors/klaviyo/auth-url",
          color: "#2D2D2D",
        },
      ],
    },
    {
      title: "SUBSCRIPTIONS",
      subtitle: "Optional",
      platforms: [
        { id: "recharge", name: "Recharge", type: "selection", color: "#5433FF" },
        { id: "bold", name: "Bold", type: "selection", color: "#FF6B35" },
        { id: "skio", name: "Skio", type: "selection", color: "#6366F1" },
        { id: "no_subscriptions", name: "None", type: "selection", color: "#9CA3AF" },
      ],
    },
    {
      title: "PAGES OUTSIDE SHOPIFY",
      subtitle: "",
      platforms: [
        {
          id: "shopify_only",
          name: "Everything is on Shopify",
          type: "selection",
          color: "#10B981",
        },
        {
          id: "external_pages",
          name: "I have external landing pages",
          type: "selection",
          color: "#F59E0B",
        },
      ],
    },
  ];

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------
  let connectedPlatforms: Set<string> = new Set();
  let selectedPlatforms: Set<string> = new Set(["shopify_only"]);
  let loading = "";
  let error = "";
  let activating = false;
  let clientId = "";

  const adPlatformIds = new Set(["facebook_ads", "google_ads", "tiktok_ads", "pinterest_ads", "amazon_ads"]);

  $: hasAdPlatform = [...connectedPlatforms].some((p) => adPlatformIds.has(p));

  function isConnected(id: string): boolean {
    return connectedPlatforms.has(id);
  }

  function isSelected(id: string): boolean {
    return selectedPlatforms.has(id);
  }

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------
  async function handleOAuthConnect(platform: Platform) {
    if (isConnected(platform.id)) return;
    error = "";
    loading = platform.id;

    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}${platform.authUrlPath}`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Failed to get auth URL for ${platform.name}`);
      const data = await res.json();

      // Store which platform we're connecting for the callback
      sessionStorage.setItem("onboard_oauth_platform", platform.id);
      sessionStorage.setItem("onboard_oauth_return", "/onboard/stack");

      // Platforms that use CSRF state (e.g. TikTok) echo it back in the callback URL.
      if (data.state) {
        sessionStorage.setItem(`${platform.id}_oauth_state`, data.state);
      }

      window.location.href = data.url;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = "";
    }
  }

  function handleSelectionToggle(platform: Platform) {
    // For radio-style groups (subscriptions, external pages)
    const subscriptionIds = new Set(["recharge", "bold", "skio", "no_subscriptions"]);
    const pagesIds = new Set(["shopify_only", "external_pages"]);

    if (subscriptionIds.has(platform.id)) {
      subscriptionIds.forEach((id) => selectedPlatforms.delete(id));
    } else if (pagesIds.has(platform.id)) {
      pagesIds.forEach((id) => selectedPlatforms.delete(id));
    }

    if (selectedPlatforms.has(platform.id)) {
      selectedPlatforms.delete(platform.id);
    } else {
      selectedPlatforms.add(platform.id);
    }
    selectedPlatforms = selectedPlatforms; // trigger reactivity
  }

  function handleDirectOAuth(platform: Platform) {
    if (isConnected(platform.id) || !platform.directAuthUrl) return;
    error = "";
    loading = platform.id;

    const redirect = encodeURIComponent(
      `${window.location.origin}/onboard/stack`,
    );
    const state = encodeURIComponent(platform.id);
    const url = platform.directAuthUrl.replace("{redirect}", redirect) + `&state=${state}`;

    sessionStorage.setItem("onboard_oauth_platform", platform.id);
    window.location.href = url;
  }

  // ---------------------------------------------------------------------------
  // Google Ads: popup SDK flow with account selection
  // ---------------------------------------------------------------------------
  let showAccountModal = false;
  let accountModalLoading = false;
  let accountModalPlatform = "";
  let accountModalAccounts: Array<{ id: string; name: string }> = [];
  let googleAuthCode = "";
  let fbAccessToken = "";
  let fbEmail = "";
  let fbSdkReady = false;
  let googleManagerMap: Record<string, string> = {};
  let tiktokState = "";

  // Single onMount: seed client_id + connected platforms from the server,
  // handle any OAuth return params, then load the Google/Facebook SDKs.
  onMount(async () => {
    // 1. Fetch client_id + onboarding state from the backend.
    const me = await onboardMe();
    if (me?.client_id) {
      clientId = me.client_id;

      // Seed connectedPlatforms from both sources for robustness:
      //   connected_platforms — legacy array, written by every OAuth handler
      //   stack_selections    — authoritative; presence of a {platform}_credentials
      //                         key means credentials are on file
      for (const p of me.connected_platforms || []) {
        if (typeof p === "object" && p.platform) {
          connectedPlatforms.add(p.platform);
        }
      }
      for (const p of connectedFromStackSelections(me.stack_selections)) {
        connectedPlatforms.add(p);
      }
      connectedPlatforms = connectedPlatforms;
    }

    // 2. Legacy OAuth-return: /connectors/callback/[platform] can redirect
    //    back here with ?connected=<platform>.
    const justConnected = $page.url.searchParams.get("connected");
    if (justConnected) {
      connectedPlatforms.add(justConnected);
      connectedPlatforms = connectedPlatforms;
      if (clientId) {
        try {
          await onboardConnect(clientId, justConnected);
        } catch {
          // Non-fatal — connection is typically already recorded by the
          // token endpoint that served the callback.
        }
      }
    }

    // 3. TikTok Business redirects back here with ?auth_code=…&state=… when
    //    the tenant finishes authorising. Verify CSRF state, then open the
    //    same AccountSelectionModal used by Google/Facebook.
    const tiktokAuthCode = $page.url.searchParams.get("auth_code");
    const tiktokStateParam = $page.url.searchParams.get("state");
    const expectedTiktokState = sessionStorage.getItem("tiktok_ads_oauth_state");
    if (tiktokAuthCode && tiktokStateParam && expectedTiktokState === tiktokStateParam) {
      await handleTikTokReturn(tiktokAuthCode, tiktokStateParam);
    }

    // 4. Load Google Identity Services SDK (Google Ads popup OAuth).
    if (!document.getElementById("gis-sdk")) {
      const script = document.createElement("script");
      script.id = "gis-sdk";
      script.src = "https://accounts.google.com/gsi/client";
      script.async = true;
      document.head.appendChild(script);
    }

    // 5. Load Facebook SDK (Facebook Ads popup OAuth).
    loadFacebookSdk();
  });

  async function handleTikTokReturn(code: string, state: string) {
    error = "";
    loading = "tiktok_ads";
    try {
      const host = get(runtime).host;
      const res = await fetch(
        `${host}/bratrax/onboard/tiktok/accounts?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
        { credentials: "include" },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? "Failed to list TikTok advertisers");
      }
      const body = (await res.json()) as {
        state: string;
        advertisers: Array<{ id: string; name: string }>;
      };
      tiktokState = body.state;
      accountModalAccounts = body.advertisers.map((a) => ({
        id: a.id,
        name: a.name || a.id,
      }));
      accountModalPlatform = "TikTok Ads";
      showAccountModal = true;

      // Strip the callback params so a refresh doesn't re-trigger.
      const url = new URL(window.location.href);
      url.searchParams.delete("auth_code");
      url.searchParams.delete("state");
      url.searchParams.delete("scopes");
      window.history.replaceState({}, "", url.toString());
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
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
            // Fetch account hierarchy via onboarding endpoint (no auth issues)
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

  async function handleAccountSelection(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      const host = get(runtime).host;
      const isFacebook = accountModalPlatform === "Facebook Ads";
      const isTikTok = accountModalPlatform === "TikTok Ads";

      let endpoint: string;
      let body: Record<string, unknown>;
      let connectedKey: string;

      if (isTikTok) {
        endpoint = `${host}/bratrax/onboard/tiktok/connect`;
        body = {
          state: tiktokState,
          client_id: clientId,
          selected_advertiser_ids: selected.map((a) => a.accountId),
        };
        connectedKey = "tiktok_ads";
      } else if (isFacebook) {
        endpoint = `${host}/bratrax/onboard/facebook/connect`;
        body = {
          short_lived_token: fbAccessToken,
          email: fbEmail,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "facebook_ads";
      } else {
        endpoint = `${host}/bratrax/onboard/google/connect`;
        body = {
          code: googleAuthCode,
          client_id: clientId,
          selectedAccounts: selected.map((a) => ({
            ...a,
            managerAccountId: googleManagerMap[a.accountId] || "",
          })),
        };
        connectedKey = "google_ads";
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

      connectedPlatforms.add(connectedKey);
      connectedPlatforms = connectedPlatforms;
      showAccountModal = false;

      if (isTikTok) {
        sessionStorage.removeItem("tiktok_ads_oauth_state");
        tiktokState = "";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      accountModalLoading = false;
    }
  }

  function handleCardClick(platform: Platform) {
    if (platform.type === "oauth") {
      handleOAuthConnect(platform);
    } else if (platform.type === "oauth_direct") {
      handleDirectOAuth(platform);
    } else if (platform.type === "client_sdk") {
      if (platform.id === "google_ads") handleGoogleAdsConnect();
      else if (platform.id === "facebook_ads") handleFacebookConnect();
    } else if (platform.type === "coming_soon") {
      // Do nothing
    } else {
      handleSelectionToggle(platform);
    }
  }

  async function handleActivate() {
    error = "";
    activating = true;

    try {
      // Record selections as stack_selections
      const stackSelections: Record<string, string[]> = {
        connected: [...connectedPlatforms],
        subscriptions: [...selectedPlatforms].filter((p) =>
          ["recharge", "bold", "skio", "no_subscriptions"].includes(p),
        ),
        pages: [...selectedPlatforms].filter((p) =>
          ["shopify_only", "external_pages"].includes(p),
        ),
      };

      await onboardActivate(clientId, stackSelections);
      await goto("/onboard/loading");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      activating = false;
    }
  }

</script>

<div class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12">
  <div class="relative w-full max-w-2xl border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        ONBOARDING
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Build your <span class="font-serif italic text-bratrax-acid">stack</span>
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        Connect the tools you use. We'll set up your analytics automatically.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="flex flex-col gap-6">
      {#each categories as category}
        <div>
          <div class="mb-2 flex items-baseline gap-2">
            <span class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
              {category.title}
            </span>
            {#if category.subtitle}
              <span class="font-mono text-[10px] text-bratrax-text-muted">({category.subtitle})</span>
            {/if}
          </div>
          <div class="flex flex-wrap gap-2">
            {#each category.platforms as platform}
              <button
                on:click={() => handleCardClick(platform)}
                disabled={loading === platform.id || platform.id === "shopify" || platform.type === "coming_soon"}
                class="flex items-center gap-2 border px-4 py-2.5 font-mono text-xs font-bold uppercase tracking-wider transition-all
                  {isConnected(platform.id)
                    ? 'border-bratrax-acid/50 bg-bratrax-acid/10 text-bratrax-acid'
                    : isSelected(platform.id)
                      ? 'border-bratrax-cyan/50 bg-bratrax-cyan/10 text-bratrax-cyan'
                      : platform.type === 'coming_soon'
                        ? 'border-bratrax-border bg-bratrax-bg text-bratrax-text-muted cursor-not-allowed'
                        : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover'}
                  {loading === platform.id ? 'opacity-50' : ''}
                  {platform.id === 'shopify' ? 'cursor-default' : ''}"
              >
                <span
                  class="h-3 w-3"
                  style="background-color: {platform.color}"
                ></span>
                {#if isConnected(platform.id)}
                  <svg class="h-4 w-4 text-bratrax-acid" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                {/if}
                {platform.name}
                {#if platform.type === "coming_soon"}
                  <span class="text-bratrax-text-muted normal-case tracking-normal">coming soon</span>
                {:else if loading === platform.id}
                  <span class="text-bratrax-text-muted normal-case tracking-normal">connecting...</span>
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>

    <div class="mt-8 flex items-center justify-between border-t border-bratrax-border pt-6">
      <p class="font-mono text-[10px] text-bratrax-text-muted">
        {#if !hasAdPlatform}
          Connect at least 1 ad platform to continue
        {:else}
          Ready to set up your analytics
        {/if}
      </p>
      <button
        on:click={handleActivate}
        disabled={!hasAdPlatform || activating}
        class="bg-bratrax-acid px-6 py-3 font-mono text-xs font-bold uppercase tracking-[1.5px] text-bratrax-bg transition-all hover:opacity-90 hover:-translate-y-px disabled:opacity-50"
      >
        {activating ? "SETTING UP..." : "SET UP MY ANALYTICS →"}
      </button>
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
