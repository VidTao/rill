<script lang="ts">
  import { onDestroy } from "svelte";
  import { createEventDispatcher } from "svelte";
  import {
    wooPluginDownloadUrl,
    verifyWooInstall,
    type WooVerifyInstallResult,
  } from "./api";

  // The active client to verify events for. Required.
  export let clientId: string;

  const dispatch = createEventDispatcher<{
    verified: WooVerifyInstallResult;
  }>();

  type Status = "idle" | "checking" | "waiting" | "connected" | "error";
  let status: Status = "idle";
  let result: WooVerifyInstallResult | null = null;
  let error = "";
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let pollsLeft = 0;

  const EXPECTED = [
    "page_viewed",
    "product_viewed",
    "product_added_to_cart",
    "checkout_started",
    "checkout_completed",
  ];

  const downloadUrl = wooPluginDownloadUrl();

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  async function checkOnce(): Promise<void> {
    if (!clientId) {
      error = "No active client to verify.";
      status = "error";
      return;
    }
    try {
      result = await verifyWooInstall(clientId);
      error = "";
      if (result.connected) {
        status = "connected";
        stopPolling();
        dispatch("verified", result);
      } else if (status !== "waiting") {
        status = "idle";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Verification failed";
      status = "error";
      stopPolling();
    }
  }

  // Arm a short auto-poll window (the merchant browses their store while we watch).
  function startVerifying(): void {
    status = "waiting";
    pollsLeft = 24; // ~2 min at 5s
    void checkOnce();
    stopPolling();
    pollTimer = setInterval(() => {
      pollsLeft -= 1;
      if (pollsLeft <= 0) {
        stopPolling();
        if (status === "waiting") status = "idle";
        return;
      }
      void checkOnce();
    }, 5000);
  }

  onDestroy(stopPolling);
</script>

<div class="border border-bratrax-border bg-bratrax-surface px-5 py-4">
  <p
    class="mb-1 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-acid"
  >
    WooCommerce tracking
  </p>
  <h3 class="text-base font-black text-bratrax-text-headline">
    Install the tracking plugin
  </h3>
  <p class="mt-1 text-sm font-light text-bratrax-text-body">
    Connecting your store lets us read orders. The plugin adds on-site tracking
    (page views, add-to-cart, checkout) so we can attribute those orders to the
    right channels. Orders stay the source of truth — events are evidence only.
  </p>

  <ol class="mt-4 flex flex-col gap-2 text-sm text-bratrax-text-body">
    <li class="flex items-start gap-2">
      <span class="font-mono text-bratrax-acid">1.</span>
      <a
        href={downloadUrl}
        class="font-semibold text-bratrax-acid underline underline-offset-2"
        download
      >
        Download the plugin (.zip)
      </a>
    </li>
    <li class="flex items-start gap-2">
      <span class="font-mono text-bratrax-acid">2.</span>
      <span
        >In WordPress admin: <strong class="text-bratrax-text-headline"
          >Plugins → Add New → Upload Plugin</strong
        >, choose the zip, install and activate.</span
      >
    </li>
    <li class="flex items-start gap-2">
      <span class="font-mono text-bratrax-acid">3.</span>
      <span
        >That's it — the plugin links to your workspace automatically. No setup,
        no IDs to copy. (Status shows under <strong
          class="text-bratrax-text-headline">WooCommerce → Bratrax</strong
        >.)</span
      >
    </li>
    <li class="flex items-start gap-2">
      <span class="font-mono text-bratrax-acid">4.</span>
      <span>Visit a product and add it to the cart, then verify below.</span>
    </li>
  </ol>

  <div class="mt-4 flex flex-wrap items-center gap-3">
    <a
      href={downloadUrl}
      download
      class="bg-bratrax-acid px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-[3px] text-bratrax-bg transition-opacity hover:opacity-90"
    >
      Download plugin →
    </a>
    <button
      type="button"
      class="border border-bratrax-border px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-text-muted transition-colors hover:border-bratrax-acid hover:text-bratrax-acid disabled:opacity-40"
      on:click={startVerifying}
      disabled={status === "waiting" || status === "connected"}
    >
      {#if status === "waiting"}
        Watching for events…
      {:else if status === "connected"}
        Connected
      {:else}
        Verify install
      {/if}
    </button>

    {#if status === "connected"}
      <span
        class="font-mono text-[11px] font-bold uppercase tracking-wider text-bratrax-acid"
      >
        ✓ Receiving events ({result?.events_seen})
      </span>
    {:else if status === "waiting"}
      <span class="text-sm text-bratrax-text-muted">
        Browse your store in another tab — events appear within a minute.
      </span>
    {/if}
  </div>

  {#if error}
    <p class="mt-3 font-mono text-xs text-bratrax-tomato">{error}</p>
  {/if}

  {#if result}
    <div
      class="mt-4 border border-bratrax-border bg-bratrax-bg px-3 py-3 text-sm"
    >
      <p
        class="mb-1 font-mono text-[11px] uppercase tracking-wider text-bratrax-text-muted"
      >
        Events seen (last {result.lookback_minutes} min): {result.events_seen}
      </p>
      <ul class="grid grid-cols-1 gap-0.5 sm:grid-cols-2">
        {#each EXPECTED as ev}
          <li class="flex items-center gap-2">
            <span
              class={result.by_type[ev]
                ? "text-bratrax-acid"
                : "text-bratrax-text-muted"}
            >
              {result.by_type[ev] ? "✓" : "—"}
            </span>
            <span class="font-mono text-xs text-bratrax-text-body">{ev}</span>
            {#if result.by_type[ev]}
              <span class="text-xs text-bratrax-text-muted"
                >({result.by_type[ev]})</span
              >
            {/if}
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
