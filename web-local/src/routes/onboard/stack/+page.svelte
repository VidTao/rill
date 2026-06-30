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
  import FbUrlTagsConfirmModal from "../../connectors/FbUrlTagsConfirmModal.svelte";
  import ExternalPagesBuilderPickerModal from "../../connectors/ExternalPagesBuilderPickerModal.svelte";
  import FunnelishInstallModal from "../../connectors/FunnelishInstallModal.svelte";
  import TaboolaCredentialModal from "../../connectors/TaboolaCredentialModal.svelte";
  import OutbrainLoginModal from "../../connectors/OutbrainLoginModal.svelte";
  import { getOAuthConfig } from "$lib/bratrax/onboarding/api";
  import WooTrackingInstall from "$lib/bratrax/onboarding/WooTrackingInstall.svelte";

  // ---------------------------------------------------------------------------
  // Onboarding-time tracking-template flows (paused — pending team alignment)
  // ---------------------------------------------------------------------------
  // When true, the corresponding flow runs after a successful platform
  // connect: previews existing tracking templates / url_tags and (with user
  // confirmation if needed) writes the canonical Bratrax value to every
  // selected leaf customer (Google) or campaign (Facebook).
  //
  // Both default to false — the connection itself completes normally; only
  // the post-connect template-write step is skipped. Backend endpoints,
  // audit table, modal components, and helper functions stay in place; flip
  // a flag to true once aligned with the team.
  const ENABLE_GOOGLE_TRACKING_TEMPLATES = true;
  const ENABLE_FB_URL_TAGS = false;

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
    // "credential_modal" opens an in-page form (Taboola, Outbrain) whose
    // submit triggers a direct credential→token exchange on the backend,
    // bypassing the OAuth redirect dance entirely.
    type:
      | "oauth"
      | "oauth_direct"
      | "client_sdk"
      | "selection"
      | "coming_soon"
      | "credential_modal";
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
        {
          id: "woocommerce",
          name: "WooCommerce",
          type: "selection",
          color: "#7F54B3",
        },
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
          id: "bing_ads",
          name: "Microsoft Bing Ads",
          type: "oauth",
          authUrlPath: "/bratrax/onboard/bing-ads/auth-url",
          color: "#00A4EF",
        },
        {
          id: "taboola",
          name: "Taboola",
          type: "credential_modal",
          color: "#1376DC",
        },
        {
          id: "outbrain",
          name: "Outbrain",
          type: "credential_modal",
          color: "#EE6E33",
        },
        {
          id: "pinterest_ads",
          name: "Pinterest",
          type: "oauth",
          authUrlPath: "/bratrax/onboard/pinterest/auth-url",
          color: "#E60023",
        },
        {
          id: "amazon_ads",
          name: "Amazon Ads",
          type: "coming_soon",
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
        {
          id: "recharge",
          name: "Recharge",
          type: "coming_soon",
          color: "#5433FF",
        },
        { id: "bold", name: "Bold", type: "coming_soon", color: "#FF6B35" },
        { id: "skio", name: "Skio", type: "coming_soon", color: "#6366F1" },
        {
          id: "no_subscriptions",
          name: "None",
          type: "selection",
          color: "#9CA3AF",
        },
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
  let shopifyShopDomain = "";
  let connectedAt: Record<string, string> = {};

  // Two-step external-pages flow: clicking "I have external landing pages"
  // opens the builder picker; picking Funnelish opens the install snippet
  // modal. Each builder gets its own install pixel + snippet (we ship
  // Funnelish first; ClickFunnels / GoHighLevel / Other are placeholders).
  let showBuilderPickerModal = false;
  let showFunnelishModal = false;

  // Set of builder ids that map to the "external_pages" umbrella. Keep this
  // in sync with ExternalPagesBuilderPickerModal — adding a new builder
  // requires updating both this set and the picker's `available` list.
  const externalBuilderIds = new Set(["funnelish"]);
  $: hasExternalBuilder = [...connectedPlatforms].some((p) =>
    externalBuilderIds.has(p),
  );

  const adPlatformIds = new Set([
    "facebook_ads",
    "google_ads",
    "tiktok_ads",
    "bing_ads",
    "taboola",
    "outbrain",
    "pinterest_ads",
    "amazon_ads",
  ]);

  // Ids that behave as radio buttons (one-of-N selection) — distinct from
  // `type === "selection"`, which also includes the display-only Shopify card.
  // Keep in sync with the two Sets in handleSelectionToggle().
  const radioToggleIds = new Set([
    "shopify_only",
    "external_pages",
    "recharge",
    "bold",
    "skio",
    "no_subscriptions",
  ]);

  $: hasAdPlatform = [...connectedPlatforms].some((p) => adPlatformIds.has(p));

  // Reactive declarations so the template sees connectedPlatforms /
  // selectedPlatforms as direct dependencies. Using a function declaration
  // hides the Set behind an opaque call, so Svelte only re-renders when
  // some OTHER reactive variable changes (observed: card didn't turn green
  // after connect, only refreshed when `loading` flipped on the next click).
  // "external_pages" is a UI umbrella — there's no platform of that name in
  // connected_platforms. Treat it as connected when any underlying builder
  // (Funnelish, future ClickFunnels / GoHighLevel) is connected.
  $: isConnected = (id: string): boolean => {
    if (id === "external_pages") return hasExternalBuilder;
    return connectedPlatforms.has(id);
  };
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
      if (!res.ok)
        throw new Error(`Failed to get auth URL for ${platform.name}`);
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
    const subscriptionIds = new Set([
      "recharge",
      "bold",
      "skio",
      "no_subscriptions",
    ]);
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
    const url =
      platform.directAuthUrl.replace("{redirect}", redirect) +
      `&state=${state}`;

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
  let bingAdsState = "";
  let pinterestAdsState = "";
  let taboolaState = "";
  let outbrainState = "";

  // Credential-modal state (Taboola / Outbrain). Mirrors the pattern on
  // /connectors — modal stays open on error so the user can retry without
  // re-typing credentials, closes only on success.
  let showTaboolaModal = false;
  let taboolaModalLoading = false;
  let taboolaModalError = "";

  let showOutbrainModal = false;
  let outbrainModalLoading = false;
  let outbrainModalError = "";

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

  // Facebook url_tags confirmation modal — wedged between
  // /onboard/facebook/connect and /onboard/activate.
  interface FbAccountPreview {
    act_id: string;
    account_name: string;
    total_campaigns: number;
    with_url_tags: number;
    conflicting: Array<{ campaign_id: string; name: string; current: string }>;
    conflicting_count: number;
  }
  let showFbUrlTagsModal = false;
  let fbUrlTagsLoading = false;
  let fbUrlTagsProposed = "";
  let fbUrlTagsAccounts: FbAccountPreview[] = [];
  let fbUrlTagsAdAccountIds: string[] = [];
  type FbUrlTagsChoice = "replace" | "empty_only" | "cancel";
  let fbUrlTagsResolve: ((choice: FbUrlTagsChoice) => void) | null = null;

  // True when this page was hit purely as the OAuth-callback dispatcher for a
  // flow initiated elsewhere (e.g. /connectors). The "Build your stack" UI is
  // hidden in this mode — we render a minimal "Completing connection…" screen
  // and let the AccountSelectionModal pop on top, then goto(returnTo) once
  // the user submits. Detected synchronously in onMount before paint.
  let isOAuthBounce = false;

  // Single source of truth for "which platforms are connected": the server.
  // Rebuilds the connectedPlatforms Set as a new reference so Svelte
  // reactivity always fires (Set.add + self-assign is unreliable).
  function explainOAuthError(
    code: string,
    desc: string | null,
    bingClientId: string,
  ): string {
    // Microsoft AADSTS error codes have specific guidance — surface the
    // actionable fix rather than the wall-of-text Azure trace string.
    // For unknown codes, fall back to the provider's own description.
    if (desc && desc.includes("AADSTS650052")) {
      // Service-principal-missing: enterprise tenants need an admin to
      // pre-consent so Azure provisions the Microsoft Advertising service
      // principal in the tenant. The admin-consent URL provisions it in
      // one click.
      const adminUrl = bingClientId
        ? `https://login.microsoftonline.com/common/adminconsent?client_id=${bingClientId}`
        : "(admin-consent URL unavailable — OAuth config failed to load)";
      return (
        "Microsoft Ads connection failed: your organization hasn't consented " +
        "to the Microsoft Advertising service yet. An Azure AD admin needs to " +
        `visit this URL once to grant tenant-level consent, then you can retry:\n\n${adminUrl}\n\n` +
        "(Error AADSTS650052 — service principal not provisioned in your tenant.)"
      );
    }
    if (desc && desc.includes("AADSTS65001")) {
      return (
        "Microsoft Ads connection failed: the user or admin hasn't consented to " +
        "the requested permissions. Re-run the OAuth flow and click 'Consent on " +
        "behalf of your organization' if you're an admin, otherwise ask your IT " +
        "admin to grant consent. (Error AADSTS65001.)"
      );
    }
    if (code === "access_denied") {
      return "OAuth cancelled — you declined to authorise the connection. Retry from the platform card to try again.";
    }
    // Generic fallback. Strip Microsoft trace IDs from the description so
    // the error stays readable; users don't need the GUID-soup.
    const cleanedDesc = (desc || "")
      .replace(/Trace ID:\s*[\w-]+\.?\s*/g, "")
      .replace(/Correlation ID:\s*[\w-]+\.?\s*/g, "")
      .replace(/Timestamp:\s*[\d:\sZT-]+\.?\s*/g, "")
      .trim();
    return `OAuth error (${code}): ${cleanedDesc || "no description provided by the OAuth provider"}`;
  }

  async function refreshFromServer() {
    const me = await onboardMe();
    if (!me?.client_id) return;
    clientId = me.client_id;

    const stack = (me.stack_selections || {}) as Record<string, unknown>;
    const creds = stack.shopify_credentials as { shop?: string } | undefined;
    shopifyShopDomain = creds?.shop ?? "";

    const next = new Set<string>();
    const dates: Record<string, string> = {};
    for (const p of me.connected_platforms || []) {
      if (typeof p === "object" && p.platform) {
        next.add(p.platform);
        if (p.connected_at) dates[p.platform] = p.connected_at;
      }
    }
    for (const p of connectedFromStackSelections(me.stack_selections)) {
      next.add(p);
    }
    connectedPlatforms = next;
    connectedAt = dates;

    // If any external-pages builder (Funnelish, …) is connected, default
    // the "pages" radio to "I have external landing pages" on hard refresh
    // so the card renders correctly and the user doesn't have to re-click.
    const hasExternalBuilderNow = [...next].some((p) =>
      externalBuilderIds.has(p),
    );
    if (hasExternalBuilderNow) {
      selectedPlatforms.delete("shopify_only");
      selectedPlatforms.add("external_pages");
      selectedPlatforms = selectedPlatforms;
    }
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

    // 0b. WooCommerce wc-auth returns here (when initiated during onboarding)
    //     with ?success=…&user_id=<signed token> appended. The key pair already
    //     arrived via the server-to-server callback and the step advanced to
    //     platforms_connected, so just scrub the params so a refresh / back
    //     button doesn't carry the token around. Not an OAuth callback (no
    //     code/auth_code), so this never collides with the handlers below.
    if (
      $page.url.searchParams.has("user_id") ||
      $page.url.searchParams.has("success")
    ) {
      const wooUrl = new URL(window.location.href);
      wooUrl.searchParams.delete("success");
      wooUrl.searchParams.delete("user_id");
      window.history.replaceState({}, "", wooUrl.toString());
    }

    // 0a. OAuth providers return ?error=…&error_description=… when the user
    //     cancels, the consent fails, or — most commonly for Microsoft —
    //     the user's tenant lacks a service principal for the requested
    //     service (AADSTS650052). Surface a useful error instead of silently
    //     landing on the build-your-stack screen with no feedback.
    const oauthError = $page.url.searchParams.get("error");
    const oauthErrorDesc = $page.url.searchParams.get("error_description");
    if (oauthError) {
      // Fetch the OAuth config now so the AADSTS650052 branch can render a
      // real admin-consent URL inline. Best-effort: a failed fetch only
      // degrades the URL placeholder, not the rest of the error message.
      let bingClientIdForError = "";
      try {
        const cfg = await getOAuthConfig();
        bingClientIdForError = cfg.bing_ads_client_id || "";
      } catch {
        // Non-fatal — fall through to the unavailable-URL message branch.
      }
      const errorMsg = explainOAuthError(
        oauthError,
        oauthErrorDesc,
        bingClientIdForError,
      );
      const where = sessionStorage.getItem("onboard_oauth_return");
      // Strip the params so a refresh doesn't re-show the error and we
      // don't leak Microsoft trace/correlation IDs to address-bar history.
      const url = new URL(window.location.href);
      url.searchParams.delete("error");
      url.searchParams.delete("error_description");
      url.searchParams.delete("error_uri");
      url.searchParams.delete("state");
      window.history.replaceState({}, "", url.toString());
      // Clean up pending OAuth sessionStorage so the user can retry from a
      // clean slate (stale CSRF state would block the next attempt).
      sessionStorage.removeItem("bing_ads_oauth_state");
      sessionStorage.removeItem("klaviyo_oauth_state");
      sessionStorage.removeItem("tiktok_ads_oauth_state");
      sessionStorage.removeItem("onboard_oauth_return");
      sessionStorage.removeItem("onboard_oauth_platform");
      isOAuthBounce = false;
      // If the flow originated from /connectors, push the error through
      // sessionStorage and bounce back — leaving the user stranded on
      // /onboard/stack with an error message would be a confusing UX.
      if (where && where !== "/onboard/stack") {
        sessionStorage.setItem("connectors_error", errorMsg);
        await goto(where);
        return;
      }
      error = errorMsg;
      return;
    }

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
    const expectedTiktokState = sessionStorage.getItem(
      "tiktok_ads_oauth_state",
    );
    if (
      tiktokAuthCode &&
      tiktokStateParam &&
      expectedTiktokState === tiktokStateParam
    ) {
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

    // 3c. Microsoft Bing Ads redirects back with ?code=…&state=…. Disambiguated
    //     from Klaviyo (which uses the same param names) by which sessionStorage
    //     state key matches the returned state.
    const bingCode = $page.url.searchParams.get("code");
    const bingStateParam = $page.url.searchParams.get("state");
    const expectedBingState = sessionStorage.getItem("bing_ads_oauth_state");
    if (bingCode && bingStateParam && expectedBingState === bingStateParam) {
      await handleBingAdsReturn(bingCode, bingStateParam);
    }

    // 3d. Pinterest redirects back with ?code=…&state=… (same param names as
    //     Klaviyo/Bing). Disambiguated by the sessionStorage state-key match.
    const pinterestCode = $page.url.searchParams.get("code");
    const pinterestStateParam = $page.url.searchParams.get("state");
    const expectedPinterestState = sessionStorage.getItem(
      "pinterest_ads_oauth_state",
    );
    if (
      pinterestCode &&
      pinterestStateParam &&
      expectedPinterestState === pinterestStateParam
    ) {
      await handlePinterestReturn(pinterestCode, pinterestStateParam);
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
        throw new Error(
          (body as { error?: string }).error ??
            "Failed to list TikTok advertisers",
        );
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
        throw new Error(
          (body as { error?: string }).error ??
            "Failed to list Klaviyo accounts",
        );
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

  async function handleBingAdsReturn(code: string, state: string) {
    error = "";
    loading = "bing_ads";
    try {
      const host = get(runtime).host;
      const res = await fetch(
        `${host}/bratrax/onboard/bing-ads/accounts?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
        { credentials: "include" },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(
          (body as { error?: string }).error ??
            "Failed to list Bing Ads accounts",
        );
      }
      const body = (await res.json()) as {
        state: string;
        accounts: Array<{ id: string; name: string }>;
      };
      bingAdsState = body.state;
      accountModalAccounts = body.accounts.map((a) => ({
        id: a.id,
        name: a.name || a.id,
      }));
      accountModalPlatform = "Microsoft Bing Ads";
      showAccountModal = true;

      // Strip the callback params so a refresh doesn't re-trigger.
      const url = new URL(window.location.href);
      url.searchParams.delete("code");
      url.searchParams.delete("state");
      window.history.replaceState({}, "", url.toString());
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = "";
    }
  }

  // Exit the "Completing connection…" spinner gracefully when an account-fetch
  // step fails (e.g. the provider returns no ad accounts). Without this the
  // bounce screen would spin forever. Mirrors the OAuth-error path above: clear
  // pending state, surface the message — inline on /onboard/stack, or bounced
  // back to /connectors via `connectors_error` when that's where we came from.
  function bounceOutOfOAuth(message: string) {
    isOAuthBounce = false;
    const url = new URL(window.location.href);
    url.searchParams.delete("code");
    url.searchParams.delete("state");
    window.history.replaceState({}, "", url.toString());
    sessionStorage.removeItem("pinterest_ads_oauth_state");
    sessionStorage.removeItem("bing_ads_oauth_state");
    sessionStorage.removeItem("klaviyo_oauth_state");
    sessionStorage.removeItem("tiktok_ads_oauth_state");
    sessionStorage.removeItem("onboard_oauth_platform");
    const where = sessionStorage.getItem("onboard_oauth_return");
    sessionStorage.removeItem("onboard_oauth_return");
    if (where && where !== "/onboard/stack") {
      sessionStorage.setItem("connectors_error", message);
      goto(where);
      return;
    }
    error = message;
  }

  async function handlePinterestReturn(code: string, state: string) {
    error = "";
    loading = "pinterest_ads";
    try {
      const host = get(runtime).host;
      const res = await fetch(
        `${host}/bratrax/onboard/pinterest/accounts?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
        { credentials: "include" },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(
          (body as { error?: string }).error ??
            "Failed to list Pinterest accounts",
        );
      }
      const body = (await res.json()) as {
        state: string;
        accounts: Array<{ id: string; name: string }>;
      };
      pinterestAdsState = body.state;
      accountModalPlatform = "Pinterest";

      // Strip the callback params so a refresh doesn't re-trigger.
      const url = new URL(window.location.href);
      url.searchParams.delete("code");
      url.searchParams.delete("state");
      window.history.replaceState({}, "", url.toString());

      const pinAccounts = body.accounts || [];
      if (pinAccounts.length === 0) {
        // No ad accounts on this login — persist the token anyway (empty list)
        // so the card shows connected. handleAccountSelection does the /connect
        // POST + refresh + bounce-back; skip the account picker entirely.
        await handleAccountSelection([]);
        return;
      }

      accountModalAccounts = pinAccounts.map((a) => ({
        id: a.id,
        name: a.name || a.id,
      }));
      showAccountModal = true;
    } catch (e) {
      // No ad accounts / token failure / etc. — surface it and leave the
      // spinner instead of hanging on "Completing connection…".
      bounceOutOfOAuth(e instanceof Error ? e.message : String(e));
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

        // Run the multi-step account discovery in an async IIFE so we can
        // await sequential Graph calls instead of nesting callbacks. The
        // FB JS SDK doesn't return Promises natively; `fbApi` is the
        // Promise-wrapped shim defined below.
        (async () => {
          try {
            const userInfo = await fbApi<{ email?: string }>("/me", {
              fields: "name,email",
            });
            fbEmail = userInfo?.email ?? "";
          } catch (e) {
            console.warn("FB /me lookup failed", e);
          }

          const accounts = await listAllFbAdAccounts(token);
          loading = "";

          if (accounts.length === 0) {
            error = "No Facebook ad accounts found for this user.";
            return;
          }
          accountModalPlatform = "Facebook Ads";
          accountModalAccounts = accounts;
          showAccountModal = true;
        })();
      },
      // business_management is required to enumerate /me/businesses and
      // each BM's owned + client ad accounts — without it the user only
      // sees accounts they have direct personal permission on (the historical
      // single-BM bug).
      { scope: "email,ads_read,ads_management,business_management" },
    );
  }

  // Promise-wrap the callback-based FB.api so the discovery flow can use
  // async/await. Resolves with whatever the SDK passes the callback —
  // including FB error objects (caller inspects resp.error).
  function fbApi<T = unknown>(
    path: string,
    params: Record<string, unknown>,
  ): Promise<T> {
    return new Promise((resolve) => {
      (window as any).FB?.api(path, params, (resp: T) => resolve(resp));
    });
  }

  type FbPagedResponse<T> = {
    data?: T[];
    paging?: { next?: string };
    error?: { message?: string; code?: number };
  };

  // Walk paging.next cursors until exhausted. FB returns next as an absolute
  // Graph URL; FB.api accepts that as the path directly. Capped to avoid
  // accidental infinite loops.
  async function fbPagedList<T>(
    path: string,
    params: Record<string, unknown>,
  ): Promise<T[]> {
    const out: T[] = [];
    let nextPath: string | undefined = path;
    let nextParams: Record<string, unknown> | undefined = params;
    for (let i = 0; i < 50 && nextPath; i++) {
      const resp: FbPagedResponse<T> = await fbApi(nextPath, nextParams ?? {});
      if (resp?.error) {
        throw resp.error;
      }
      if (Array.isArray(resp?.data)) out.push(...resp.data);
      nextPath = resp?.paging?.next;
      nextParams = undefined;
    }
    return out;
  }

  // Build the full list of ad accounts the user can reach: direct-permission
  // accounts + every account owned-by or shared-with each Business Manager
  // they have access to. Each entry is tagged with the BM name (or
  // "Personal" for direct-permission-only accounts) so the modal can
  // section the list by Business Manager. Deduped by account_id;
  // BMs are walked BEFORE /me/adaccounts so accounts present in both
  // surface under their BM section rather than the Personal fallback.
  // Per-leg failures are logged and skipped rather than aborted — better
  // to surface a partial list than block the entire onboarding flow.
  async function listAllFbAdAccounts(
    token: string,
  ): Promise<Array<{ id: string; name: string; group: string }>> {
    type Row = { account_id: string; name?: string };
    const seen = new Map<string, { name: string; group: string }>();
    const add = (rows: Row[] | undefined, group: string) => {
      for (const a of rows ?? []) {
        if (a?.account_id && !seen.has(a.account_id)) {
          seen.set(a.account_id, {
            name: a.name || a.account_id,
            group,
          });
        }
      }
    };

    // 1. BMs first — needs business_management scope. If declined or the
    // FB app isn't approved for it, this returns empty/error and we fall
    // back to direct-permission accounts only.
    let businesses: Array<{ id: string; name?: string }> = [];
    try {
      businesses = await fbPagedList<{ id: string; name?: string }>(
        "/me/businesses",
        { access_token: token, fields: "id,name", limit: 100 },
      );
    } catch (e) {
      console.warn(
        "FB /me/businesses failed — falling back to direct-permission accounts only",
        e,
      );
    }

    await Promise.all(
      businesses.map(async (biz) => {
        const groupLabel = biz.name || `Business ${biz.id}`;
        await Promise.all([
          fbPagedList<Row>(`/${biz.id}/owned_ad_accounts`, {
            access_token: token,
            fields: "account_id,name",
            limit: 100,
          })
            .then((rows) => add(rows, groupLabel))
            .catch((e) =>
              console.warn(`FB /${biz.id}/owned_ad_accounts failed`, e),
            ),
          fbPagedList<Row>(`/${biz.id}/client_ad_accounts`, {
            access_token: token,
            fields: "account_id,name",
            limit: 100,
          })
            .then((rows) => add(rows, groupLabel))
            .catch((e) =>
              console.warn(`FB /${biz.id}/client_ad_accounts failed`, e),
            ),
        ]);
      }),
    );

    // 2. Direct-permission accounts fill in anything not already attributed
    // to a BM (personal ad accounts with no BM, or accounts the user is
    // added to as a user but whose BM the user can't enumerate).
    try {
      add(
        await fbPagedList<Row>("/me/adaccounts", {
          access_token: token,
          fields: "account_id,name",
          limit: 100,
        }),
        "Personal",
      );
    } catch (e) {
      console.warn("FB /me/adaccounts failed", e);
    }

    return [...seen.entries()]
      .map(([id, v]) => ({ id, name: v.name, group: v.group }))
      .sort((a, b) => {
        if (a.group !== b.group) return a.group.localeCompare(b.group);
        return a.name.localeCompare(b.name);
      });
  }

  interface GoogleAdAccountNode {
    customer_id: string;
    name?: string;
    descriptive_name?: string;
    manager_account_id?: string;
    children?: GoogleAdAccountNode[];
  }

  function flattenAccounts(
    accounts: GoogleAdAccountNode[],
  ): Array<{ id: string; name: string; group: string }> {
    const flat: Array<{ id: string; name: string; group: string }> = [];
    googleManagerMap = {};
    // A manager node (has children) becomes its own group header — children
    // inherit its name. A leaf inherits whatever parent we walked in with.
    // Standalone top-level accounts (no parent, no children) get "" and
    // render in the modal's ungrouped fallback section.
    function walk(list: GoogleAdAccountNode[], parentName: string) {
      for (const acc of list) {
        const id = String(acc.customer_id);
        const name = acc.name || acc.descriptive_name || id;
        const isManager = !!(acc.children && acc.children.length > 0);
        const group = isManager ? name : parentName;
        flat.push({ id, name, group });
        if (acc.manager_account_id) {
          googleManagerMap[id] = String(acc.manager_account_id);
        }
        if (acc.children) walk(acc.children, name);
      }
    }
    walk(accounts, "");
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
  async function runGoogleTrackingTemplateFlow(
    customerIds: string[],
  ): Promise<void> {
    if (!customerIds.length) return;
    const host = get(runtime).host;

    const previewRes = await fetch(
      `${host}/bratrax/onboard/google/preview-tracking-templates`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          code: googleAuthCode,
          customer_ids: customerIds,
        }),
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

  // ---------------------------------------------------------------------------
  // Facebook url_tags flow
  // ---------------------------------------------------------------------------
  // Per-campaign metadata; FB does NOT cascade url_tags from MCC/business
  // managers, and new campaigns post-onboarding start empty (the daily sweep
  // DAG is the long-term backstop). Run after /onboard/facebook/connect to
  // set the canonical Bratrax template on every existing ACTIVE/PAUSED
  // campaign on the selected ad accounts.
  async function runFacebookUrlTagsFlow(adAccountIds: string[]): Promise<void> {
    if (!adAccountIds.length || !clientId) return;
    const host = get(runtime).host;

    const previewRes = await fetch(
      `${host}/bratrax/onboard/facebook/preview-url-tags`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          client_id: clientId,
          ad_account_ids: adAccountIds,
        }),
      },
    );
    if (!previewRes.ok) {
      console.warn("FB url_tags preview failed", previewRes.status);
      return;
    }

    interface FbPreviewItem {
      act_id: string;
      account_name: string;
      total_campaigns: number;
      with_url_tags: number;
      conflicting: Array<{
        campaign_id: string;
        name: string;
        current: string;
      }>;
      conflicting_count: number;
      needs_confirm: boolean;
      error?: string;
    }
    const previewBody = (await previewRes.json()) as {
      proposed?: string;
      data?: FbPreviewItem[];
    };
    const items: FbPreviewItem[] = (previewBody.data || []).filter(
      (i) => !i.error,
    );

    const anyConflicts = items.some((i) => i.needs_confirm);
    if (!anyConflicts) {
      // Clean state — silently apply (sets where empty, skips where matches).
      await applyFacebookUrlTags(adAccountIds, false);
      return;
    }

    fbUrlTagsProposed = previewBody.proposed || "";
    fbUrlTagsAccounts = items.map((i) => ({
      act_id: i.act_id,
      account_name: i.account_name,
      total_campaigns: i.total_campaigns,
      with_url_tags: i.with_url_tags,
      conflicting: i.conflicting,
      conflicting_count: i.conflicting_count,
    }));
    fbUrlTagsAdAccountIds = adAccountIds;
    showFbUrlTagsModal = true;

    const choice: FbUrlTagsChoice = await new Promise((resolve) => {
      fbUrlTagsResolve = resolve;
    });
    fbUrlTagsResolve = null;

    if (choice === "cancel") {
      showFbUrlTagsModal = false;
      throw new Error("Facebook url_tags setup cancelled");
    }

    fbUrlTagsLoading = true;
    try {
      // "Replace all" → force_replace=true; "Apply only to empty" →
      // force_replace=false (existing url_tags on conflicting campaigns are
      // left untouched server-side).
      await applyFacebookUrlTags(fbUrlTagsAdAccountIds, choice === "replace");
    } finally {
      fbUrlTagsLoading = false;
      showFbUrlTagsModal = false;
    }
  }

  async function applyFacebookUrlTags(
    adAccountIds: string[],
    forceReplace: boolean,
  ): Promise<void> {
    if (!adAccountIds.length || !clientId) return;
    const host = get(runtime).host;
    const res = await fetch(`${host}/bratrax/onboard/facebook/apply-url-tags`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        client_id: clientId,
        ad_account_ids: adAccountIds,
        force_replace: forceReplace,
      }),
    });
    if (res.status === 409) {
      const body = await res.json().catch(() => ({}));
      console.warn("FB url_tags apply returned 409", body);
      throw new Error("Facebook url_tags conflict");
    }
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(
        ((body as Record<string, unknown>).error as string) ||
          "Failed to apply Facebook url_tags",
      );
    }
  }

  function handleFbUrlTagsChoice(choice: FbUrlTagsChoice) {
    if (fbUrlTagsResolve) fbUrlTagsResolve(choice);
  }

  async function handleAccountSelection(selected: AdAccountInfo[]) {
    accountModalLoading = true;
    try {
      const host = get(runtime).host;
      const isFacebook = accountModalPlatform === "Facebook Ads";
      const isTikTok = accountModalPlatform === "TikTok Ads";
      const isKlaviyo = accountModalPlatform === "Klaviyo";
      const isBingAds = accountModalPlatform === "Microsoft Bing Ads";
      const isTaboola = accountModalPlatform === "Taboola";
      const isOutbrain = accountModalPlatform === "Outbrain";
      const isPinterest = accountModalPlatform === "Pinterest";

      let endpoint: string;
      let body: Record<string, unknown>;
      let connectedKey: string;

      if (isTaboola) {
        endpoint = `${host}/bratrax/onboard/taboola/connect`;
        body = {
          state: taboolaState,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "taboola";
      } else if (isOutbrain) {
        endpoint = `${host}/bratrax/onboard/outbrain/connect`;
        body = {
          state: outbrainState,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "outbrain";
      } else if (isBingAds) {
        endpoint = `${host}/bratrax/onboard/bing-ads/connect`;
        body = {
          state: bingAdsState,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "bing_ads";
      } else if (isPinterest) {
        endpoint = `${host}/bratrax/onboard/pinterest/connect`;
        body = {
          state: pinterestAdsState,
          client_id: clientId,
          selectedAccounts: selected,
        };
        connectedKey = "pinterest_ads";
      } else if (isKlaviyo) {
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
      // existing template would be overwritten). Gated behind
      // ENABLE_GOOGLE_TRACKING_TEMPLATES — flip to true to activate.
      if (ENABLE_GOOGLE_TRACKING_TEMPLATES && connectedKey === "google_ads") {
        try {
          const customerIds = selected.map((a) => a.accountId);
          await runGoogleTrackingTemplateFlow(customerIds);
        } catch (e) {
          console.warn("Tracking template flow:", e);
        }
      }

      // Facebook Ads: set url_tags on every ACTIVE/PAUSED campaign across the
      // selected accounts. Gated behind ENABLE_FB_URL_TAGS.
      if (ENABLE_FB_URL_TAGS && connectedKey === "facebook_ads") {
        try {
          const adAccountIds = selected.map((a) => a.accountId);
          await runFacebookUrlTagsFlow(adAccountIds);
        } catch (e) {
          console.warn("FB url_tags flow:", e);
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
      if (isBingAds) {
        sessionStorage.removeItem("bing_ads_oauth_state");
        bingAdsState = "";
      }
      if (isPinterest) {
        sessionStorage.removeItem("pinterest_ads_oauth_state");
        pinterestAdsState = "";
      }
      if (isTaboola) taboolaState = "";
      if (isOutbrain) outbrainState = "";

      // Re-pull from the server so the card renders from the source of
      // truth. Also guarantees a fresh Set reference for Svelte reactivity.
      await refreshFromServer();

      // Honor the return-to set by the originating page. When the OAuth was
      // initiated from /connectors (or any other page), bounce back there
      // after the modal completes. Default "/onboard/stack" is a no-op.
      const returnTo =
        sessionStorage.getItem("onboard_oauth_return") || "/onboard/stack";
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

  // ---------------------------------------------------------------------------
  // Credential-modal flow (Taboola / Outbrain): the modal collects
  // credentials, we POST them to /{platform}/accounts to get the listed
  // accounts + a state token, then open AccountSelectionModal — which
  // routes selection through handleAccountSelection above.
  // ---------------------------------------------------------------------------
  async function handleTaboolaCredentials(creds: {
    clientId: string;
    clientSecret: string;
  }) {
    taboolaModalError = "";
    taboolaModalLoading = true;
    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}/bratrax/onboard/taboola/accounts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          client_id: creds.clientId,
          client_secret: creds.clientSecret,
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (data as any).error || "Failed to list Taboola accounts",
        );
      }
      taboolaState = (data as any).state;
      accountModalAccounts = ((data as any).accounts || []).map((a: any) => ({
        id: String(a.id),
        name: a.name || a.id,
      }));
      accountModalPlatform = "Taboola";
      showTaboolaModal = false;
      showAccountModal = true;
    } catch (e) {
      taboolaModalError = e instanceof Error ? e.message : String(e);
    } finally {
      taboolaModalLoading = false;
    }
  }

  async function handleOutbrainCredentials(creds: { encodedAuth: string }) {
    outbrainModalError = "";
    outbrainModalLoading = true;
    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}/bratrax/onboard/outbrain/accounts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ auth: creds.encodedAuth }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (data as any).error || "Failed to list Outbrain marketers",
        );
      }
      outbrainState = (data as any).state;
      accountModalAccounts = ((data as any).accounts || []).map((a: any) => ({
        id: String(a.id),
        name: a.name || a.id,
      }));
      accountModalPlatform = "Outbrain";
      showOutbrainModal = false;
      showAccountModal = true;
    } catch (e) {
      outbrainModalError = e instanceof Error ? e.message : String(e);
    } finally {
      outbrainModalLoading = false;
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
    } else if (platform.type === "credential_modal") {
      if (isConnected(platform.id)) return;
      error = "";
      if (platform.id === "taboola") {
        taboolaModalError = "";
        showTaboolaModal = true;
      } else if (platform.id === "outbrain") {
        outbrainModalError = "";
        showOutbrainModal = true;
      }
    } else if (platform.type === "coming_soon") {
      // Do nothing
    } else if (platform.id === "external_pages") {
      // Selecting the radio alone is opaque — open the builder picker so
      // the user picks which funnel platform (Funnelish, ClickFunnels, …)
      // hosts their pages. The picker then opens the matching install
      // modal; "I've installed it" persists the builder-specific platform
      // (e.g. "funnelish") via /onboard/connect, which flows back through
      // refreshFromServer → connectedPlatforms.
      handleSelectionToggle(platform);
      if (selectedPlatforms.has("external_pages")) {
        error = "";
        showBuilderPickerModal = true;
      }
    } else {
      handleSelectionToggle(platform);
    }
  }

  function handleBuilderPicked(e: CustomEvent<{ builder: string }>) {
    showBuilderPickerModal = false;
    if (e.detail.builder === "funnelish") {
      showFunnelishModal = true;
    }
  }

  async function handleFunnelishInstalled() {
    showFunnelishModal = false;
    await refreshFromServer();
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
      sessionStorage.setItem(
        "onboard_stack_selections",
        JSON.stringify(stackSelections),
      );

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
      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
      >
        ONBOARDING
      </div>
      <div
        class="font-mono text-xs uppercase tracking-wider text-bratrax-text-muted"
      >
        Completing connection…
      </div>
    </div>
  </div>
{:else}
  <div
    class="flex h-full w-full items-start justify-center overflow-y-auto bg-bratrax-bg py-12"
  >
    <div
      class="relative w-full max-w-2xl border border-bratrax-border bg-bratrax-surface p-8"
    >
      <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

      <div class="mb-6">
        <div
          class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
        >
          ONBOARDING
        </div>
        <h1 class="text-2xl font-black text-bratrax-text-headline">
          Build your stack
        </h1>
        <p class="mt-2 text-sm font-light text-bratrax-text-body">
          Connect the tools you use. We'll set up your analytics automatically.
        </p>
      </div>

      {#if error}
        <div
          class="mb-4 whitespace-pre-wrap break-words border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
        >
          {error}
        </div>
      {/if}

      {#if clientId && isConnected("woocommerce")}
        <div class="mb-6">
          <WooTrackingInstall {clientId} />
        </div>
      {/if}

      <div class="flex flex-col gap-6">
        {#each categories as category}
          <div>
            <div class="mb-2 flex items-baseline gap-2">
              <span
                class="font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70"
              >
                {category.title}
              </span>
              {#if category.subtitle}
                <span class="font-mono text-[10px] text-bratrax-text-muted"
                  >({category.subtitle})</span
                >
              {/if}
            </div>
            <div class="flex flex-wrap gap-2">
              {#each category.platforms as platform}
                <button
                  on:click={() => handleCardClick(platform)}
                  disabled={loading === platform.id ||
                    platform.id === "shopify" ||
                    platform.id === "woocommerce" ||
                    platform.type === "coming_soon"}
                  class="flex items-center gap-2 border px-4 py-2.5 font-mono text-xs font-bold uppercase tracking-wider transition-all
                  {radioToggleIds.has(platform.id)
                    ? isSelected(platform.id)
                      ? 'border-bratrax-acid/50 bg-bratrax-acid/10 text-bratrax-acid'
                      : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover'
                    : isConnected(platform.id)
                      ? 'border-bratrax-acid/50 bg-bratrax-acid/10 text-bratrax-acid'
                      : isSelected(platform.id)
                        ? 'border-bratrax-cyan/50 bg-bratrax-cyan/10 text-bratrax-cyan'
                        : platform.type === 'coming_soon'
                          ? 'border-bratrax-border bg-bratrax-bg text-bratrax-text-muted cursor-not-allowed'
                          : 'border-bratrax-border bg-bratrax-bg text-bratrax-text-body hover:border-bratrax-text-muted hover:bg-bratrax-hover'}
                  {loading === platform.id ? 'opacity-50' : ''}
                  {platform.id === 'shopify' || platform.id === 'woocommerce'
                    ? 'cursor-default'
                    : ''}"
                >
                  <span
                    class="h-3 w-3"
                    style="background-color: {platform.color}"
                  ></span>
                  {#if radioToggleIds.has(platform.id) ? isSelected(platform.id) : isConnected(platform.id)}
                    <svg
                      class="h-4 w-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                  {/if}
                  {platform.name}
                  {#if platform.type === "coming_soon"}
                    <span
                      class="text-bratrax-text-muted normal-case tracking-normal"
                      >coming soon</span
                    >
                  {:else if loading === platform.id}
                    <span
                      class="text-bratrax-text-muted normal-case tracking-normal"
                      >connecting...</span
                    >
                  {/if}
                </button>
              {/each}
            </div>
          </div>
        {/each}
      </div>

      <div
        class="mt-8 flex items-center justify-between border-t border-bratrax-border pt-6"
      >
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
  onClose={() => {
    showAccountModal = false;
  }}
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

<FbUrlTagsConfirmModal
  open={showFbUrlTagsModal}
  loading={fbUrlTagsLoading}
  proposed={fbUrlTagsProposed}
  accounts={fbUrlTagsAccounts}
  onReplaceAll={() => handleFbUrlTagsChoice("replace")}
  onApplyEmptyOnly={() => handleFbUrlTagsChoice("empty_only")}
  onCancel={() => handleFbUrlTagsChoice("cancel")}
/>

{#if showBuilderPickerModal}
  <ExternalPagesBuilderPickerModal
    on:select={handleBuilderPicked}
    on:close={() => (showBuilderPickerModal = false)}
  />
{/if}

{#if showFunnelishModal}
  <FunnelishInstallModal
    {clientId}
    {shopifyShopDomain}
    connected={connectedPlatforms.has("funnelish")}
    connectedAt={connectedAt["funnelish"] || ""}
    on:installed={handleFunnelishInstalled}
    on:close={() => (showFunnelishModal = false)}
  />
{/if}

<TaboolaCredentialModal
  open={showTaboolaModal}
  loading={taboolaModalLoading}
  error={taboolaModalError}
  onSubmit={handleTaboolaCredentials}
  onClose={() => {
    showTaboolaModal = false;
    taboolaModalError = "";
  }}
/>

<OutbrainLoginModal
  open={showOutbrainModal}
  loading={outbrainModalLoading}
  error={outbrainModalError}
  onSubmit={handleOutbrainCredentials}
  onClose={() => {
    showOutbrainModal = false;
    outbrainModalError = "";
  }}
/>
