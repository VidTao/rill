<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { goto } from "$app/navigation";
  import { get } from "svelte/store";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import {
    onboardMe,
    onboardDisconnect,
    connectedFromStackSelections,
    needsReconnectFromStackSelections,
    getOAuthConfig,
    verifyEmbedStatus,
  } from "$lib/bratrax/onboarding/api";
  import type { AdAccountInfo } from "$lib/bratrax/connectors/api";
  import AccountSelectionModal from "./AccountSelectionModal.svelte";
  import ExternalPagesBuilderPickerModal from "./ExternalPagesBuilderPickerModal.svelte";
  import FunnelishInstallModal from "./FunnelishInstallModal.svelte";
  import TaboolaCredentialModal from "./TaboolaCredentialModal.svelte";
  import OutbrainLoginModal from "./OutbrainLoginModal.svelte";
  import BloomreachCredentialModal from "./BloomreachCredentialModal.svelte";
  import AmazonRegionModal from "$lib/bratrax/onboarding/AmazonRegionModal.svelte";
  import TrackingTemplateGuide from "$lib/bratrax/TrackingTemplateGuide.svelte";
  import { getAuthConfig } from "$lib/bratrax/auth";
  import ConnectorPill from "$lib/bratrax/connectors/ConnectorPill.svelte";
  import { trackConnectedSources } from "$lib/bratrax/analytics";
  import WooTrackingInstall from "$lib/bratrax/onboarding/WooTrackingInstall.svelte";
  import {
    fetchSyncStatus,
    aggregateBySource,
    anyBackfilling,
    backfillLabel,
    relativeTime,
    type SourceSyncStatus,
  } from "$lib/bratrax/syncStatus";

  // Public OAuth client IDs — fetched from /onboard/oauth-config on mount
  // (BUG-01: not hardcoded in source). Empty until the fetch completes;
  // OAuth-init click handlers guard against empty values.
  let fbAppId = "";
  let googleClientId = "";

  // ---------------------------------------------------------------------------
  // Platform registry
  //
  // Every entry here is migrated to the new rill_onboarding_state surface
  // (stack_selections.{id}_credentials + connected_platforms). Nothing in this
  // registry still touches the legacy bratrax_advertising_connections /
  // bratrax_user_platform_credentials path.
  // ---------------------------------------------------------------------------
  interface Platform {
    id: string; // matches connected_platforms[].platform + stack_selections key prefix
    name: string;
    // "shopify" needs a shop-domain input page first; "snippet_install" is a
    // copy-paste pixel (no OAuth) — flagged installed when the user clicks
    // "I've installed it" in the snippet modal. "credential_modal" opens a
    // username/password (or client_id/secret) form whose submit triggers a
    // direct credential→token exchange on the backend (Taboola, Outbrain).
    // "oauth_region" is "oauth" plus a region picker shown first, because
    // Amazon SP-API's consent host differs per region and can't be discovered.
    type:
      | "oauth"
      | "oauth_region"
      | "client_sdk"
      | "shopify"
      | "snippet_install"
      | "credential_modal";
    authUrlPath?: string; // for redirect OAuth (TikTok / Klaviyo)
    color: string;
  }

  const platforms: Platform[] = [
    { id: "shopify", name: "Shopify", type: "shopify", color: "#95BF47" },
    // WooCommerce is gated behind the ALLOW_WOOCOMMERCE env flag. It is
    // filtered out of `visiblePlatforms` below when the flag is off.
    {
      id: "woocommerce",
      name: "WooCommerce",
      type: "credential_modal",
      color: "#7F54B3",
    },
    {
      id: "google_ads",
      name: "Google Ads",
      type: "client_sdk",
      color: "#4285F4",
    },
    {
      id: "facebook_ads",
      name: "Facebook Ads",
      type: "client_sdk",
      color: "#1877F2",
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
      id: "pinterest_ads",
      name: "Pinterest",
      type: "oauth",
      authUrlPath: "/bratrax/onboard/pinterest/auth-url",
      color: "#E60023",
    },
    {
      id: "klaviyo",
      name: "Klaviyo",
      type: "oauth",
      authUrlPath: "/bratrax/onboard/klaviyo/auth-url",
      color: "#2D2D2D",
    },
    {
      id: "bloomreach",
      name: "Bloomreach",
      type: "credential_modal",
      color: "#502EA8",
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
      id: "amazon_ads",
      name: "Amazon Ads",
      type: "oauth",
      authUrlPath: "/bratrax/onboard/amazon-ads/auth-url",
      color: "#FF9900",
    },
    {
      id: "amazon_sp",
      name: "Amazon Seller Central",
      type: "oauth_region",
      authUrlPath: "/bratrax/onboard/amazon-sp/auth-url",
      color: "#FF9900",
    },
    {
      id: "external_pages",
      name: "External Landing Pages",
      type: "snippet_install",
      color: "#F59E0B",
    },

    // --- Re-enable as each platform is migrated to rill_onboarding_state ---
    // (Amazon Ads + Amazon Seller Central migrated 2026-08-04.)
  ];

  // WooCommerce is gated behind the ALLOW_WOOCOMMERCE env flag (Go proxy,
  // surfaced via /bratrax/auth/config). Off → filtered out of the grid.
  // Defaults false until getAuthConfig() resolves in onMount.
  let allowWoocommerce = false;

  // ---------------------------------------------------------------------------
  // Store gating
  //
  // A client picks exactly ONE store during onboarding (/onboard/store), and
  // that choice is what _determine_stack() compiles into config.yaml — there is
  // no dual-store stack. So we hide the store they didn't pick, rather than
  // offering a Connect button that would contradict their compiled stack.
  //
  // Derived from the same inputs _determine_stack() reads (onboarding.py:1417).
  // Explicitly NOT from me.template_name, which is hardcoded server-side to
  // "shopify-paid-media" (onboarding.py:1951, :2057) and lies for Woo clients.
  // ---------------------------------------------------------------------------

  // True once /onboard/me has answered. Until then BOTH store rows are hidden
  // (fail closed) — rendering a Connect button we're about to remove is worse
  // than a brief gap: it is live and navigates straight to /onboard/store.
  let storeResolved = false;

  // Precedence: the LOCK wins over the connected set. The lock is an explicit
  // admin decision stamped once at /onboard/start (onboarding.py:1174) and
  // never rewritten or cleared. The connected set is derived and CAN be
  // polluted: the store-lock guard lives only in _record_platform_connection
  // (onboarding.py:2342), while the credential write that precedes it
  // (onboarding.py:2171 shopify / :4666 woo) is unguarded — so a refused store
  // still lands in stack_selections.{p}_credentials, and
  // connectedFromStackSelections() turns that key alone into "connected".
  // Connected-first would hide the locked store the client is licensed for.
  $: lockedStore =
    stackSelections?.locked_store_platform === "shopify" ||
    stackSelections?.locked_store_platform === "woocommerce"
      ? (stackSelections.locked_store_platform as "shopify" | "woocommerce")
      : "";

  $: storeChoice =
    lockedStore ||
    (connectedPlatforms.has("shopify")
      ? "shopify"
      : connectedPlatforms.has("woocommerce")
        ? "woocommerce"
        : "");

  // Never suppress Shopify in favour of a WooCommerce row we can't render.
  // getAuthConfig() fails CLOSED to allow_woocommerce=false (auth.ts:127,130),
  // so without the allowWoocommerce term one flaky /bratrax/auth/config would
  // leave a Woo client with no store row at all.
  $: suppressedStore =
    storeChoice === "shopify"
      ? "woocommerce"
      : storeChoice === "woocommerce" && allowWoocommerce
        ? "shopify"
        : "";

  $: visiblePlatforms = platforms.filter((p) => {
    if (p.id !== "shopify" && p.id !== "woocommerce") return true;
    // ALLOW_WOOCOMMERCE deployment gate (unchanged).
    if (p.id === "woocommerce" && !allowWoocommerce) return false;
    if (!storeResolved) return false;
    // A connected store is ALWAYS shown — its Disconnect button is the only
    // cleanup UI for a stray connection, and hiding it would orphan the
    // Shopify embed banner + WooTrackingInstall panel that sit above the grid.
    if (connectedPlatforms.has(p.id)) return true;
    return p.id !== suppressedStore;
  });

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------
  let connectedPlatforms: Set<string> = new Set();
  // Connectors whose stored token has gone bad (an extract script hit a
  // platform auth error and set needs_reconnect on their credentials). Populated
  // generically from stack_selections, so a newly-flagging platform needs no
  // change here. Cleared server-side on a successful reconnect.
  let needsReconnect: Set<string> = new Set();
  let stackSelections: Record<string, any> = {};
  let connectedAt: Record<string, string> = {};
  let clientId = "";
  let loading = "";
  let error = "";

  // Per-source backfill/sync status, keyed by source (=== platform.id), from
  // /bratrax/sync-status. Drives the pill fill, the "Last sync" label, the
  // reassurance copy, and the per-account detail in the View Accounts pop-up.
  // Polled every 10s while this page is open (see SYNC_POLL_MS) so the pill
  // fills live without a reload; the timer is cleared on destroy.
  let syncBySource: Map<string, SourceSyncStatus> = new Map();
  $: showBackfillReassurance = anyBackfilling(syncBySource);
  const SYNC_POLL_MS = 10_000;
  let syncPollTimer: ReturnType<typeof setInterval> | undefined;

  // "View accounts" modal — generic shape for ad platforms (accounts list)
  // plus a Shopify-specific shape (single store with shop + name + currency).
  let showInfoModal = false;
  let infoModalPlatform = "";
  let infoModalAccounts: Array<{
    id: string;
    name: string;
    progressPct?: number;
    lastSyncAt?: string | null;
  }> = [];
  let infoModalConnectedAt = "";
  let infoModalShopify: {
    shop: string;
    shopName: string;
    currency: string;
  } | null = null;
  // Single-store (Shopify / Woo) backfill card data, null when no sync row.
  let infoModalBackfill: {
    progressPct: number;
    lastSyncAt: string | null;
  } | null = null;

  // Disconnect confirmation modal — replaces the browser-native confirm() so
  // the prompt matches Bratrax styling.
  let showConfirmModal = false;
  let confirmPlatform: Platform | null = null;

  // OAuth completion modal (used by Google + Facebook popup flows; TikTok &
  // Klaviyo use redirect → land on /onboard/stack → bounce back here).
  let showAccountModal = false;
  let accountModalPlatform = "";
  let accountModalAccounts: Array<{
    id: string;
    name: string;
    group?: string;
  }> = [];
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

  // External-pages install flow: clicking the "External Landing Pages"
  // card opens a builder picker (Funnelish active, ClickFunnels / GHL /
  // Custom coming soon); picking a builder opens the matching install
  // modal. Each builder is recorded as its own platform in
  // connected_platforms (e.g. "funnelish"); the umbrella card is "connected"
  // when any underlying builder is connected.
  let showBuilderPickerModal = false;
  // Amazon SP-API needs its region chosen before the redirect.
  let showAmazonRegionModal = false;
  let showFunnelishModal = false;

  // Credential-modal state for Taboola / Outbrain. Each modal owns its
  // own visibility + error/loading flags so the parent can keep the modal
  // open on credential-exchange failure (so the user can retry without
  // re-typing). On success we close the credential modal and stash the
  // state token returned by the backend, then open AccountSelectionModal.
  let showTaboolaModal = false;
  let taboolaModalLoading = false;
  let taboolaModalError = "";
  let taboolaState = "";

  let showOutbrainModal = false;
  let outbrainModalLoading = false;
  let outbrainModalError = "";
  let outbrainState = "";

  let showBloomreachModal = false;
  let bloomreachModalLoading = false;
  let bloomreachModalError = "";
  let bloomreachState = "";

  // Builder ids that map to the "external_pages" umbrella. Keep in sync with
  // ExternalPagesBuilderPickerModal's `available` list.
  const externalBuilderIds = new Set(["funnelish"]);
  $: hasExternalBuilder = [...connectedPlatforms].some((p) =>
    externalBuilderIds.has(p),
  );
  $: connectedBuilderName = hasExternalBuilder
    ? [...connectedPlatforms].find((p) => externalBuilderIds.has(p))
    : "";

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
    // No client → nothing to suppress. Open the store gate so a client-less
    // user still sees both store rows instead of a grid with no store at all.
    if (!me?.client_id) {
      storeResolved = true;
      return;
    }
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
    // Connectors whose token died — an extract script set needs_reconnect on
    // their credentials. Drives the third row state below.
    needsReconnect = new Set(
      needsReconnectFromStackSelections(stackSelections),
    );

    // GA4 `data_source_connected` — post-onboarding connects land here rather
    // than on /onboard/stack. See trackConnectedSources(); a client with no
    // recorded baseline stays silent, so this never retro-fires for the sources
    // an established workspace connected months ago.
    trackConnectedSources(clientId, [...next]);

    // Open the store gate HERE, not at the end of this function: everything
    // below is informational and awaits two more round trips (verifyEmbedStatus
    // hits the Shopify Asset API and can take seconds). The store rows only
    // need connectedPlatforms + stackSelections, which are now set.
    storeResolved = true;

    // Backfill/sync status — informational, non-fatal. Also polled on an
    // interval (see startSyncPolling) so the pill fills live without a reload.
    await refreshSyncStatus();

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

  // Re-fetch ONLY the backfill/sync status (not the heavier onboardMe +
  // embed-status round trips — connections don't change without a user action,
  // which already calls refreshFromServer). Reassigning syncBySource drives the
  // pill fill, "Last sync", and reassurance copy reactively. In-flight guard so
  // a slow request can't stack up behind the 10s interval.
  let syncPolling = false;
  async function refreshSyncStatus() {
    if (syncPolling) return;
    syncPolling = true;
    try {
      syncBySource = aggregateBySource(await fetchSyncStatus());
    } finally {
      syncPolling = false;
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

  // "external_pages" is a UI umbrella — there's no platform of that name in
  // connected_platforms. Treat it as connected when any underlying builder
  // (Funnelish, future ClickFunnels / GoHighLevel) is connected.
  $: isConnected = (id: string): boolean => {
    if (id === "external_pages") return hasExternalBuilder;
    return connectedPlatforms.has(id);
  };

  // Third row state: connected, but the token is dead. Requires isConnected so a
  // stale flag on a since-disconnected platform can't render a "Needs
  // reconnect" row for something that shows as not connected — the flag lives
  // inside {platform}_credentials, and disconnect deletes that whole key, so
  // this is belt-and-braces rather than a live case.
  $: needsReconnectFor = (id: string): boolean =>
    isConnected(id) && needsReconnect.has(id);

  // ---------------------------------------------------------------------------
  // OAuth init handlers (mirror /onboard/stack — only difference: return-to
  // is set to /connectors so the dispatcher in /onboard/stack bounces here
  // after Klaviyo / TikTok complete).
  // ---------------------------------------------------------------------------
  async function handleOAuthConnect(platform: Platform, extraQuery = "") {
    if (isConnected(platform.id)) return;
    if (!platform.authUrlPath) return;
    error = "";
    loading = platform.id;

    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}${platform.authUrlPath}${extraQuery}`, {
        credentials: "include",
      });
      if (!res.ok)
        throw new Error(`Failed to get auth URL for ${platform.name}`);
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
      // single-BM bug). Mirrors /onboard/stack so both flows surface the
      // same set of accounts.
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
  // to surface a partial list than block the entire flow.
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
    is_manager_account?: boolean;
    children?: GoogleAdAccountNode[];
  }

  function flattenAccounts(
    accounts: GoogleAdAccountNode[],
  ): Array<{ id: string; name: string }> {
    const flat: Array<{ id: string; name: string }> = [];
    googleManagerMap = {};
    function walk(list: GoogleAdAccountNode[]) {
      for (const acc of list) {
        const id = String(acc.customer_id);
        const isManager =
          !!acc.is_manager_account ||
          !!(acc.children && acc.children.length > 0);
        // Manager (MCC) accounts can't serve metrics — never selectable.
        if (!isManager) {
          flat.push({ id, name: acc.name || acc.descriptive_name || id });
        }
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
      const isTaboola = accountModalPlatform === "Taboola";
      const isOutbrain = accountModalPlatform === "Outbrain";
      const isBloomreach = accountModalPlatform === "Bloomreach";

      let endpoint: string;
      let body: Record<string, unknown>;

      if (isTaboola) {
        endpoint = `${host}/bratrax/onboard/taboola/connect`;
        body = {
          state: taboolaState,
          client_id: clientId,
          selectedAccounts: selected,
        };
      } else if (isOutbrain) {
        endpoint = `${host}/bratrax/onboard/outbrain/connect`;
        body = {
          state: outbrainState,
          client_id: clientId,
          selectedAccounts: selected,
        };
      } else if (isBloomreach) {
        endpoint = `${host}/bratrax/onboard/bloomreach/connect`;
        body = {
          state: bloomreachState,
          client_id: clientId,
          selectedAccounts: selected,
        };
      } else if (isFacebook) {
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

      if (isTaboola) taboolaState = "";
      if (isOutbrain) outbrainState = "";
      if (isBloomreach) bloomreachState = "";

      showAccountModal = false;
      await refreshFromServer();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      accountModalLoading = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Credential-modal flow (Taboola / Outbrain): the modal collects
  // credentials, we POST them to /{platform}/accounts to get a listed set of
  // accounts + a state token, then open AccountSelectionModal — which
  // routes selection through the existing handleAccountSelection above.
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

  async function handleBloomreachCredentials(creds: {
    projectToken: string;
    apiKey: string;
    apiSecret: string;
    baseUrl: string;
  }) {
    bloomreachModalError = "";
    bloomreachModalLoading = true;
    try {
      const host = get(runtime).host;
      const res = await fetch(`${host}/bratrax/onboard/bloomreach/accounts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          project_token: creds.projectToken,
          api_key: creds.apiKey,
          api_secret: creds.apiSecret,
          base_url: creds.baseUrl,
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (data as any).error || "Failed to validate Bloomreach credentials",
        );
      }
      bloomreachState = (data as any).state;
      accountModalAccounts = ((data as any).accounts || []).map((a: any) => ({
        id: String(a.id),
        name: a.name || a.id,
      }));
      accountModalPlatform = "Bloomreach";
      showBloomreachModal = false;
      showAccountModal = true;
    } catch (e) {
      bloomreachModalError = e instanceof Error ? e.message : String(e);
    } finally {
      bloomreachModalLoading = false;
    }
  }

  // WooCommerce: needs the store URL first, then redirects to the store's
  // wc-auth authorize screen (the Auth-Endpoint flow). /onboard/store collects
  // the URL and kicks off the redirect — same "store-domain page first" pattern
  // as Shopify. The store POSTs the key pair back to the public callback, then
  // the browser lands back here and refreshFromServer shows it connected.
  function handleWooCommerceConnect() {
    if (isConnected("woocommerce")) return;
    sessionStorage.setItem("onboard_oauth_return", "/connectors");
    goto("/onboard/store?provider=woocommerce");
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
      // The "external_pages" card is a UI umbrella — there's no platform of
      // that name in connected_platforms. Translate to the actual underlying
      // builder (Funnelish today; ClickFunnels / GHL later) so the backend
      // disconnect path can remove the right row.
      const target =
        platform.id === "external_pages" && connectedBuilderName
          ? connectedBuilderName
          : platform.id;
      await onboardDisconnect(clientId, target);
      await refreshFromServer();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = "";
    }
  }

  // ---------------------------------------------------------------------------
  // Reconnect (needs_reconnect state)
  // ---------------------------------------------------------------------------
  /**
   * Replace a dead connection with a fresh one: drop the stored credentials,
   * then run the platform's normal connect flow.
   *
   * Disconnecting first is REQUIRED, not just tidy. Every connect entry point
   * early-returns when the platform already reads as connected
   * (handleFacebookConnect, handleOAuthConnect, handleShopifyConnect,
   * handleWooCommerceConnect all open with `if (isConnected(...)) return;`), so
   * without this the click would be a silent no-op.
   *
   * It also keeps connected_platforms honest: _record_platform_connection is
   * append-only (`if platform not in existing`), so an in-place reconnect
   * refreshes the token in stack_selections but leaves the array entry's
   * account_id / account_name / connected_at frozen at their original values —
   * and /bratrax/sync-status reads its account list from that array. Removing
   * the entry means the reconnect re-appends it with the newly-selected
   * accounts.
   *
   * Trade-off: abandoning the provider popup leaves the row at "Not connected"
   * with a Connect button. That's the honest state — the old token was dead.
   *
   * Platform-agnostic: works for any connector that starts setting the flag.
   */
  async function handleReconnect(platform: Platform) {
    if (!clientId) return;
    error = "";
    loading = platform.id;
    try {
      const target =
        platform.id === "external_pages" && connectedBuilderName
          ? connectedBuilderName
          : platform.id;
      await onboardDisconnect(clientId, target);
      // Await the refresh so connectedPlatforms is up to date before
      // handleCardConnect evaluates its isConnected guard.
      await refreshFromServer();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      loading = "";
      return;
    }
    // Clear the spinner before handing off: the connect flow owns `loading`
    // from here, and for redirect-OAuth platforms it navigates away entirely.
    loading = "";
    handleCardConnect(platform);
  }

  // ---------------------------------------------------------------------------
  // View accounts modal
  // ---------------------------------------------------------------------------
  function handleView(platform: Platform) {
    const cred = stackSelections[`${platform.id}_credentials`] || {};
    infoModalConnectedAt = connectedAt[platform.id] || cred.connected_at || "";
    infoModalPlatform = platform.name;

    // Backfill rows for this source, keyed by account_id for per-card merge.
    const sync = syncBySource.get(platform.id);
    const syncByAccount = new Map(
      (sync?.accounts ?? []).map((a) => [a.account_id, a]),
    );
    infoModalBackfill = null;

    if (platform.id === "shopify") {
      // Single-store shape — no accounts list.
      infoModalShopify = {
        shop: cred.shop || "",
        shopName: cred.shop_name || "",
        currency: cred.currency || "",
      };
      infoModalAccounts = [];
      if (sync) {
        infoModalBackfill = {
          progressPct: sync.progressPct,
          lastSyncAt: sync.lastSyncAt,
        };
      }
    } else if (platform.id === "woocommerce") {
      // Single-store shape, WooCommerce field names.
      infoModalShopify = {
        shop: cred.store_url || "",
        shopName: cred.store_name || "",
        currency: cred.currency || "",
      };
      infoModalAccounts = [];
      if (sync) {
        infoModalBackfill = {
          progressPct: sync.progressPct,
          lastSyncAt: sync.lastSyncAt,
        };
      }
    } else {
      // Multi-account shape: TikTok stores them under `advertisers`; everyone else uses `accounts`.
      const list: any[] = cred.accounts || cred.advertisers || [];
      infoModalAccounts = list.map((a: any) => {
        const id = String(a.accountId || a.id || a.account_id || "");
        const row = syncByAccount.get(id);
        return {
          id,
          name:
            a.accountName || a.name || a.account_name || a.id || "(unnamed)",
          progressPct: row?.progress_pct,
          lastSyncAt: row?.last_sync_at,
        };
      });
      infoModalShopify = null;
    }
    showInfoModal = true;
  }

  function handleShopifyConnect() {
    if (isConnected("shopify")) return;
    // Shopify needs a shop-domain input first — delegate to the existing
    // /onboard/store page (which has the input form + redirect-to-Shopify
    // OAuth init). The callback at /onboard/shopify/callback honors
    // onboard_oauth_return and bounces back here.
    sessionStorage.setItem("onboard_oauth_return", "/connectors");
    goto("/onboard/store");
  }

  function handleCardConnect(platform: Platform) {
    if (platform.id === "woocommerce") {
      handleWooCommerceConnect();
      return;
    }
    if (platform.type === "client_sdk") {
      if (platform.id === "google_ads") handleGoogleAdsConnect();
      else if (platform.id === "facebook_ads") handleFacebookConnect();
    } else if (platform.type === "shopify") {
      handleShopifyConnect();
    } else if (platform.type === "snippet_install") {
      openBuilderPickerModal();
    } else if (platform.type === "oauth_region") {
      if (isConnected(platform.id)) return;
      error = "";
      showAmazonRegionModal = true;
    } else if (platform.type === "credential_modal") {
      error = "";
      if (platform.id === "taboola") {
        taboolaModalError = "";
        showTaboolaModal = true;
      } else if (platform.id === "outbrain") {
        outbrainModalError = "";
        showOutbrainModal = true;
      } else if (platform.id === "bloomreach") {
        bloomreachModalError = "";
        showBloomreachModal = true;
      }
    } else {
      handleOAuthConnect(platform);
    }
  }

  function handleAmazonRegionPicked(
    e: CustomEvent<{ region: "na" | "eu" | "fe" }>,
  ) {
    showAmazonRegionModal = false;
    const amazonSp = platforms.find((p) => p.id === "amazon_sp");
    if (!amazonSp) return;
    handleOAuthConnect(amazonSp, `?region=${e.detail.region}`);
  }

  function openBuilderPickerModal() {
    error = "";
    showBuilderPickerModal = true;
  }

  // "View snippet" on a connected external_pages row — open the install
  // modal for whichever builder is currently connected, so the merchant can
  // re-copy the right snippet or see their connected timestamp. Falls back
  // to the picker if (somehow) nothing is connected yet.
  function openConnectedBuilderModal() {
    error = "";
    if (connectedBuilderName === "funnelish") {
      showFunnelishModal = true;
    } else {
      showBuilderPickerModal = true;
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

  // ---------------------------------------------------------------------------
  // Init
  // ---------------------------------------------------------------------------
  onMount(async () => {
    // If the OAuth flow failed on the /onboard/stack side and bounced us
    // back here, /onboard/stack's onMount stashed the explanation under
    // `connectors_error`. Surface it once and clear so a refresh doesn't
    // re-show it.
    const carriedError = sessionStorage.getItem("connectors_error");
    if (carriedError) {
      error = carriedError;
      sessionStorage.removeItem("connectors_error");
    }

    // WooCommerce wc-auth returns here with ?success=…&user_id=<signed token>
    // appended. The key pair already arrived via the server-to-server callback,
    // so just surface a failure (if any) and scrub the params from the URL +
    // history so a refresh / back-button doesn't carry the token around.
    const wooReturn = new URLSearchParams(window.location.search);
    if (wooReturn.has("user_id") || wooReturn.has("success")) {
      if (wooReturn.get("success") === "0") {
        error = "WooCommerce connection was cancelled or didn't complete.";
      }
      window.history.replaceState(
        window.history.state,
        "",
        window.location.pathname,
      );
    }

    // ALLOW_WOOCOMMERCE gate (Go proxy env, via /bratrax/auth/config).
    allowWoocommerce = (await getAuthConfig()).allow_woocommerce;

    await refreshFromServer();
    // Poll backfill status every 10s so the pill fills live while the user
    // watches this page. Cleared in onDestroy below.
    syncPollTimer = setInterval(refreshSyncStatus, SYNC_POLL_MS);
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

  // Stop the 10s backfill poll when the page is torn down so it doesn't keep
  // hitting /bratrax/sync-status after the user navigates away.
  onDestroy(() => {
    if (syncPollTimer) clearInterval(syncPollTimer);
  });
</script>

<div class="connectors-canvas">
  <div class="connectors-inner">
    <div class="mb-7">
      <h1 class="connectors-title">Manage your connections</h1>
      <p class="connectors-subhead">
        Connect, view, or disconnect your data sources. Changes apply
        immediately.
      </p>
    </div>

    {#if showBackfillReassurance}
      <div
        class="mb-5 border-l-[3px] border-bratrax-acid bg-bratrax-surface px-4 py-3 text-sm text-bratrax-text-body"
      >
        <strong class="text-bratrax-text-headline"
          >Historical data is still loading.</strong
        >
        This can take a few days for larger accounts. If sync isn't complete within
        3 business days, reach out to
        <span class="underline">support@bratrax.com</span>.
      </div>
    {/if}

    {#if error}
      <div
        class="mb-4 whitespace-pre-wrap break-words border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato"
      >
        {error}
      </div>
    {/if}

    {#if showShopifyEmbedBanner}
      <div
        class="mb-4 border border-bratrax-acid/40 bg-bratrax-acid/10 px-4 py-3"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1">
            <p
              class="mb-1 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid"
            >
              Action required
            </p>
            <p class="text-sm text-bratrax-text-body">
              Enable the Bratrax tracker in your Shopify Theme Editor so we can
              capture email signups and on-site events. Takes about 30 seconds.
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

    {#if clientId && isConnected("woocommerce")}
      <div class="mb-5">
        <WooTrackingInstall {clientId} />
      </div>
    {/if}

    <div class="connector-list-card">
      {#each visiblePlatforms as platform}
        {@const connected = isConnected(platform.id)}
        {@const stale = needsReconnectFor(platform.id)}
        <div
          class="connector-row"
          class:row-disconnected={!connected}
          class:row-needs-reconnect={stale}
        >
          <span
            class="connector-icon"
            style="background-color: {platform.color}"
          ></span>
          <div class="connector-mid">
            <span class="connector-name">{platform.name}</span>
            {#if stale}
              <!-- Deliberately no ConnectorPill and no "Last sync" here: a
                   backfill percentage or a recent sync time next to a dead
                   token reads as reassurance, which is the opposite of what
                   this row means. -->
              <span class="bratrax-status-pill bratrax-status-pill--danger">
                Needs reconnect
              </span>
              <span class="connector-status-warning">
                We can no longer reach your account — reconnect to resume data.
              </span>
            {:else if connected}
              {@const sync = syncBySource.get(platform.id)}
              <ConnectorPill fillPercentage={sync?.progressPct} />
              {#if sync}
                {@const rel = relativeTime(
                  sync.lastSyncAt ?? connectedAt[platform.id],
                )}
                {#if rel}
                  <span class="connector-lastsync">Last sync: {rel}</span>
                {/if}
              {/if}
            {:else}
              <span class="connector-status-muted">Not connected</span>
            {/if}
          </div>
          <div class="connector-actions">
            {#if stale}
              <!-- Single action by design. "View accounts" would show the
                   accounts of a connection that no longer works, and
                   "Disconnect" is just the slow path to the same place
                   Reconnect goes through anyway. -->
              <button
                on:click={() => handleReconnect(platform)}
                disabled={loading === platform.id}
                class="btn-bratrax btn-primary btn-compact"
              >
                {loading === platform.id ? "Reconnecting…" : "Reconnect →"}
              </button>
            {:else if connected}
              <button
                on:click={() =>
                  platform.type === "snippet_install"
                    ? openConnectedBuilderModal()
                    : handleView(platform)}
                disabled={loading === platform.id}
                class="btn-bratrax btn-neutral btn-compact"
              >
                {platform.type === "snippet_install"
                  ? "View snippet"
                  : "View accounts"}
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
  onClose={() => {
    showAccountModal = false;
  }}
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
        <p
          class="mb-3 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted"
        >
          Connected on: {new Date(infoModalConnectedAt).toLocaleString()}
        </p>
      {/if}

      {#if infoModalShopify}
        <div
          class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-acid/70"
        >
          Store
        </div>
        <ul class="flex flex-col gap-1.5">
          {#if infoModalShopify.shopName}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div
                class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
              >
                Name
              </div>
              <div
                class="font-mono text-xs font-bold text-bratrax-text-headline truncate"
              >
                {infoModalShopify.shopName}
              </div>
            </li>
          {/if}
          {#if infoModalShopify.shop}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div
                class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
              >
                Domain
              </div>
              <div
                class="font-mono text-xs font-bold text-bratrax-text-headline truncate"
              >
                {infoModalShopify.shop}
              </div>
            </li>
          {/if}
          {#if infoModalShopify.currency}
            <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
              <div
                class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
              >
                Currency
              </div>
              <div
                class="font-mono text-xs font-bold text-bratrax-text-headline truncate"
              >
                {infoModalShopify.currency}
              </div>
            </li>
          {/if}
          {#if infoModalBackfill}
            <li
              class="border border-bratrax-border border-l-[3px] border-l-bratrax-acid bg-bratrax-bg px-3 py-2"
            >
              <div
                class="font-mono text-[10px] uppercase tracking-wider text-bratrax-text-muted"
              >
                Backfill
              </div>
              <div
                class="font-mono text-xs font-bold text-bratrax-text-headline"
              >
                {backfillLabel(infoModalBackfill.progressPct)}
              </div>
              {#if infoModalBackfill.lastSyncAt}
                <div class="mt-1 font-mono text-[10px] text-bratrax-text-muted">
                  Last sync: {new Date(
                    infoModalBackfill.lastSyncAt,
                  ).toLocaleString()}
                </div>
              {/if}
            </li>
          {/if}
        </ul>
      {:else}
        <div
          class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[1.5px] text-bratrax-acid/70"
        >
          Accounts ({infoModalAccounts.length})
        </div>

        {#if infoModalAccounts.length === 0}
          <p class="font-mono text-xs text-bratrax-text-muted">
            No accounts on file.
          </p>
        {:else}
          <ul class="flex max-h-80 flex-col gap-1.5 overflow-y-auto">
            {#each infoModalAccounts as acct}
              <li class="border border-bratrax-border bg-bratrax-bg px-3 py-2">
                <div
                  class="font-mono text-xs font-bold text-bratrax-text-headline truncate"
                >
                  {acct.name}
                </div>
                <div
                  class="font-mono text-[10px] text-bratrax-text-muted truncate"
                >
                  {acct.id}
                </div>
                {#if acct.progressPct !== undefined}
                  <div
                    class="mt-1.5 font-mono text-[11px] text-bratrax-text-body"
                  >
                    {backfillLabel(acct.progressPct)}
                  </div>
                {/if}
                {#if acct.lastSyncAt}
                  <div class="font-mono text-[10px] text-bratrax-text-muted">
                    Last sync: {new Date(acct.lastSyncAt).toLocaleString()}
                  </div>
                {/if}
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

      <div
        class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-tomato/80"
      >
        Confirm disconnect
      </div>
      <h2 class="text-lg font-black text-bratrax-text-headline">
        Disconnect {confirmPlatform.name}?
      </h2>
      <p class="mt-3 text-sm font-light text-bratrax-text-body">
        This removes all stored credentials and account selections for {confirmPlatform.name}
        from your workspace. You can reconnect at any time.
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

{#if showAmazonRegionModal}
  <AmazonRegionModal
    on:select={handleAmazonRegionPicked}
    on:close={() => (showAmazonRegionModal = false)}
  />
{/if}

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

<BloomreachCredentialModal
  open={showBloomreachModal}
  loading={bloomreachModalLoading}
  error={bloomreachModalError}
  onSubmit={handleBloomreachCredentials}
  onClose={() => {
    showBloomreachModal = false;
    bloomreachModalError = "";
  }}
/>

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
    top: 0;
    left: 0;
    right: 0;
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

  /* Dead token. Stronger tint than .row-disconnected plus a danger edge — a
     never-connected connector is a to-do, this one is actively losing data.
     Both classes can be present in theory; this rule follows so it wins. */
  .row-needs-reconnect {
    background: rgba(230, 50, 38, 0.06);
    box-shadow: inset 2px 0 0 0 var(--bratrax-tomato);
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
    color: #0a0a0a;
  }

  .connector-status-muted {
    font-family: "Space Mono", monospace;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  .connector-lastsync {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.4px;
    color: var(--color-text-muted);
  }

  /* Explanatory line under the "Needs reconnect" pill. Sentence case and not
     uppercase-tracked, unlike the status labels above — it's a sentence to
     read, not a status to scan. */
  .connector-status-warning {
    font-family: "Space Mono", monospace;
    font-size: 11px;
    letter-spacing: 0.4px;
    color: var(--bratrax-tomato);
  }

  .connector-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
</style>
