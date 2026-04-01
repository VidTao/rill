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
        // Brief delay so user sees "Dashboards ready ✓" before redirect
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

<div class="flex h-screen w-screen items-center justify-center bg-surface">
  <div
    class="w-full max-w-lg rounded-lg border border-border bg-white p-8 shadow-sm"
  >
    <div class="mb-6 text-center">
      <h1 class="text-xl font-semibold text-fg-primary">
        Setting up your analytics
      </h1>
      <p class="mt-1 text-sm text-fg-secondary">
        This usually takes a few minutes. You can close this tab and come back
        — we'll keep working in the background.
      </p>
    </div>

    {#if error}
      <div
        class="mb-4 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
      >
        {error}
      </div>
    {/if}

    <div class="flex flex-col gap-3">
      {#each steps as step}
        {@const state = status ? step.check(status) : "pending"}
        <div class="flex items-center gap-3">
          {#if state === "done"}
            <div
              class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-green-100"
            >
              <svg
                class="h-4 w-4 text-green-600"
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
            </div>
          {:else if state === "running"}
            <div
              class="flex h-6 w-6 flex-shrink-0 items-center justify-center"
            >
              <div
                class="h-5 w-5 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600"
              ></div>
            </div>
          {:else if state === "error"}
            <div
              class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-red-100"
            >
              <svg
                class="h-4 w-4 text-red-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
          {:else}
            <div
              class="h-6 w-6 flex-shrink-0 rounded-full border-2 border-gray-200"
            ></div>
          {/if}

          <span
            class="text-sm {state === 'done'
              ? 'text-fg-primary'
              : state === 'running'
                ? 'font-medium text-fg-primary'
                : 'text-fg-secondary'}"
          >
            {step.label}
          </span>
        </div>
      {/each}
    </div>

    <!-- Tap-level extraction details -->
    {#if status && getTapDetails(status).length > 0}
      <div class="mt-6 border-t border-border pt-4">
        <p class="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-secondary">
          Data sync progress
        </p>
        <div class="flex flex-col gap-1.5">
          {#each getTapDetails(status) as tap}
            <div class="flex items-center justify-between text-xs">
              <span class="capitalize text-fg-secondary">{tap.name}</span>
              <span class="font-mono text-fg-primary">
                {tap.rows.toLocaleString()} rows
                {#if tap.status === "running"}
                  <span class="text-indigo-500">syncing</span>
                {:else if tap.status === "done"}
                  <span class="text-green-600">done</span>
                {/if}
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>
