<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { get } from "svelte/store";
  import { goto } from "$app/navigation";
  import { onboardStatus, onboardMe } from "$lib/bratrax/onboarding/api";
  import type { OnboardStatus } from "$lib/bratrax/onboarding/api";
  import {
    runtimeServiceListResources,
    V1ReconcileStatus,
    type V1Resource,
    type V1ResourceName,
  } from "@rilldata/web-common/runtime-client";
  import { runtime } from "@rilldata/web-common/runtime-client/runtime-store";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors";

  type VerifyState = "pending" | "running" | "done" | "error";

  let status: OnboardStatus | null = null;
  let error = "";
  let pollInterval: ReturnType<typeof setInterval> | null = null;
  let verifyInterval: ReturnType<typeof setInterval> | null = null;
  let clientId = "";
  let verifyState: VerifyState = "pending";
  let redirectTimer: ReturnType<typeof setTimeout> | null = null;

  // ---------------------------------------------------------------------------
  // Step display configuration
  // ---------------------------------------------------------------------------
  interface StepDisplay {
    label: string;
    check: (s: OnboardStatus) => "done" | "running" | "pending" | "error";
  }

  const steps: StepDisplay[] = [
    {
      label: "Shopify store connected",
      check: (s) => {
        const platforms = s.connected_platforms || [];
        return platforms.some(
          (p) => typeof p === "object" && p.platform === "shopify",
        )
          ? "done"
          : "pending";
      },
    },
    {
      label: "Ad platforms connected",
      check: (s) => {
        const platforms = s.connected_platforms || [];
        const adIds = new Set([
          "facebook_ads",
          "google_ads",
          "tiktok_ads",
          "pinterest_ads",
          "amazon_ads",
        ]);
        return platforms.some(
          (p) => typeof p === "object" && adIds.has(p.platform),
        )
          ? "done"
          : "pending";
      },
    },
    {
      label: "Building your data pipeline",
      check: (s) => {
        if (s.compile_status === "done") return "done";
        if (s.compile_status === "running") return "running";
        if (s.step === "compiling") return "running";
        if (s.step === "activating") return "running";
        return "pending";
      },
    },
    {
      label: "Deploying analytics tables",
      check: (s) => {
        if (s.deploy_status === "done") return "done";
        if (s.deploy_status === "running") return "running";
        if (s.step === "deploying") return "running";
        return "pending";
      },
    },
    {
      label: "Pulling your data",
      check: (s) => {
        const ext = s.extraction_status || {};
        if (s.step === "ready") return "done";
        if (s.step === "extracting") return "running";
        if (ext.status === "running") return "running";
        if (ext.status === "done") return "done";
        return "pending";
      },
    },
    {
      label: "Dashboards ready",
      check: (s) => {
        if (s.step === "ready") return "done";
        return "pending";
      },
    },
    {
      label: "Verifying dashboards",
      check: (s) => {
        if (s.step !== "ready") return "pending";
        return verifyState;
      },
    },
  ];

  // ---------------------------------------------------------------------------
  // Extraction tap details
  // ---------------------------------------------------------------------------
  interface TapProgress {
    label: string;
    ready: boolean;
    failed: boolean;
  }

  // Pretty-print "shopify/2qbgfb-3y.myshopify.com" -> "Shopify (2qbgfb-3y)".
  // Falls back to the raw key if it doesn't match the source/account pattern.
  function formatTapLabel(key: string): string {
    const slash = key.indexOf("/");
    if (slash === -1) {
      return key.replace("tap-", "").replace(/_/g, " ");
    }
    const source = key.slice(0, slash);
    const account = key.slice(slash + 1);
    const sourceLabel = source.replace(/_/g, " ");
    // Trim "myshopify.com" suffix etc. so the account is short and readable.
    const accountShort = account.replace(/\.myshopify\.com$/i, "").slice(0, 24);
    return accountShort ? `${sourceLabel} (${accountShort})` : sourceLabel;
  }

  function getTapDetails(s: OnboardStatus): TapProgress[] {
    const ext = s.extraction_status || {};
    const taps = (ext as Record<string, unknown>).taps;
    if (!taps || typeof taps !== "object") return [];
    return Object.entries(
      taps as Record<string, { ready?: boolean; status?: string }>,
    ).map(([name, info]) => ({
      label: formatTapLabel(name),
      ready: info?.ready === true,
      failed: info?.status === "failed",
    }));
  }

  // ---------------------------------------------------------------------------
  // Polling
  // ---------------------------------------------------------------------------
  async function poll() {
    if (!clientId) {
      error = "No client ID found. Please start over from signup.";
      return;
    }

    try {
      status = await onboardStatus(clientId);

      if (status.step === "ready") {
        if (pollInterval) clearInterval(pollInterval);
        pollInterval = null;
        startVerification();
      }

      if (status.step === "error") {
        error =
          status.error_message || "Something went wrong. Please try again.";
        if (pollInterval) clearInterval(pollInterval);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      if (pollInterval) clearInterval(pollInterval);
    }
  }

  // ---------------------------------------------------------------------------
  // Rill resource verification
  //
  // After the backend marks activation as "ready", the Rill runtime still needs
  // to finish parsing the generated project YAML and reconciling each resource
  // (connectors, sources, models, metrics views, dashboards). Redirecting to a
  // canvas before its upstream metrics view is idle produces empty/broken
  // charts. We poll ListResources until every resource transitively referenced
  // by a Canvas/Explore dashboard is IDLE with no reconcileError.
  // ---------------------------------------------------------------------------
  function refKey(ref: V1ResourceName | undefined): string {
    return `${ref?.kind ?? ""}/${ref?.name ?? ""}`;
  }

  function collectRequired(resources: V1Resource[]): Set<string> {
    const byKey = new Map<string, V1Resource>();
    for (const r of resources) {
      byKey.set(refKey(r.meta?.name), r);
    }

    const required = new Set<string>();
    const queue: string[] = [];
    for (const r of resources) {
      const kind = r.meta?.name?.kind;
      if (kind === ResourceKind.Canvas || kind === ResourceKind.Explore) {
        const k = refKey(r.meta?.name);
        required.add(k);
        queue.push(k);
      }
    }

    while (queue.length > 0) {
      const k = queue.shift()!;
      const res = byKey.get(k);
      for (const ref of res?.meta?.refs ?? []) {
        const rk = refKey(ref);
        if (!required.has(rk)) {
          required.add(rk);
          queue.push(rk);
        }
      }
    }

    return required;
  }

  async function verify() {
    try {
      const instanceId = get(runtime).instanceId;
      const resp = await runtimeServiceListResources(instanceId);
      const resources = resp.resources ?? [];

      const required = collectRequired(resources);

      // No dashboards in the project yet — parser hasn't caught up; keep polling.
      const hasDashboard = resources.some((r) => {
        const kind = r.meta?.name?.kind;
        return kind === ResourceKind.Canvas || kind === ResourceKind.Explore;
      });
      if (!hasDashboard) {
        verifyState = "running";
        return;
      }

      const failed: string[] = [];
      let anyPending = false;

      // A resource is treated as failed ONLY when it has settled (reconcile
      // status = IDLE) AND still carries a reconcileError. Mid-reconcile
      // errors are transient: Rill cancels and retries a resource whenever an
      // upstream's state version changes ("context canceled"), and those
      // brief error windows would otherwise trigger a permanent "Failed to
      // build" message and stop the polling cycle — even though the
      // reconciler self-heals seconds later.
      for (const r of resources) {
        const k = refKey(r.meta?.name);
        if (!required.has(k)) continue;
        const isIdle =
          r.meta?.reconcileStatus === V1ReconcileStatus.RECONCILE_STATUS_IDLE;
        if (isIdle && r.meta?.reconcileError) {
          failed.push(r.meta?.name?.name ?? k);
          continue;
        }
        if (!isIdle) {
          anyPending = true;
        }
      }

      if (failed.length > 0) {
        verifyState = "error";
        error = `Failed to build: ${failed.join(", ")}`;
        if (verifyInterval) clearInterval(verifyInterval);
        verifyInterval = null;
        return;
      }

      if (anyPending) {
        verifyState = "running";
        return;
      }

      verifyState = "done";
      if (verifyInterval) clearInterval(verifyInterval);
      verifyInterval = null;

      // Pick the first Canvas/Explore dashboard's file path and route to the
      // file view, mirroring the post-init redirect in the root +layout.ts.
      const firstDashboardPath = resources
        .filter((r) => {
          const kind = r.meta?.name?.kind;
          return kind === ResourceKind.Canvas || kind === ResourceKind.Explore;
        })
        .flatMap((r) => r.meta?.filePaths ?? [])
        .find((p) => p.startsWith("/dashboards/"));
      const target = firstDashboardPath ? `/files${firstDashboardPath}` : "/";

      // Brief delay so user sees the final checkmark before redirect
      redirectTimer = setTimeout(() => goto(target), 1500);
    } catch (e) {
      // Swallow transient errors — the runtime may be momentarily unreachable
      // as it restarts to pick up newly-deployed project files. Keep polling.
      verifyState = "running";
      console.warn("Rill resource verification failed, retrying:", e);
    }
  }

  function startVerification() {
    verifyState = "running";
    verify();
    verifyInterval = setInterval(verify, 2000);
  }

  onMount(async () => {
    // Fetch client_id from backend (sessionStorage may be empty)
    const me = await onboardMe();
    if (me?.client_id) {
      clientId = me.client_id;
    }
    poll();
    pollInterval = setInterval(poll, 5000);
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
    if (verifyInterval) clearInterval(verifyInterval);
    if (redirectTimer) clearTimeout(redirectTimer);
  });
</script>

<div class="loading-page flex h-screen w-screen items-center justify-center">
  <div class="halftone-bg"></div>

  <div class="relative w-full max-w-lg border border-bratrax-border bg-bratrax-surface p-8">
    <div class="absolute left-0 right-0 top-0 h-1 bg-bratrax-acid"></div>

    <div class="mb-6 text-center">
      <div class="mb-2 font-mono text-[10px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
        INITIALIZING
      </div>
      <h1 class="text-2xl font-black text-bratrax-text-headline">
        Setting up your analytics
      </h1>
      <p class="mt-2 text-sm font-light text-bratrax-text-body">
        This usually takes a few minutes. You can close this tab and come back
        — we'll keep working in the background.
      </p>
    </div>

    {#if error}
      <div class="mb-4 border border-bratrax-tomato/30 bg-bratrax-tomato/10 px-3 py-2 font-mono text-xs text-bratrax-tomato">
        {error}
      </div>
    {/if}

    <div class="flex flex-col gap-3">
      {#each steps as step}
        {@const state = status ? step.check(status) : "pending"}
        <div class="flex items-center gap-3">
          {#if state === "done"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center bg-bratrax-acid/20">
              <svg class="h-4 w-4 text-bratrax-acid" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
          {:else if state === "running"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center">
              <div class="h-5 w-5 animate-spin border-2 border-bratrax-border border-t-bratrax-acid"></div>
            </div>
          {:else if state === "error"}
            <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center bg-bratrax-tomato/20">
              <svg class="h-4 w-4 text-bratrax-tomato" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>
          {:else}
            <div class="h-6 w-6 flex-shrink-0 border-2 border-bratrax-border"></div>
          {/if}

          <span
            class="text-sm {state === 'done'
              ? 'text-bratrax-acid'
              : state === 'running'
                ? 'font-medium text-bratrax-text-primary'
                : 'text-bratrax-text-muted'}"
          >
            {step.label}
          </span>
        </div>
      {/each}
    </div>

    <!-- Tap-level extraction details -->
    {#if status && getTapDetails(status).length > 0}
      <div class="mt-6 border-t border-bratrax-border pt-4">
        <p class="mb-2 font-mono text-[11px] font-bold uppercase tracking-[2px] text-bratrax-acid/70">
          Data sync progress
        </p>
        <div class="flex flex-col gap-1.5">
          {#each getTapDetails(status) as tap}
            <div class="flex items-center justify-between text-xs">
              <span class="capitalize text-bratrax-text-body">{tap.label}</span>
              <span class="font-mono">
                {#if tap.failed}
                  <span class="text-red-500" title="Extract reported an error">✗ error</span>
                {:else if tap.ready}
                  <span class="text-bratrax-acid">✓ ready</span>
                {:else}
                  <span class="text-bratrax-text-body/60">loading…</span>
                {/if}
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .loading-page {
    background-color: #0A0A0A;
    position: relative;
    overflow: hidden;
  }

  .halftone-bg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background-image: radial-gradient(
      rgba(212, 255, 0, 0.06) 1.5px,
      transparent 1.5px
    );
    background-size: 8px 8px;
  }
</style>
