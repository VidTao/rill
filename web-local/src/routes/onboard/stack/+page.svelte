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
  import TrackingTemplateConfirmModal from "../../connectors/TrackingTemplateConfirmModal.svelte";
  import { getOAuthConfig } from "$lib/bratrax/onboarding/api";

  // Public OAuth client IDs — fetched from /onboard/oauth-config on mount.
  // Empty until the fetch completes; OAuth-init click handlers guard against
  // empty values to avoid calling SDKs with an empty appId / client_id.
  let fbAppId = "";
  let googleClientId = "";

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
          authUrlPath: "/bratrax/onboard/klaviyo/auth-url",
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

  // Reactive declarations so the template sees connectedPlatforms /
  // selectedPlatforms as direct dependencies. Using a function declaration
  // hides the Set behind an opaque call, so Svelte only re-renders when
  // some OTHER reactive variable changes (observed: card didn't turn green
  // after connect, only refreshed when `loading` flipped on the next click).
  $: isConnected = (id: string): boolean => connectedPlatforms.has(id);
  $: isSelected = (id: string): boolean => selectedPlatforms.has(id);

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
  let klaviyoState = "";

  // Google Ads tracking_url_template confirmation modal — wedged between
  // /onboard/google/connect and /onboard/activate so each leaf customer gets
  // the canonical Bratrax tracking template before activation.
  let showTrackingTemplateModal = false;
  let trackingTemplateLoading = false;
  let trackingTemplateProposed = "";
  let trackingTemplateConflicts: Array<{
    customer_id: string;
    descriptive_name: string;
    current: string;
  }> = [];
  let trackingTemplateAllLeafIds: string[] = [];
  let trackingTemplateUnaffectedIds: string[] = [];
  type TrackingTemplateChoice = "replace" | "skip" | "cancel";
  let trackingTemplateResolve:
    | ((choice: TrackingTemplateChoice) => void)
    | null = null;

  // True when this page was hit purely as the OAuth-callback dispatcher for a
  // flow initiated elsewhere (e.g. /connectors). The "Build your stack" UI is
  // hidden in this mode — we render a minimal "Completing connection…" screen
  // and let the AccountSelectionModal pop on top, then goto(returnTo) once
  // the user submits. Detected synchronously in onMount before paint.
  let isOAuthBounce = false;

  // Single source of truth for "which platforms are connected": the server.
  // Rebuilds the connectedPlatforms Set as a new reference so Svelte
  // reactivity always fires (Set.add + self-assign is unreliable).
  async function refreshFromServer() {
    const me = await onboardMe();
    if (!me?.client_id) return;
    clientId = me.client_id;

    const next = new Set<string>();
    for (const p of me.connected_platforms || []) {
      if (typeof p === "object" && p.platform) {
        next.add(p.platform);
      }
    }
    for (const p of connectedFromStackSelections(me.stack_selections)) {
      next.add(p);
    }
    connectedPlatforms = next;
  }

  onMount(async () => {
    // 0. Detect "OAuth-callback dispatcher" mode synchronously, before any
    //    awaits. If the URL has an OAuth-callback param AND the originating
    //    page wants us to bounce somewhere else, hide the build-your-stack UI
    //    so the user only sees the AccountSelectionModal during the flash.
    const hasCallback =
      $page.url.searchParams.has("code") ||
      $page.url.searchParams.has("auth_code");
    const returnTo = sessionStorage.getItem("onboard_oauth_return");
    isOAuthBounce =
      hasCallback && returnTo !== null && returnTo !== "/onboard/stack";

    // 1. Pull initial state from the server.
    await refreshFromServer();

    // 2. Legacy OAuth-return: /connectors/callback/[platform] redirects here
    //    with ?connected=<platform>. Record it, then re-read from server.
    const justConnected = $page.url.searchParams.get("connected");
    if (justConnected && clientId) {
      try {
        await onboardConnect(clientId, justConnected);
      } catch {
        // Non-fatal — the token endpoint likely already recorded it.
      }
      await refreshFromServer();
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

    // 3b. Klaviyo redirects back with ?code=…&state=…. Verify CSRF state,
    //     exchange the code, and open AccountSelectionModal.
    const klaviyoCode = $page.url.searchParams.get("code");
    const klaviyoStateParam = $page.url.searchParams.get("state");
    const expectedKlaviyoState = sessionStorage.getItem("klaviyo_oauth_state");
    if (
      klaviyoCode &&
      klaviyoStateParam &&
      expectedKlaviyoState === klaviyoStateParam
    ) {
      await handleKlaviyoReturn(klaviyoCode, klaviyoStateParam);
    }

    // 4. Load Google Identity Services SDK (Google Ads popup OAuth).
    if (!document.getElementById("gis-sdk")) {
      const script = document.createElement("script");
      script.id = "gis-sdk";
      script.src = "https://accounts.google.com/gsi/client";
      script.async = true;
      document.head.appendChild(script);
    }

    // 5. Fetch public OAuth client IDs (BUG-01: not hardcoded in source).
    try {
      const cfg = await getOAuthConfig();
      fbAppId = cfg.fb_app_id;
      googleClientId = cfg.google_client_id;
    } catch {
      // Non-fatal — connect-button handlers guard against empty values.
    }

    // 6. Load Facebook SDK (Facebook Ads popup OAuth).
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

  async function handleKlaviyoReturn(code: string, state: string) {
    error = "";
    loading = "klaviyo";
    try {
      const host = get(runtime).host;
      const res = await fetch(
        `${host}/bratrax/onboard/klaviyo/accounts?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
        { credentials: "include" },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? "Failed to list Klaviyo accounts");
      }
      const body = (await res.json()) as {
        state: string;
        accounts: Array<{ id: string; name: string }>;
      };
      klaviyoState = body.state;
      accountModalAccounts = body.accounts.map((a) => ({
        id: a.id,
        name: a.name || a.id,
      }));
      accountModalPlatform = "Klaviyo";
      showAccountModal = true;

      // Strip the callback params so a refresh doesn't re-trigger.
      const url = new URL(window.location.href);
      url.searchParams.delete("code");
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

  // ---------------------------------------------------------------------------
  // Google Ads tracking_url_template flow
  // ---------------------------------------------------------------------------
  // Per-account metadata that does NOT cascade from MCC managers to children;
  // each leaf customer needs the template set individually for Bratrax
  // attribution to work. Run after /onboard/google/connect succeeds and before
  // the user advances to /onboard/business → /onboard/activate.
  async function runGoogleTrackingTemplateFlow(customerIds: string[]): Promise<void> {
    if (!customerIds.length) return;
    const host = get(runtime).host;

    const previewRes = await fetch(
      `${host}/bratrax/onboard/google/preview-tracking-templates`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ code: googleAuthCode, customer_ids: customerIds }),
      },
    );
    if (!previewRes.ok) {
      // Best-effort: surface in console but don't block onboarding.
      console.warn("Tracking-template preview failed", previewRes.status);
      return;
    }

    interface PreviewItem {
      customer_id: string;
      descriptive_name: string;
      is_manager: boolean;
      current: string;
      proposed: string;
      needs_confirm: boolean;
      error?: string;
    }
    const previewBody = (await previewRes.json()) as { data?: PreviewItem[] };
    const items: PreviewItem[] = previewBody.data || [];

    const allLeafIds = items
      .filter((i) => !i.is_manager && !i.error)
      .map((i) => i.customer_id);
    const conflictItems = items.filter((i) => i.needs_confirm);
    const conflictIds = new Set(conflictItems.map((c) => c.customer_id));
    const unaffectedIds = allLeafIds.filter((id) => !conflictIds.has(id));

    if (conflictItems.length === 0) {
      // No conflicts — silently apply for everyone (sets where empty,
      // skips on the server where already_set, skips MCCs).
      await applyTrackingTemplates(allLeafIds, false);
      return;
    }

    // Show modal and wait for user choice.
    trackingTemplateProposed = items[0]?.proposed || "";
    trackingTemplateConflicts = conflictItems.map((c) => ({
      customer_id: c.customer_id,
      descriptive_name: c.descriptive_name,
      current: c.current,
    }));
    trackingTemplateAllLeafIds = allLeafIds;
    trackingTemplateUnaffectedIds = unaffectedIds;
    showTrackingTemplateModal = true;

    const choice: TrackingTemplateChoice = await new Promise((resolve) => {
      trackingTemplateResolve = resolve;
    });
    trackingTemplateResolve = null;

    if (choice === "cancel") {
      showTrackingTemplateModal = false;
      throw new Error("Tracking template setup cancelled");
    }

    trackingTemplateLoading = true;
    try {
      if (choice === "replace") {
        await applyTrackingTemplates(trackingTemplateAllLeafIds, true);
      } else {
        // skip: only apply to accounts that don't have a conflicting current
        // template. The conflicted accounts will not get Bratrax attribution
        // until their template is set manually in Google Ads UI.
        await applyTrackingTemplates(trackingTemplateUnaffectedIds, false);
      }
    } finally {
      trackingTemplateLoading = false;
      showTrackingTemplateModal = false;
    }
  }

  async function applyTrackingTemplates(
    customerIds: string[],
    forceReplace: boolean,
  ): Promise<void> {
    if (!customerIds.length) return;
    const host = get(runtime).host;
    const res = await fetch(
      `${host}/bratrax/onboard/google/apply-tracking-templates`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          code: googleAuthCode,
          client_id: clientId,
          customer_ids: customerIds,
          force_replace: forceReplace,
        }),
      },
    );
    if (res.status === 409) {
      // Defensive: shouldn't fire when preview was honored, but possible if
      // a customer's template changed between preview and apply.
      const body = await res.json().catch(() => ({}));
      console.warn("Tracking-template apply returned 409", body);
      throw new Error("Tracking template conflict");
    }
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(
        ((body as Record<string, unknown>).error as string) ||
          "Failed to apply tracking templates",
      );
    }
  }

  function handleTrackingTemplateChoice(choice: TrackingTemplateChoice) {
    if (trackingTemplateResolve) trackingTemplateResolve(choice);
  }

  async function handleAccountSelection(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      const host = get(runtime).host;
      const isFacebook = accountModalPlatform === "Facebook Ads";
      const isTikTok = accountModalPlatform === "TikTok Ads";
      const isKlaviyo = accountModalPlatform === "Klaviyo";

      let endpoint: string;
      let body: Record<string, unknown>;
      let connectedKey: string;

      if (isKlaviyo) {
        endpoint = `${host}/bratrax/onboard/klaviyo/connect`;
        body = {
          state: klaviyoState,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "klaviyo";
      } else if (isTikTok) {
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

      showAccountModal = false;

      // Google Ads only: ensure each connected leaf customer has the canonical
      // Bratrax tracking_url_template (and the user has confirmed if their
      // existing template would be overwritten). Best-effort — failures here
      // surface as a warning but do not unwind the connection.
      if (connectedKey === "google_ads") {
        try {
          const customerIds = selected.map((a) => a.accountId);
          await runGoogleTrackingTemplateFlow(customerIds);
        } catch (e) {
          console.warn("Tracking template flow:", e);
        }
      }

      if (isTikTok) {
        sessionStorage.removeItem("tiktok_ads_oauth_state");
        tiktokState = "";
      }
      if (isKlaviyo) {
        sessionStorage.removeItem("klaviyo_oauth_state");
        klaviyoState = "";
      }

      // Re-pull from the server so the card renders from the source of
      // truth. Also guarantees a fresh Set reference for Svelte reactivity.
      await refreshFromServer();

      // Honor the return-to set by the originating page. When the OAuth was
      // initiated from /connectors (or any other page), bounce back there
      // after the modal completes. Default "/onboard/stack" is a no-op.
      const returnTo = sessionStorage.getItem("onboard_oauth_return") || "/onboard/stack";
      sessionStorage.removeItem("onboard_oauth_return");
      if (returnTo && returnTo !== "/onboard/stack") {
        await goto(returnTo);
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
      // Store stack selections in sessionStorage for the business profile page
      // to include when it triggers activation.
      const stackSelections: Record<string, string[]> = {
        connected: [...connectedPlatforms],
        subscriptions: [...selectedPlatforms].filter((p) =>
          ["recharge", "bold", "skio", "no_subscriptions"].includes(p),
        ),
        pages: [...selectedPlatforms].filter((p) =>
          ["shopify_only", "external_pages"].includes(p),
        ),
      };
      sessionStorage.setItem("onboard_stack_selections", JSON.stringify(stackSelections));

      // Proceed to business profile questionnaire.
      // Activation (compile+deploy) is triggered from there after the user
      // answers the business questions.
      await goto("/onboard/business");
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      activating = false;
    }
  }

</script>

{#if isOAuthBounce}
  <div class="flex h-screen w-screen items-center justify-center bg-bratrax-bg">
    <div class="text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        ONBOARDING
      </div>
      <div class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted">
        Completing connection…
      </div>
    </div>
  </div>
{:else}
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
{/if}

<AccountSelectionModal
  open={showAccountModal}
  loading={accountModalLoading}
  platformName={accountModalPlatform}
  accounts={accountModalAccounts}
  onSubmit={handleAccountSelection}
  onClose={() => { showAccountModal = false; }}
/>

<TrackingTemplateConfirmModal
  open={showTrackingTemplateModal}
  loading={trackingTemplateLoading}
  proposed={trackingTemplateProposed}
  conflicts={trackingTemplateConflicts}
  onReplaceAll={() => handleTrackingTemplateChoice("replace")}
  onSkip={() => handleTrackingTemplateChoice("skip")}
  onCancel={() => handleTrackingTemplateChoice("cancel")}
/>
