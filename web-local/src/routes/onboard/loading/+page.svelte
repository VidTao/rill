<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { goto } from "$app/navigation";
  import { onboardStatus, onboardMe } from "$lib/bratrax/onboarding/api";
  import type { OnboardStatus } from "$lib/bratrax/onboarding/api";

  let status: OnboardStatus | null = null;
  let error = "";
  let pollInterval: ReturnType<typeof setInterval> | null = null;
  let clientId = "";

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
  ];

  // ---------------------------------------------------------------------------
  // Extraction tap details
  // ---------------------------------------------------------------------------
  interface TapProgress {
    name: string;
    rows: number;
    status: string;
  }

  function getTapDetails(s: OnboardStatus): TapProgress[] {
    const ext = s.extraction_status || {};
    const taps = (ext as Record<string, unknown>).taps;
    if (!taps || typeof taps !== "object") return [];
    return Object.entries(taps as Record<string, { rows?: number; status?: string }>).map(
      ([name, info]) => ({
        name: name.replace("tap-", "").replace(/_/g, " "),
        rows: info?.rows ?? 0,
        status: info?.status ?? "pending",
      }),
    );
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
        // Brief delay so user sees "Dashboards ready" before redirect
        setTimeout(() => goto("/canvas/performance_overview"), 1500);
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
        Setting up your <span class="font-serif italic text-bratrax-acid">analytics</span>
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
              <span class="capitalize text-bratrax-text-body">{tap.name}</span>
              <span class="font-mono text-bratrax-text-primary">
                {tap.rows.toLocaleString()} rows
                {#if tap.status === "running"}
                  <span class="text-bratrax-acid">syncing</span>
                {:else if tap.status === "done"}
                  <span class="text-bratrax-acid">done</span>
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
